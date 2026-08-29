package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"

	"github.com/kataras/basicauth"
)

const (
	StatusTodo       = "todo"
	StatusInProgress = "inprogress"
	StatusDone       = "done"

	bucketCards = "cards"
)

type Card struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Subtasks    string `json:"subtasks"`
	Status      string `json:"status"`
	CardOrder   int    `json:"card_order"`
}

var db *bolt.DB
var tmpl *template.Template

type OrderUpdatePayload struct {
	Status string `json:"status"`
	Order  []int  `json:"order"`
}

func main() {
	dataPath := getEnv("DATA_PATH", "./data/app.db")
	os.MkdirAll("./data", 0755)

	var err error
	db, err = bolt.Open(dataPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketCards))
		return err
	})

	users := make(map[string]string)
	data, err := os.ReadFile("users.json")
	if err == nil {
		json.Unmarshal(data, &users)
	}
	auth := basicauth.Default(users)

	funcMap := template.FuncMap{
		"split": func(s, sep string) []string {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil
			}
			return strings.Split(s, sep)
		},
		"trim": strings.TrimSpace,
	}
	tmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/card", createCardHandler)
	http.HandleFunc("/card/", cardRouter)
	http.HandleFunc("/card/order", updateOrderHandler)

	serverPort := getEnv("SERVER_PORT", "17808")
	log.Println("Server started on :" + serverPort)

	listener, err := newListener(":" + serverPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.Serve(listener, auth(http.DefaultServeMux)))
}

func newListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func cardKey(id int) []byte {
	return []byte(strconv.Itoa(id))
}

func getCardByID(id int) (*Card, error) {
	var card Card
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		v := b.Get(cardKey(id))
		if v == nil {
			return bolt.ErrBucketNotFound
		}
		return json.Unmarshal(v, &card)
	})
	if err == bolt.ErrBucketNotFound {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cardsByStatus := map[string][]Card{
		StatusTodo:       {},
		StatusInProgress: {},
		StatusDone:       {},
	}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var card Card
			if err := json.Unmarshal(v, &card); err != nil {
				continue
			}
			cardsByStatus[card.Status] = append(cardsByStatus[card.Status], card)
		}
		return nil
	})
	for _, cards := range cardsByStatus {
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].CardOrder < cards[j].CardOrder
		})
	}
	if err := tmpl.ExecuteTemplate(w, "index.html", cardsByStatus); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func createCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/card" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	title := r.FormValue("title")
	description := r.FormValue("description")
	subtasks := r.FormValue("subtasks")
	status := r.FormValue("status")

	if status != StatusTodo && status != StatusInProgress && status != StatusDone {
		status = StatusTodo
	}

	if strings.TrimSpace(title) == "" && strings.TrimSpace(description) == "" && strings.TrimSpace(subtasks) == "" {
		http.Error(w, "Empty card not allowed", http.StatusBadRequest)
		return
	}

	var maxOrder int
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var card Card
			if err := json.Unmarshal(v, &card); err != nil {
				continue
			}
			if card.Status == status && card.CardOrder > maxOrder {
				maxOrder = card.CardOrder
			}
		}
		return nil
	})
	maxOrder++

	var newID int
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		id, _ := b.NextSequence()
		newID = int(id)
		card := Card{
			ID:          newID,
			Title:       title,
			Description: description,
			Subtasks:    subtasks,
			Status:      status,
			CardOrder:   maxOrder,
		}
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		return b.Put(cardKey(newID), data)
	})

	card := Card{ID: newID, Title: title, Description: description, Subtasks: subtasks, Status: status, CardOrder: maxOrder}
	if r.Header.Get("HX-Request") != "" {
		if err := tmpl.ExecuteTemplate(w, "card_fragment.html", card); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func cardRouter(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	action := parts[3]
	switch action {
	case "move":
		moveCardHandler(w, r, id)
	case "edit":
		editCardHandler(w, r, id)
	case "update":
		updateCardHandler(w, r, id)
	case "delete":
		deleteCardHandler(w, r, id)
	case "view":
		viewCardHandler(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func moveCardHandler(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	newStatus := r.FormValue("status")
	if newStatus != StatusTodo && newStatus != StatusInProgress && newStatus != StatusDone {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}
	newOrder, err := strconv.Atoi(r.FormValue("order"))
	if err != nil {
		http.Error(w, "Invalid order", http.StatusBadRequest)
		return
	}

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		v := b.Get(cardKey(id))
		if v == nil {
			return bolt.ErrBucketNotFound
		}
		var card Card
		if err := json.Unmarshal(v, &card); err != nil {
			return err
		}
		card.Status = newStatus
		card.CardOrder = newOrder
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		return b.Put(cardKey(id), data)
	})

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		cards := make([]Card, 0)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var card Card
			if err := json.Unmarshal(v, &card); err != nil {
				continue
			}
			if card.Status == newStatus {
				cards = append(cards, card)
			}
		}
		sort.Slice(cards, func(i, j int) bool {
			if cards[i].CardOrder == cards[j].CardOrder {
				return cards[i].ID < cards[j].ID
			}
			return cards[i].CardOrder < cards[j].CardOrder
		})
		for i, card := range cards {
			card.CardOrder = i + 1
			data, err := json.Marshal(card)
			if err != nil {
				continue
			}
			b.Put(cardKey(card.ID), data)
		}
		return nil
	})

	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func editCardHandler(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	card, err := getCardByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "card_edit_fragment.html", card); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func updateCardHandler(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	title := r.FormValue("title")
	description := r.FormValue("description")
	subtasks := r.FormValue("subtasks")

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		v := b.Get(cardKey(id))
		if v == nil {
			return bolt.ErrBucketNotFound
		}
		var card Card
		if err := json.Unmarshal(v, &card); err != nil {
			return err
		}
		card.Title = title
		card.Description = description
		card.Subtasks = subtasks
		data, err := json.Marshal(card)
		if err != nil {
			return err
		}
		return b.Put(cardKey(id), data)
	})

	updated, err := getCardByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "card_fragment.html", updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteCardHandler(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		return b.Delete(cardKey(id))
	})
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func viewCardHandler(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	card, err := getCardByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "card_fragment.html", card); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func updateOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload OrderUpdatePayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketCards))
		for index, cardId := range payload.Order {
			v := b.Get(cardKey(cardId))
			if v == nil {
				continue
			}
			var card Card
			if err := json.Unmarshal(v, &card); err != nil {
				continue
			}
			card.Status = payload.Status
			card.CardOrder = index + 1
			data, err := json.Marshal(card)
			if err != nil {
				continue
			}
			b.Put(cardKey(cardId), data)
		}
		return nil
	})
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
