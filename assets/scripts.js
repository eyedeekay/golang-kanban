function initDarkMode() {
  const darkModeToggle = document.getElementById('darkModeToggle');
  const html = document.documentElement;
  const savedTheme = localStorage.getItem('theme');
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    html.classList.add('dark');
  }
  darkModeToggle.addEventListener('click', () => {
    html.classList.toggle('dark');
    localStorage.setItem('theme', html.classList.contains('dark') ? 'dark' : 'light');
  });
}
function showModal(modalId) {
  document.getElementById(modalId).classList.add('show');
}
function hideModal(modalId) {
  document.getElementById(modalId).classList.remove('show');
}
document.addEventListener('click', function(e) {
  if (e.target.classList.contains('modal')) {
    e.target.classList.remove('show');
  }
});
['cards-todo', 'cards-inprogress', 'cards-done'].forEach(function(id) {
  var container = document.getElementById(id);
  if (container) {
    new Sortable(container, {
      group: 'kanban',
      animation: 150,
      onEnd: function(evt) {
        let destContainer = evt.to;
        let destCardIds = Array.from(destContainer.children)
            .map(child => child.getAttribute('data-id'))
            .filter(id => id !== null);
        let destStatus = destContainer.id.split('-')[1];
        fetch("/card/order", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status: destStatus, order: destCardIds.map(Number) })
        });
        if (evt.from !== evt.to) {
          let originContainer = evt.from;
          let originCardIds = Array.from(originContainer.children)
              .map(child => child.getAttribute('data-id'))
              .filter(id => id !== null);
          let originStatus = originContainer.id.split('-')[1];
          fetch("/card/order", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ status: originStatus, order: originCardIds.map(Number) })
          });
        }
      }
    });
  }
});
function initSubtasks(container) {
  container.addEventListener('click', function(e) {
    let btn = e.target.closest('.remove-subtask-btn');
    if (btn && container.contains(btn)) {
      btn.closest('.subtask-row').remove();
    }
  });
  const form = container.closest('form');
  if (form) {
    form.querySelectorAll('.add-subtask-btn').forEach(function(btn) {
      btn.addEventListener('click', function() {
        const row = document.createElement('div');
        row.className = 'subtask-row flex items-center space-x-2 mb-2';
        row.innerHTML = `<input type="checkbox" class="subtask-complete w-4 h-4 text-blue-600 rounded focus:ring-blue-500"><input type="text" class="subtask-text flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm" placeholder="Subtask description"><button type="button" class="remove-subtask-btn text-red-500 hover:text-red-700 p-1" title="Remove"><i class="bi bi-x text-lg"></i></button>`;
        container.appendChild(row);
      });
    });
  }
}
function prepareSubtasks(form) {
  const container = form.querySelector('.subtasks-container');
  const hidden = form.querySelector('.subtasks-hidden');
  let subtasks = [];
  container.querySelectorAll('.subtask-row').forEach(function(row) {
    const complete = row.querySelector('.subtask-complete').checked ? "1" : "0";
    const text = row.querySelector('.subtask-text').value.trim();
    if (text !== "") {
      subtasks.push(complete + "|" + text);
    }
  });
  hidden.value = subtasks.join("\n");
  return true;
}
document.body.addEventListener("htmx:afterSwap", function(evt) {
  if (evt.detail.target.id === "cards-todo") {
    hideModal('addCardModal');
    var form = document.getElementById('addCardForm');
    if (form) {
      form.reset();
      const container = form.querySelector('.subtasks-container');
      if (container) { container.innerHTML = ""; }
    }
  }
  if (evt.detail.target.id === "editCardModalContent") {
    showModal('editCardModal');
    var container = document.getElementById('editCardModalContent').querySelector('.subtasks-container');
    if (container) { initSubtasks(container); }
  }
  if (evt.detail.target && evt.detail.target.id && evt.detail.target.id.startsWith("card-")) {
    hideModal('editCardModal');
  }
});
document.addEventListener("DOMContentLoaded", function() {
  initDarkMode();
  document.querySelectorAll('.add-subtask-btn').forEach(function(btn) {
    btn.addEventListener('click', function() {
      const container = btn.closest('form').querySelector('.subtasks-container');
      const row = document.createElement('div');
      row.className = 'subtask-row flex items-center space-x-2 mb-2';
      row.innerHTML = `<input type="checkbox" class="subtask-complete w-4 h-4 text-blue-600 rounded focus:ring-blue-500"><input type="text" class="subtask-text flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm" placeholder="Subtask description"><button type="button" class="remove-subtask-btn text-red-500 hover:text-red-700 p-1" title="Remove"><i class="bi bi-x text-lg"></i></button>`;
      container.appendChild(row);
    });
  });
  document.getElementById('addCardForm').addEventListener('submit', function(e) {
    return prepareSubtasks(this);
  });
  document.querySelectorAll('[data-action="show-modal"]').forEach(function(el) {
    el.addEventListener('click', function() { showModal(el.getAttribute('data-modal')); });
  });
  document.querySelectorAll('[data-action="hide-modal"]').forEach(function(el) {
    el.addEventListener('click', function() { hideModal(el.getAttribute('data-modal')); });
  });
  var addForm = document.getElementById('addCardForm');
  if (addForm) {
    var container = addForm.querySelector('.subtasks-container');
    if (container) { initSubtasks(container); }
  }
});
