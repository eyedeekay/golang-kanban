## Simple Kanban Board
A no-nonsense, lightweight Kanban board built with Golang and HTMX. I couldn't find a reasonable self-hosted Kanban board that wasn't using some JavaScript monstrosity like Node or Next.js, I don't need the next [9.1 CVE](https://github.com/advisories/GHSA-f82v-jwr5-mffw) on my server — so I decided to build my own. This project is my sweet little solution to manage tasks simply while learning and sharing a project with the community.

### Overview
This project provides a clean, minimalistic Kanban board with the following technologies:

Backend: Golang

Frontend: HTMX, Bootstrap 5, and SortableJS

Database: BoltDB (embedded, file-based)

It's designed to be simple to deploy and maintain without the extra bloat of modern JavaScript frameworks or external database servers.

### Features
- Minimalistic Design: A straightforward Kanban board to manage your tasks.
- Dynamic Interactivity: Partial page updates using HTMX for a smooth user experience.
- Drag-and-Drop: Rearrange cards effortlessly using SortableJS.
- No external database required: Uses BoltDB embedded key-value store.

### How it looks
![Screenshot](assets/Screenshot_v1.0.0.png "Screenshot")

Note: `users.json` is an example auth config file and must be replaced with hashed credentials for production use.

Note: TLS must be terminated by a reverse proxy (e.g., nginx, traefik) or configured externally. Direct TLS is not enabled by default.

### Using Docker Compose
A sample docker-compose.yml is provided, just use `docker-compose up --build`
This command will build and run the Kanban service.

### Using the Pre-built Docker Image
Alternatively, you can pull the pre-built Docker image from GitHub Container Registry:

``` bash
docker pull ghcr.io/nicolashaas/golang-kanban:latest
docker run -p 17808:17808 -v kanban_data:/data ghcr.io/nicolashaas/golang-kanban:latest
```

### Prerequisites
No external database required. The application uses BoltDB, an embedded file-based database.

### Environment Variables:
``` bash
DATA_PATH=./data/app.db
SERVER_PORT=17808
```

The BoltDB database file is stored at the path specified by `DATA_PATH`. Ensure the directory is writable and backed up as needed.

#### Todo's
If I feel like it I might work on some of these things:
- [ ] darkmode
- [ ] remove/add/edit collums
- [ ] make it pretty
- [ ] tls
- [ ] oidc
- [ ] ...

#### Contributing
Contributions are welcome! If you have ideas, bug fixes, or enhancements, feel free to fork the repository, open an issue, or submit a pull request.

#### License
This project is open-sourced under the MIT License.
