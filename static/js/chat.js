// Chat Page WebSocket logic
if (window.location.pathname.endsWith('/chat')) {
  const ws = new WebSocket((location.protocol==='https:'?'wss':'ws') + '://' + location.host + '/ws');
  const log = document.getElementById('chatLog');
  const inp = document.getElementById('chatInput');
  const btn = document.getElementById('sendBtn');

  ws.onmessage = e => {
    const msg = JSON.parse(e.data);
    const div = document.createElement('div');
    const t = new Date(msg.time).toLocaleTimeString();
    div.textContent = `[${t}] ${msg.user}: ${msg.content}`;
    log.appendChild(div);
    log.scrollTop = log.scrollHeight;
  };

  btn.onclick = () => {
    const text = inp.value.trim();
    if (!text) return;
    ws.send(JSON.stringify({content: text}));
    inp.value = '';
  };

  inp.addEventListener('keypress', e => {
    if (e.key === 'Enter') btn.click();
  });
}
