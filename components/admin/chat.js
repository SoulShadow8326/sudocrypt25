let currentUser = '';
let currentChecksum = '';

async function fetchAllMessages() {
  const resp = await fetch('/api/messages?mode=admin', { credentials: 'same-origin' });
  if (!resp.ok && resp.status !== 304) throw new Error('messages list failed: ' + resp.status);
  if (resp.status === 304) return { checksum: currentChecksum, messages: null };
  return await resp.json();
}

function buildConversationList(allMsgs) {
  const map = new Map();
  (allMsgs || []).forEach(m => {

    const me = window.__adminEmail || '';
    const adminAddress = "admin@sudocrypt.com";

    let other = '';
    if (m.from === adminAddress) other = m.to;
    else if (m.to === adminAddress) other = m.from;
    else if (m.from === me) other = m.to; // legacy rows
    else if (m.to === me) other = m.from; // legacy rows
    else return; // not related to this admin

    if (!other || other === adminAddress) return; 

    const prev = map.get(other);
    const ts = m.created_at || m.CreatedAt || 0;
    if (!prev || ts > prev.ts) map.set(other, { last: m.content, ts });
  });
  const items = Array.from(map.entries()).sort((a,b)=>b[1].ts - a[1].ts);
  const cont = document.getElementById('adminChatList');
  cont.innerHTML = '';
  items.forEach(([email, meta]) => {
    const d = document.createElement('div');
    d.className = 'admin-chat-item' + (email === currentUser ? ' active' : '');
    d.textContent = email;
    d.addEventListener('click', () => { selectConversation(email); });
    cont.appendChild(d);
  });
}

async function fetchThread(user, checksum) {
  const url = '/api/messages?mode=admin&user=' + encodeURIComponent(user) + (checksum ? '&checksum=' + encodeURIComponent(checksum) : '');
  const resp = await fetch(url, { credentials: 'same-origin' });
  if (resp.status === 304) return { checksum, messages: null };
  if (!resp.ok) throw new Error('thread fetch failed: ' + resp.status);
  return await resp.json();
}

function renderThread(user, data) {
  const header = document.getElementById('adminChatHeader');
  header.textContent = user || 'Select a conversation';
  const cont = document.getElementById('adminChatMessages');
  if (!data || !data.messages) return;
  const msgs = data.messages;
  const me = window.__adminEmail || '';
  const adminAddress = "admin@sudocrypt.com";
  
  cont.innerHTML = '';
  msgs.forEach(m => {
    const row = document.createElement('div');
    const isFromAdmin = (m.from === me || m.from === adminAddress);
    row.className = 'msg-row' + (isFromAdmin ? ' msg-me' : '');
    
    const bubble = document.createElement('div');
    bubble.className = 'msg-bubble';
    const content = document.createElement('div');
    content.className = 'msg-content';
    content.textContent = m.content;
    bubble.appendChild(content);
    row.appendChild(bubble);
    cont.appendChild(row);
  });
  cont.scrollTop = cont.scrollHeight;
}

async function selectConversation(user) {
  currentUser = user;
  currentChecksum = '';
  const data = await fetchThread(user, currentChecksum);
  if (data) {
    currentChecksum = data.checksum || currentChecksum;
    renderThread(user, data);
  }
}

async function pollThreadLoop() {
  while (true) {
    try {
      const all = await fetchAllMessages();
      if (all && all.messages) {
        buildConversationList(all.messages || []);
      }
      
      if (currentUser) {
        const data = await fetchThread(currentUser, currentChecksum);
        if (data && data.messages) {
          currentChecksum = data.checksum || currentChecksum;
          renderThread(currentUser, data);
        }
      }
    } catch (e) {
      console.warn('[admin/chat] poll error', e);
    }
    await new Promise(r => setTimeout(r, 2000));
  }
}

async function sendToCurrent() {
  const input = document.getElementById('adminChatInput');
  const content = (input.value || '').trim();
  if (!content || !currentUser) return;
  
  try {
    (function optimisticAppend() {
      const cont = document.getElementById('adminChatMessages');
      const row = document.createElement('div');
      row.className = 'msg-row msg-me';
      const bubble = document.createElement('div');
      bubble.className = 'msg-bubble';
      const text = document.createElement('div');
      text.className = 'msg-content';
      text.textContent = content;
      bubble.appendChild(text);
      row.appendChild(bubble);
      cont.appendChild(row);
      cont.scrollTop = cont.scrollHeight;
    })();

    await fetch('/api/message/send', { 
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' }, 
      credentials: 'same-origin', 
      body: JSON.stringify({ to: currentUser, type: 'message', content }) 
    });
    

    const data = await fetchThread(currentUser, '');
    if (data) {
      currentChecksum = data.checksum || '';
      renderThread(currentUser, data);
    }
    
    input.value = '';
  } catch (e) {
    console.error('Failed to send message:', e);
  }
}

async function bootstrap() {
  try {
    const all = await fetchAllMessages();
    buildConversationList(all.messages || []);
  } catch (e) {
    console.warn('[admin/chat] init list error', e);
  }
  document.getElementById('adminChatSend').addEventListener('click', sendToCurrent);
  document.getElementById('adminChatInput').addEventListener('keydown', function(e){ if (e.key==='Enter') sendToCurrent(); });
  pollThreadLoop();


  function visibilityRefresh() {
    if (currentUser) {
      fetchThread(currentUser, '').then(data => { if (data) { currentChecksum = data.checksum || ''; renderThread(currentUser, data); } });
    }
    fetchAllMessages().then(all => { if (all && all.messages) buildConversationList(all.messages || []); }).catch(()=>{});
  }
  document.addEventListener('visibilitychange', function(){ if (!document.hidden) visibilityRefresh(); });
  window.addEventListener('focus', visibilityRefresh);
}

document.addEventListener('DOMContentLoaded', bootstrap);
