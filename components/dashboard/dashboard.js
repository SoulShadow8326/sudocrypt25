let currentUser = '';
let currentChecksum = '';
let currentUserCTF = '';
let currentChecksumCTF = '';
let currentCTFLevel = '';
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
    const levelVal = (m.level_id || m.LevelID || m.level || m.Level || '') || '';
    if (String(levelVal || '').toLowerCase().startsWith('ctf')) return;
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
    const isIncomingToAdmin = (m.to === adminAddress);
    const isUnread = Number(m.read || 0) === 0 && isIncomingToAdmin && String(m.from || '').toLowerCase() === String(other).toLowerCase();
    if (!prev || ts > Number(prev.ts || 0)) map.set(other, { last: m.content, ts, name: otherName, unread: isUnread });
    else if (prev && isUnread) map.set(other, Object.assign({}, prev, { unread: true }));
  });
  const items = Array.from(map.entries()).sort((a,b)=> Number(b[1].ts || 0) - Number(a[1].ts || 0));
  const cont = document.getElementById('adminChatList');
  cont.innerHTML = '';
  items.forEach(([email, meta]) => {
    const d = document.createElement('div');
    d.className = 'admin-chat-item' + (email === currentUser ? ' active' : '') + (meta && meta.unread ? ' unread' : '');
    d.dataset.email = email;
    d.textContent = meta.name || email;
    d.addEventListener('click', () => { selectConversation(email); });
    cont.appendChild(d);
  });
}

