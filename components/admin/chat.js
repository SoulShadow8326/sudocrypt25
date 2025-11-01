let currentUser = '';
let currentChecksum = '';
const lastRenderedMaxID = {};
let adminListChecksum = '';

async function fetchAllMessages() {
  const url = adminListChecksum ? ('/api/messages?mode=admin&checksum=' + encodeURIComponent(adminListChecksum)) : '/api/messages?mode=admin';
  const resp = await fetch(url, { credentials: 'same-origin' });
  if (!resp.ok && resp.status !== 304) throw new Error('messages list failed: ' + resp.status);
  if (resp.status === 304) return { checksum: adminListChecksum, messages: null };
  const js = await resp.json();
  adminListChecksum = js.checksum || adminListChecksum;
  return js;
}

function buildConversationList(allMsgs) {
  const map = new Map();
  (allMsgs || []).forEach(m => {
    const me = window.__adminEmail || '';
    const adminAddress = "admin@sudocrypt.com";

    let other = '';
    if (m.from === adminAddress) other = m.to;
    else if (m.to === adminAddress) other = m.from;
    else if (m.from === me) other = m.to;
    else if (m.to === me) other = m.from;
    else return;

    if (!other || other === adminAddress) return;

    const prev = map.get(other);
    const ts = Number(m.created_at || m.CreatedAt || 0);
    const otherName = (m.from && String(m.from).toLowerCase() === String(other).toLowerCase()) ? (m.from_name || other) : (m.to_name || other);
    if (!prev || ts > Number(prev.ts || 0)) map.set(other, { last: m.content, ts, name: otherName });
  });
  const items = Array.from(map.entries()).sort((a,b)=> Number(b[1].ts || 0) - Number(a[1].ts || 0));
  const cont = document.getElementById('adminChatList');
  cont.innerHTML = '';
  items.forEach(([email, meta]) => {
    const d = document.createElement('div');
    d.className = 'admin-chat-item' + (email === currentUser ? ' active' : '');
    d.textContent = meta.name || email;
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
  let maxID = 0;
  msgs.forEach(m => {
    const id = typeof m.id === 'number' ? m.id : parseInt(m.id || 0, 10) || 0;
    if (id > maxID) maxID = id;
  });
  const prevMax = lastRenderedMaxID[user] || 0;
  if (maxID > prevMax) {
    cont.scrollTop = cont.scrollHeight;
  }
  lastRenderedMaxID[user] = maxID;
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

async function fetchLeadsForLevel(level) {
  try {
    const resp = await fetch('/api/hints?level=' + encodeURIComponent(level), { credentials: 'same-origin' });
    if (!resp.ok) return [];
    const js = await resp.json();
    return Array.isArray(js.hints) ? js.hints : [];
  } catch (e) { return []; }
}

function renderLeads(list, container) {
  container.innerHTML = '';
  if (!Array.isArray(list) || list.length === 0) {
    container.innerHTML = '<div style="padding:8px;color:#888">No leads for this level.</div>';
    return;
  }
  list.forEach(h => {
    const el = document.createElement('div');
    el.style.padding = '8px';
    el.style.borderRadius = '6px';
    el.style.background = 'rgba(255,255,255,0.02)';
    el.style.display = 'flex';
    el.style.justifyContent = 'space-between';
    el.style.alignItems = 'center';
    el.style.gap = '8px';
    const txt = document.createElement('div');
    txt.style.flex = '1';
    txt.textContent = h.content || '';
    const del = document.createElement('button');
    del.className = 'button';
    del.textContent = 'Delete';
    del.addEventListener('click', async () => {
      const selEl = document.getElementById('leadLevelSelect');
      const curLevel = selEl ? selEl.value : '';
      await fetch('/api/admin/hints?level=' + encodeURIComponent(curLevel) + '&id=' + encodeURIComponent(h.id), { method: 'DELETE', credentials: 'same-origin' });
      const leads = await fetchLeadsForLevel(curLevel);
      renderLeads(leads, document.getElementById('leadList'));
    });
    el.appendChild(txt);
    el.appendChild(del);
    container.appendChild(el);
  });
}

function populateLeadLevelSelect() {
  const sel = document.getElementById('leadLevelSelect');
  if (!sel) return;
  sel.innerHTML = '';
  const levels = window.__adminLevels || {};
  const keys = Object.keys(levels).sort();
  keys.forEach(k => {
    const opt = document.createElement('option');
    opt.value = k;
    opt.textContent = k;
    sel.appendChild(opt);
  });
  sel.addEventListener('change', async () => {
    const v = sel.value;
    const leads = await fetchLeadsForLevel(v);
    renderLeads(leads, document.getElementById('leadList'));
  });
  if (keys.length > 0) {
    sel.value = keys[0];
    fetchLeadsForLevel(keys[0]).then(list => renderLeads(list, document.getElementById('leadList')));
  }
}

async function setupLeadsUI() {
  populateLeadLevelSelect();
  const addBtn = document.getElementById('leadAddBtn');
  if (!addBtn) return;
  addBtn.addEventListener('click', async () => {
    const sel = document.getElementById('leadLevelSelect');
    const input = document.getElementById('leadContentInput');
    if (!sel || !input) return;
    const level = sel.value;
    const content = input.value.trim();
    if (!content || !level) return;
    await fetch('/api/admin/hints', { method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin', body: JSON.stringify({ level, content, type: level.startsWith('ctf-') ? 'ctf' : 'cryptic' }) });
    input.value = '';
    const leads = await fetchLeadsForLevel(level);
    renderLeads(leads, document.getElementById('leadList'));
  });
}

document.addEventListener('DOMContentLoaded', function(){
  try { setupLeadsUI(); } catch(e) {}
});
