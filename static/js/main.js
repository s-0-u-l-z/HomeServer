// Sidebar & content animations + Theme toggling
window.addEventListener('DOMContentLoaded', () => {
  setTimeout(() => {
    const sidebar = document.querySelector('.sidebar');
    const mainContent = document.querySelector('.main-content');
    if (sidebar) sidebar.style.transform = 'translateX(0)';
    if (mainContent) mainContent.style.opacity = '1';
  }, 100);

  const btn = document.getElementById('toggle-theme-btn');
  const themes = ['default', 'green', 'white'];
  let cur = localStorage.getItem('theme') || 'default';
  applyTheme(cur);
  if (btn) btn.addEventListener('click', () => {
    let idx = themes.indexOf(cur);
    cur = themes[(idx + 1) % themes.length];
    applyTheme(cur);
    localStorage.setItem('theme', cur);
  });
});

function applyTheme(name) {
  document.body.classList.remove('theme-default','theme-green','theme-white');
  document.body.classList.add('theme-' + name);
}

// HOME PAGE: Recent Uploads & Search & Upload
(function() {
  const uploadsList = document.getElementById('uploadsList');
  const fileInput = document.getElementById('fileInput');
  const uploadBtn = document.getElementById('uploadBtn');
  const searchInput = document.getElementById('searchInput');
  const searchResults = document.getElementById('searchResults');

  if (!uploadsList) return;

  function loadRecentUploads() {
    fetch('/api/listImages')
      .then(res => res.json())
      .then(data => {
        uploadsList.innerHTML = '';
        data.forEach(fn => {
          const li = document.createElement('li');
          li.textContent = fn;
          li.onclick = () => downloadFile(fn);
          uploadsList.appendChild(li);
        });
      });
  }

  if (uploadBtn) {
    uploadBtn.addEventListener('click', () => {
      const file = fileInput.files[0];
      if (!file) return alert('Select file');
      const formData = new FormData();
      formData.append('file', file);
      fetch('/api/upload', { method: 'POST', body: formData })
        .then(() => loadRecentUploads())
        .catch(() => alert('Upload failed.'));
    });
  }

  if (searchInput && searchResults) {
    searchInput.addEventListener('input', () => {
      const q = searchInput.value.trim().toLowerCase();
      if (!q) {
        searchResults.innerHTML = '';
        return;
      }
      fetch(`/api/listImages?q=${encodeURIComponent(q)}`)
        .then(res => res.json())
        .then(data => {
          searchResults.innerHTML = '';
          data.forEach(fn => {
            const img = document.createElement('img');
            img.src = `/uploads/${fn}`;
            img.alt = fn;
            img.classList.add('fade-in');
            img.onclick = () => downloadFile(fn);
            searchResults.appendChild(img);
          });
        });
    });
  }

  function downloadFile(filename) {
    const link = document.createElement('a');
    link.href = `/uploads/${filename}`;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  loadRecentUploads();
})();

// UPLOADS PAGE: Images & Videos sizing fix and playable videos
(function() {
  const uploadFileBtn = document.getElementById('uploadFileBtn');
  const uploadInput   = document.getElementById('uploadInput');
  const imageGrid     = document.getElementById('imageGrid');
  const videoGrid     = document.getElementById('videoGrid');

  if (!(imageGrid || videoGrid)) return;

  if (uploadFileBtn) {
    uploadFileBtn.addEventListener('click', () => {
      const file = uploadInput.files[0];
      if (!file) return alert('Select file');
      const formData = new FormData();
      formData.append('file', file);
      fetch('/api/upload', { method: 'POST', body: formData })
        .then(() => loadUploadsGrid())
        .catch(() => alert('Upload failed.'));
    });
  }

  function loadUploadsGrid() {
    fetch('/api/listImages')
      .then(res => res.json())
      .then(list => {
        if (imageGrid) imageGrid.innerHTML = '';
        if (videoGrid) videoGrid.innerHTML = '';

        list.forEach(fn => {
          const ext = fn.split('.').pop().toLowerCase();

          // IMAGE types
          if (['png','jpg','jpeg','gif','bmp','svg'].includes(ext)) {
            const img = document.createElement('img');
            img.src = `/uploads/${fn}`;
            img.alt = fn;
            img.style.width       = '80px';
            img.style.height      = 'auto';
            img.style.margin      = '10px';
            img.style.cursor      = 'pointer';
            img.style.borderRadius= '8px';
            img.style.boxShadow   = '0 4px 8px rgba(0,0,0,0.5)';
            img.onclick           = () => downloadFile(fn);
            imageGrid.appendChild(img);
          }

          // VIDEO types (only mp4/webm/ogg)
          else if (['mp4','webm','ogg'].includes(ext)) {
            const video = document.createElement('video');
            video.src      = `/uploads/${fn}`;
            video.controls = true;
            video.preload  = 'metadata';
            video.style.width       = '240px';
            video.style.height      = 'auto';
            video.style.objectFit   = 'cover';
            video.style.margin      = '10px';
            video.style.borderRadius= '8px';
            video.style.boxShadow   = '0 4px 8px rgba(0,0,0,0.5)';
            videoGrid.appendChild(video);
          }

          // other types: download link
          else {
            const link = document.createElement('a');
            link.href = `/uploads/${fn}`;
            link.textContent = `Download ${fn}`;
            link.style.display = 'block';
            link.style.margin = '10px';
            videoGrid.appendChild(link);
          }
        });
      });
  }

  function downloadFile(filename) {
    const link = document.createElement('a');
    link.href     = `/uploads/${filename}`;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  loadUploadsGrid();
})();

// STATS PAGE: Dynamic fetch
(function() {
  const statUploads = document.getElementById('statUploads');
  const statMessages= document.getElementById('statMessages');
  if (!statUploads || !statMessages) return;
  fetch('/api/userStats')
    .then(res => res.json())
    .then(d => {
      statUploads.textContent  = d.uploads;
      statMessages.textContent = d.messages;
    });
})();

// ADMIN PAGE
(function() {
  if (!window.location.pathname.endsWith('/admin')) return;
  const cpu         = document.getElementById('cpu');
  const mem         = document.getElementById('mem');
  const disk        = document.getElementById('disk');
  const imageList   = document.getElementById('imageList');
  const messageList = document.getElementById('messageList');

  function updateSystemStats() {
    fetch('/api/systemStats')
      .then(res => res.json())
      .then(d => {
        cpu.textContent  = d.CPU;
        mem.textContent  = (d.MemoryUsed/1e6).toFixed(1) + '/' + (d.MemoryTotal/1e6).toFixed(1) + ' MB';
        disk.textContent = (d.DiskUsed/1e6).toFixed(1) + '/' + (d.DiskTotal/1e6).toFixed(1) + ' MB';
      });
  }

  function loadAdminImages() {
    fetch('/api/listImages')
      .then(res => res.json())
      .then(list => {
        imageList.innerHTML = '';
        list.forEach(fn => {
          const li = document.createElement('li');
          li.textContent = fn + ' ';
          const btn = document.createElement('button');
          btn.textContent = 'Delete';
          btn.onclick = () => {
            fetch('/api/deleteImage', {
              method: 'POST',
              headers: {'Content-Type':'application/x-www-form-urlencoded'},
              body: `image=${encodeURIComponent(fn)}`
            }).then(loadAdminImages);
          };
          li.appendChild(btn);
          imageList.appendChild(li);
        });
      });
  }

  function loadAdminMessages() {
    fetch('/api/listMessages')
      .then(res => res.json())
      .then(list => {
        messageList.innerHTML = '';
        list.forEach(m => {
          const li = document.createElement('li');
          li.textContent = `[${new Date(m.time).toLocaleTimeString()}] ${m.user}: ${m.content} `;
          const btn = document.createElement('button');
          btn.textContent = 'Delete';
          btn.onclick = () => {
            fetch('/api/deleteMessage', {
              method: 'POST',
              headers: {'Content-Type':'application/x-www-form-urlencoded'},
              body: `id=${m.id}`
            }).then(loadAdminMessages);
          };
          li.appendChild(btn);
          messageList.appendChild(li);
        });
      });
  }

  setInterval(updateSystemStats, 2000);
  updateSystemStats();
  loadAdminImages();
  loadAdminMessages();
})();

// GREETING FUNCTION (Home only)
function setGreeting(username) {
  const now = new Date();
  let greeting = now.getHours() < 12   ? 'Good morning, '
               : now.getHours() < 17   ? 'Good afternoon, '
               :                         'Good evening, ';
  const g = document.getElementById('greeting');
  if (g) g.textContent = greeting + username;
}