function buildConversationListCTF(allMsgs) {
  const map = new Map();
  (allMsgs || []).forEach(m => {
    const me = window.__adminEmail || '';
    const adminAddress = "admin@sudocrypt.com";
    const levelVal = (m.level_id || m.LevelID || m.level || m.Level || '') || '';
    if (!String(levelVal || '').toLowerCase().startsWith('ctf')) return;
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
    const isIncomingToAdmin = (m.to === adminAddress);
    const isUnread = Number(m.read || 0) === 0 && isIncomingToAdmin && String(m.from || '').toLowerCase() === String(other).toLowerCase();
    if (!prev || ts > Number(prev.ts || 0)) map.set(other, { last: m.content, ts, name: otherName, unread: isUnread });
    else if (prev && isUnread) map.set(other, Object.assign({}, prev, { unread: true }));
  });
  const items = Array.from(map.entries()).sort((a,b)=> Number(b[1].ts || 0) - Number(a[1].ts || 0));
  const cont = document.getElementById('adminChatListCTF');
  cont.innerHTML = '';
  items.forEach(([email, meta]) => {
    const d = document.createElement('div');
    d.className = 'admin-chat-item' + (email === currentUserCTF ? ' active' : '') + (meta && meta.unread ? ' unread' : '');
    d.dataset.email = email;
    d.textContent = meta.name || email;
    d.addEventListener('click', () => { selectConversationCTF(email); });
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
  
  renderInternalCheckpoints(user);
  const cont = document.getElementById('adminChatMessages');
  if (!data || !data.messages) return;
  const msgs = data.messages;
  const me = window.__adminEmail || '';
  const adminAddress = "admin@sudocrypt.com";
  
  cont.innerHTML = '';
  msgs.forEach(m => {
    const level = m.level_id || m.LevelID || m.level || m.Level || '';
    if (String(level || '').toLowerCase().startsWith('ctf')) return;
    const row = document.createElement('div');
    const isFromAdmin = (m.from === me || m.from === adminAddress);
    row.className = 'msg-row' + (isFromAdmin ? ' msg-me' : '');
    
    const bubble = document.createElement('div');
    bubble.className = 'msg-bubble';
    const content = document.createElement('div');
    content.className = 'msg-content';
    if (level) {
      const lvlEl = document.createElement('div');
      lvlEl.className = 'msg-level';
      lvlEl.style.fontSize = '11px';
      lvlEl.style.opacity = '0.75';
      lvlEl.style.marginBottom = '6px';
      lvlEl.textContent = String(level);
      bubble.appendChild(lvlEl);
    }
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

function renderThreadCTF(user, data) {
  const header = document.getElementById('adminChatHeaderCTF');
  header.textContent = user || 'Select a CTF conversation';
  renderInternalCheckpointsCTF(user);
  const cont = document.getElementById('adminChatMessagesCTF');
  if (!data || !data.messages) return;
  const msgs = data.messages;
  const me = window.__adminEmail || '';
  const adminAddress = "admin@sudocrypt.com";
  cont.innerHTML = '';
  msgs.forEach(m => {
    const level = m.level_id || m.LevelID || m.level || m.Level || '';
    if (!String(level || '').toLowerCase().startsWith('ctf')) return;
    const row = document.createElement('div');
    const isFromAdmin = (m.from === me || m.from === adminAddress);
    row.className = 'msg-row' + (isFromAdmin ? ' msg-me' : '');
    const bubble = document.createElement('div');
    bubble.className = 'msg-bubble';
    const content = document.createElement('div');
    content.className = 'msg-content';
    if (level) {
      const lvlEl = document.createElement('div');
      lvlEl.className = 'msg-level';
      lvlEl.style.fontSize = '11px';
      lvlEl.style.opacity = '0.75';
      lvlEl.style.marginBottom = '6px';
      lvlEl.textContent = String(level);
      bubble.appendChild(lvlEl);
    }
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

async function selectConversationCTF(user) {
  currentUserCTF = user;
  currentChecksumCTF = '';
  const data = await fetchThread(user, currentChecksumCTF);
  if (data) {
    currentChecksumCTF = data.checksum || currentChecksumCTF;
    renderThreadCTF(user, data);
  }
}

async function renderInternalCheckpoints(user) {
  try {
    const resp = await fetch('/api/admin/user/progress?email=' + encodeURIComponent(user), { credentials: 'same-origin' });
    if (!resp.ok) return;
    const js = await resp.json();
    const prog = js.progress || {};
    let crypticArr = [];
    let ctfArr = [];
    if (Array.isArray(prog)) {
      if (prog.length >= 2) crypticArr = prog;
    } else if (prog && typeof prog === 'object') {
      if (Array.isArray(prog.cryptic)) crypticArr = prog.cryptic;
      if (Array.isArray(prog.ctf)) ctfArr = prog.ctf;
    }

    const container = document.getElementById('adminCheckList');
    if (!container) return;
    container.innerHTML = '';

    function buildSection(type, arr) {
      const section = document.createElement('div');
      section.style.marginBottom = '24px';
      const title = document.createElement('h3');
      title.textContent = type === 'cryptic' ? 'Cryptic Walkthrough' : 'CTF Walkthrough';
      title.style.margin = '0 0 12px 0';
      section.appendChild(title);

      const levelId = (Array.isArray(arr) && arr.length > 0 && typeof arr[0] === 'string') ? arr[0] : null;
      if (type === 'ctf') {
        currentCTFLevel = levelId || '';
      }
      const progressVal = (Array.isArray(arr) && arr.length > 1) ? Number(arr[1] || 0) : 0;

      const info = document.createElement('div');
      info.style.fontSize = '13px';
      info.style.opacity = '0.9';
      info.style.marginBottom = '12px';
      info.textContent = levelId ? String(levelId) : 'No level'
      section.appendChild(info);

      const partsWrap = document.createElement('div');
      partsWrap.className = 'checkpoint-section';

      if (!levelId) {
        const empty = document.createElement('div'); 
        empty.textContent = 'No walkthrough parts'; 
        empty.style.paddingLeft = '0';
        partsWrap.appendChild(empty);
      } else {
        const levels = window.__adminLevels || {};
        const lvl = levels[levelId] || {};
        let parts = [];
        try {
          const raw = lvl.walkthrough || '';
          const parsed = JSON.parse(raw || 'null');
          if (Array.isArray(parsed)) parts = parsed.map(p=>String(p||''));
          else if (raw && String(raw).trim() !== '') parts = [String(raw)];
        } catch(e) {
          const raw = lvl.walkthrough || '';
          if (raw && String(raw).trim() !== '') parts = [String(raw)];
        }
        for (let i=0;i<parts.length;i++) {
          const item = document.createElement('div');
          item.className = 'checkpoint-item';

          const cb = document.createElement('input');
          cb.type = 'checkbox';
          cb.className = 'checkpoint-node';
          cb.id = 'chk-' + type + '-' + levelId + '-' + String(i);
          cb.dataset.type = type;
          cb.dataset.level = levelId;
          cb.dataset.index = String(i);
          cb.checked = i < progressVal;
          cb.addEventListener('change', async function(){
            try {
              const wrap = this.closest('#adminCheckList');
              const nodeList = wrap.querySelectorAll('input[type="checkbox"][data-type="'+type+'"][data-level="'+levelId+'"]');
              const cbs = Array.from(nodeList).sort((a,b)=> Number(a.dataset.index) - Number(b.dataset.index));
              const idx = Number(this.dataset.index || 0);
              if (this.checked) {
                for (let j=0;j<=idx && j<cbs.length;j++) cbs[j].checked = true;
              } else {
                for (let j=idx;j<cbs.length;j++) cbs[j].checked = false;
              }
              let cnt = 0;
              for (const el of cbs) if (el.checked) cnt++;
              await fetch('/api/admin/user/progress', { method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ email: user, action: 'set', type: type, progress: [levelId, cnt] }) });
            } catch(e) {}
          });

          const line = document.createElement('div');
          line.className = 'checkpoint-line';

          const content = document.createElement('label');
          content.className = 'checkpoint-content';
          content.htmlFor = cb.id;
          content.textContent = parts[i] || ('Part ' + String(i));

          item.appendChild(cb);
          item.appendChild(line);
          item.appendChild(content);
          partsWrap.appendChild(item);
        }
        if (parts.length === 0) { 
          const none = document.createElement('div'); 
          none.textContent = 'No walkthrough parts'; 
          none.style.paddingLeft = '0';
          partsWrap.appendChild(none); 
        }
      }
      section.appendChild(partsWrap);
      return section;
    }

    const crypticSection = buildSection('cryptic', crypticArr);
    container.appendChild(crypticSection);
  } catch (e) {}
}

async function renderInternalCheckpointsCTF(user) {
  try {
    const resp = await fetch('/api/admin/user/progress?email=' + encodeURIComponent(user), { credentials: 'same-origin' });
    if (!resp.ok) return;
    const js = await resp.json();
    const prog = js.progress || {};
    let crypticArr = [];
    let ctfArr = [];
    if (Array.isArray(prog)) {
      if (prog.length >= 2) crypticArr = prog;
    } else if (prog && typeof prog === 'object') {
      if (Array.isArray(prog.cryptic)) crypticArr = prog.cryptic;
      if (Array.isArray(prog.ctf)) ctfArr = prog.ctf;
    }
    const container = document.getElementById('adminCheckListCTF');
    if (!container) return;
    container.innerHTML = '';
    function buildSection(type, arr) {
      const section = document.createElement('div');
      section.style.marginBottom = '24px';
      const title = document.createElement('h3');
      title.textContent = type === 'cryptic' ? 'Cryptic Walkthrough' : 'CTF Walkthrough';
      title.style.margin = '0 0 12px 0';
      section.appendChild(title);
      const levelId = (Array.isArray(arr) && arr.length > 0 && typeof arr[0] === 'string') ? arr[0] : null;
      const progressVal = (Array.isArray(arr) && arr.length > 1) ? Number(arr[1] || 0) : 0;
      const info = document.createElement('div');
      info.style.fontSize = '13px';
      info.style.opacity = '0.9';
      info.style.marginBottom = '12px';
      info.textContent = levelId ? String(levelId) : 'No level'
      section.appendChild(info);
      const partsWrap = document.createElement('div');
      partsWrap.className = 'checkpoint-section';
      if (!levelId) {
        const empty = document.createElement('div'); 
        empty.textContent = 'No walkthrough parts'; 
        empty.style.paddingLeft = '0';
        partsWrap.appendChild(empty);
      } else {
        const levels = window.__adminLevels || {};
        const lvl = levels[levelId] || {};
        let parts = [];
        try {
          const raw = lvl.walkthrough || '';
          const parsed = JSON.parse(raw || 'null');
          if (Array.isArray(parsed)) parts = parsed.map(p=>String(p||''));
          else if (raw && String(raw).trim() !== '') parts = [String(raw)];
        } catch(e) {
          const raw = lvl.walkthrough || '';
          if (raw && String(raw).trim() !== '') parts = [String(raw)];
        }
        for (let i=0;i<parts.length;i++) {
          const item = document.createElement('div');
          item.className = 'checkpoint-item';
          const cb = document.createElement('input');
          cb.type = 'checkbox';
          cb.className = 'checkpoint-node';
          cb.id = 'chk-' + type + '-' + levelId + '-' + String(i);
          cb.dataset.type = type;
          cb.dataset.level = levelId;
          cb.dataset.index = String(i);
          cb.checked = i < progressVal;
          cb.addEventListener('change', async function(){
            try {
              const wrap = this.closest('#adminCheckListCTF');
              const nodeList = wrap.querySelectorAll('input[type="checkbox"][data-type="'+type+'"][data-level="'+levelId+'"]');
              const cbs = Array.from(nodeList).sort((a,b)=> Number(a.dataset.index) - Number(b.dataset.index));
              const idx = Number(this.dataset.index || 0);
              if (this.checked) {
                for (let j=0;j<=idx && j<cbs.length;j++) cbs[j].checked = true;
              } else {
                for (let j=idx;j<cbs.length;j++) cbs[j].checked = false;
              }
              let cnt = 0;
              for (const el of cbs) if (el.checked) cnt++;
              await fetch('/api/admin/user/progress', { method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ email: user, action: 'set', type: type, progress: [levelId, cnt] }) });
            } catch(e) {}
          });
          const line = document.createElement('div');
          line.className = 'checkpoint-line';
          const content = document.createElement('label');
          content.className = 'checkpoint-content';
          content.htmlFor = cb.id;
          content.textContent = parts[i] || ('Part ' + String(i));
          item.appendChild(cb);
          item.appendChild(line);
          item.appendChild(content);
          partsWrap.appendChild(item);
        }
        if (parts.length === 0) { 
          const none = document.createElement('div'); 
          none.textContent = 'No walkthrough parts'; 
          none.style.paddingLeft = '0';
          partsWrap.appendChild(none); 
        }
      }
      section.appendChild(partsWrap);
      return section;
    }
    const ctfSection = buildSection('ctf', ctfArr);
    container.appendChild(ctfSection);
  } catch (e) {}
}

async function pollThreadLoop() {
  while (true) {
    try {
      const all = await fetchAllMessages();
      if (all && all.messages) {
        buildConversationList(all.messages || []);
        buildConversationListCTF(all.messages || []);
      }
      
      if (currentUser) {
        const data = await fetchThread(currentUser, currentChecksum);
        if (data && data.messages) {
          currentChecksum = data.checksum || currentChecksum;
          renderThread(currentUser, data);
        }
      }
      if (currentUserCTF) {
        const data2 = await fetchThread(currentUserCTF, currentChecksumCTF);
        if (data2 && data2.messages) {
          currentChecksumCTF = data2.checksum || currentChecksumCTF;
          renderThreadCTF(currentUserCTF, data2);
        }
      }
    } catch (e) {
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

    const resp = await fetch('/api/message/send', { 
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' }, 
      credentials: 'same-origin', 
      body: JSON.stringify({ to: currentUser, type: 'message', content }) 
    });
    if (!resp.ok) {
      try {
        const data = await resp.json().catch(()=>null);
        if (data && data.when) {
          const n = typeof Notyf !== 'undefined' ? new Notyf() : null
          if (n) n.error(data.error || (data.when === 'before' ? 'The event has not commenced yet' : 'The event has concluded'))
          setTimeout(function(){ window.location = '/timegate?toast=1&from=/send_message&when=' + encodeURIComponent(data.when); }, 1200)
          return;
        }
      } catch (e) {}
    }
    

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

async function sendToCurrentCTF() {
  const input = document.getElementById('adminChatInputCTF');
  const content = (input.value || '').trim();
  if (!content || !currentUserCTF) return;
  try {
    // prefer the selected user's known CTF level, fall back to DOM or generic
    let inferredLevel = currentCTFLevel || '';
    try {
      if (!inferredLevel) {
        const cont = document.getElementById('adminChatMessagesCTF');
        if (cont) {
          const lvls = cont.querySelectorAll('.msg-level');
          if (lvls && lvls.length > 0) inferredLevel = lvls[lvls.length - 1].textContent || '';
        }
      }
    } catch (e) {}
    if (!inferredLevel) inferredLevel = 'ctf';

    (function optimisticAppend() {
      const cont = document.getElementById('adminChatMessagesCTF');
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

    const resp = await fetch('/api/message/send', { 
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' }, 
      credentials: 'same-origin', 
      body: JSON.stringify({ to: currentUserCTF, type: 'message', content, level: inferredLevel }) 
    });
    if (!resp.ok) {
      try { const data = await resp.json().catch(()=>null); } catch(e) {}
    }
    const data = await fetchThread(currentUserCTF, '');
    if (data) {
      currentChecksumCTF = data.checksum || '';
      renderThreadCTF(currentUserCTF, data);
    }
    input.value = '';
  } catch (e) {
  }
}

async function bootstrap() {
  try {
    const all = await fetchAllMessages();
    buildConversationList(all.messages || []);
    buildConversationListCTF(all.messages || []);
  } catch (e) {
  }
  document.getElementById('adminChatSend').addEventListener('click', sendToCurrent);
  document.getElementById('adminChatInput').addEventListener('keydown', function(e){ if (e.key==='Enter') sendToCurrent(); });
  const sendCTF = document.getElementById('adminChatSendCTF');
  if (sendCTF) sendCTF.addEventListener('click', sendToCurrentCTF);
  const inputCTF = document.getElementById('adminChatInputCTF');
  if (inputCTF) inputCTF.addEventListener('keydown', function(e){ if (e.key==='Enter') sendToCurrentCTF(); });
  const markBtn = document.getElementById('adminMarkRead');
  if (markBtn) markBtn.addEventListener('click', markCurrentAsRead);
  const markBtnCTF = document.getElementById('adminMarkReadCTF');
  if (markBtnCTF) markBtnCTF.addEventListener('click', markCurrentAsReadCTF);
  pollThreadLoop();


  function visibilityRefresh() {
    if (currentUser) {
      fetchThread(currentUser, '').then(data => { if (data) { currentChecksum = data.checksum || ''; renderThread(currentUser, data); } });
    }
    if (currentUserCTF) fetchThread(currentUserCTF, '').then(data => { if (data) { currentChecksumCTF = data.checksum || ''; renderThreadCTF(currentUserCTF, data); } });
    fetchAllMessages().then(all => { if (all && all.messages) { buildConversationList(all.messages || []); buildConversationListCTF(all.messages || []); } }).catch(()=>{});
  }
  document.addEventListener('visibilitychange', function(){ if (!document.hidden) visibilityRefresh(); });
  window.addEventListener('focus', visibilityRefresh);
}

async function markCurrentAsRead() {
  try {
    if (!currentUser) return;
    const resp = await fetch('/api/admin/messages/mark_read', { method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ email: currentUser }) });
    if (!resp.ok) return;
    const js = await resp.json().catch(()=>null);
    try { const all = await fetchAllMessages(); if (all && all.messages) buildConversationList(all.messages || []); } catch(e) {}
    try { const data = await fetchThread(currentUser, ''); if (data) { currentChecksum = data.checksum || ''; renderThread(currentUser, data); } } catch(e) {}
    const listEl = document.getElementById('adminChatList');
    if (listEl) {
      const items = listEl.querySelectorAll('.admin-chat-item');
      items.forEach(it => {
        if ((it.textContent || '').trim() === (currentUser || '').trim() || it.dataset && it.dataset.email === currentUser) {
          it.classList.remove('unread');
        }
      });
    }
    return js;
  } catch (e) {}
}

async function markCurrentAsReadCTF() {
  try {
    if (!currentUserCTF) return;
    const resp = await fetch('/api/admin/messages/mark_read', { method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ email: currentUserCTF }) });
    if (!resp.ok) return;
    const js = await resp.json().catch(()=>null);
    try { const all = await fetchAllMessages(); if (all && all.messages) { buildConversationList(all.messages || []); buildConversationListCTF(all.messages || []); } } catch(e) {}
    try { const data = await fetchThread(currentUserCTF, ''); if (data) { currentChecksumCTF = data.checksum || ''; renderThreadCTF(currentUserCTF, data); } } catch(e) {}
    const listEl = document.getElementById('adminChatListCTF');
    if (listEl) {
      const items = listEl.querySelectorAll('.admin-chat-item');
      items.forEach(it => {
        if ((it.textContent || '').trim() === (currentUserCTF || '').trim() || it.dataset && it.dataset.email === currentUserCTF) {
          it.classList.remove('unread');
        }
      });
    }
    return js;
  } catch (e) {}
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
  if (!container) container = document.getElementById('allLeadList');
  if (!container) return;
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
      try { await renderAllLeads(document.getElementById('allLeadList')); } catch(e) {}
    });
    el.appendChild(txt);
    el.appendChild(del);
    container.appendChild(el);
  });
}

async function renderAllLeads(container) {
  if (!container) return;
  container.innerHTML = '';
  const levels = window.__adminLevels || {};
  const keys = Object.keys(levels).sort();
  if (keys.length === 0) {
    container.innerHTML = '<div style="padding:8px;color:#888">No levels available.</div>';
    return;
  }
  for (const k of keys) {
    const section = document.createElement('div');
    section.style.display = 'flex';
    section.style.flexDirection = 'column';
    section.style.gap = '6px';
    const header = document.createElement('div');
    header.textContent = k;
    header.style.fontWeight = '600';
    header.style.fontSize = '13px';
    section.appendChild(header);
    const sub = document.createElement('div');
    sub.style.display = 'flex';
    sub.style.flexDirection = 'column';
    sub.style.gap = '6px';
    section.appendChild(sub);
    container.appendChild(section);
    const leads = await fetchLeadsForLevel(k);
    renderLeads(leads, sub);
  }
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
    try { await renderAllLeads(document.getElementById('allLeadList')); } catch(e) {}
    scrollToLevel(v);
  });
  if (keys.length > 0) {
    sel.value = keys[0];
    try { renderAllLeads(document.getElementById('allLeadList')); } catch(e) {}
    scrollToLevel(keys[0]);
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
    try { await renderAllLeads(document.getElementById('allLeadList')); } catch(e) {}
    scrollToLevel(level);
  });
}

function scrollToLevel(level) {
  if (!level) return;
  try {
    const container = document.getElementById('allLeadList');
    if (!container) return;
    const headers = container.querySelectorAll('div > div');
    for (const h of headers) {
      if (String(h.textContent || '').trim() === String(level)) {
        h.scrollIntoView({ behavior: 'smooth', block: 'start' });
        return;
      }
    }
  } catch (e) {}
}

document.addEventListener('DOMContentLoaded', function(){
  try { setupLeadsUI(); } catch(e) {}
});
