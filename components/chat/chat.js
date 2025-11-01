function Signal(key, initialValue) {
	let value = initialValue;
	let onChange = null;
	return {
		Value: function () { return value; },
		setValue: function (newValue) { value = newValue; if (onChange) onChange(); },
		set onChange(callback) { onChange = callback; }
	};
}

var chatSignal = Signal('chatOpenState', 'close');

let chatOpen = false;
let lastChecksum = '';
let lastSeenIncomingTs = 0; // latest timestamp o incoming message the user has sen
let lastRenderedMaxIDChat = 0;
let chatLastSendAt = 0;
const chatCooldownMs = 5000;
let chatCooldownInterval = null;

function getLatestIncomingTs(msgs) {
	let maxTs = 0;
	if (!Array.isArray(msgs)) return 0;
	msgs.forEach(m => {
		const ts = m.created_at || m.CreatedAt || 0;
		if (!m.is_me && typeof ts === 'number') {
			if (ts > maxTs) maxTs = ts;
		}
	});
	return maxTs;
}

function setupChatSignalHandlers() {
	const chatToggleBtn = document.getElementById("chatToggleBtn");
	const chatPopup = document.getElementById("chatPopup");
	const chatCloseBtn = document.getElementById("chatCloseBtn");
	const chatMinimizeBtn = document.getElementById("chatMinimizeBtn");
	if (!chatToggleBtn || !chatPopup) {
		console.error('Chat elements not found:', { chatToggleBtn: !!chatToggleBtn, chatPopup: !!chatPopup });
		return;
	}
	chatSignal.onChange = function () {
		if (chatSignal.Value() === "open") {
			chatOpen = true;
			chatPopup.style.display = "flex";
			chatToggleBtn.style.opacity = 0;
			chatToggleBtn.style.transform = "scale(0)";
			setTimeout(function () {
				chatPopup.style.opacity = 1;
				chatPopup.style.transform = "translateY(0px)";
				chatToggleBtn.style.display = "none";
			}, 10);
			if (typeof refreshChatContent === 'function') refreshChatContent();
			const messagecontainer = document.getElementById("chatContainer");
			setTimeout(function () {
				if (messagecontainer) messagecontainer.scrollTop = messagecontainer.scrollHeight;
			}, 200);
			const dot = document.getElementById('chatToggleNotificationDot');
			if (dot) dot.style.display = 'none';
		} else {
			chatOpen = false;
			chatToggleBtn.style.display = "block";
			chatPopup.style.opacity = 0;
			chatPopup.style.transform = "translateY(900px)";
			setTimeout(function () {
				chatPopup.style.display = "none";
				chatToggleBtn.style.opacity = 1;
				chatToggleBtn.style.transform = "scale(1)";
			}, 400);
		}
	};
	if (chatToggleBtn) {
		chatToggleBtn.addEventListener("click", function () {
			chatSignal.setValue("open");
		});
	}
	if (chatCloseBtn) {
		chatCloseBtn.addEventListener("click", function () {
			chatSignal.setValue("close");
		});
	}
	if (chatMinimizeBtn) {
		chatMinimizeBtn.addEventListener("click", function () {
			chatSignal.setValue("close");
		});
	}
}

async function fetchMessages(checksum) {
	try {
		const levelType = new URLSearchParams((new URL(window.location.href)).search).get('type') || 'cryptic';
		const base = checksum ? ("/api/messages?checksum=" + encodeURIComponent(checksum)) : "/api/messages";
		const sep = base.indexOf('?') !== -1 ? '&' : '?';
		const url = base + sep + "type=" + encodeURIComponent(levelType);
		const resp = await fetch(url, { credentials: 'same-origin' });
		if (resp.status === 304) return { checksum, messages: null };
		if (!resp.ok) throw new Error("messages fetch failed: " + resp.status);
		return await resp.json();
	} catch (e) {
		console.warn('[chat] fetchMessages error', e);
		return { checksum, messages: null };
	}
}

function renderMessages(msgs) {
	const container = document.getElementById('chatContainer')
	if (!container || !Array.isArray(msgs)) return;
	container.innerHTML = '';
	let maxID = 0;
	msgs.forEach(m => {
		const message = document.createElement('div');
		message.className = 'chat-message ' + (m.is_me ? 'user' : 'admin');
		const content = document.createElement('div');
		content.className = 'chat-message-content';
		
		const sender = document.createElement('div');
		sender.className = 'chat-message-label';
		sender.textContent = m.from_label || (m.is_me ? 'You' : 'Admin');
		
		const text = document.createElement('div');
		text.className = 'chat-message-text';
		text.textContent = m.content || '';
		
		content.appendChild(sender);
		content.appendChild(text);
		message.appendChild(content);
		container.appendChild(message);
		
	});
	msgs.forEach(m => {
		const id = typeof m.id === 'number' ? m.id : parseInt(m.id || 0, 10) || 0;
		if (id > maxID) maxID = id;
	});
	if (maxID > lastRenderedMaxIDChat) {
		container.scrollTop = container.scrollHeight;
	}
	lastRenderedMaxIDChat = Math.max(lastRenderedMaxIDChat, maxID);
	const latestIncoming = getLatestIncomingTs(msgs);
	if (chatOpen && latestIncoming > 0) {
		lastSeenIncomingTs = latestIncoming;
		const dot = document.getElementById('chatToggleNotificationDot');
		if (dot) dot.style.display = 'none';
	}
}

function renderHintsArray(hints) {
	let hintsContainer = document.getElementById('hintsContainer');
	if (!hintsContainer) {
		const chatPopup = document.getElementById('chatPopup') || document.body;
		if (chatPopup) {
			hintsContainer = document.createElement('div');
			hintsContainer.id = 'hintsContainer';
			hintsContainer.className = 'chat-messages-container no-scrollbar';
			hintsContainer.style.display = 'none';
			chatPopup.appendChild(hintsContainer);
		}
	}
	if (!hintsContainer) return;
	hintsContainer.innerHTML = '';
	if (!Array.isArray(hints) || hints.length === 0) {
		hintsContainer.innerHTML = '<div class="empty-state"><div class="empty-icon">?</div><p>No hints available yet.</p></div>';
		return;
	}
	hints.sort((a,b)=> (Number(a.time||0) - Number(b.time||0)));
	for (const h of hints) {
		const wrapper = document.createElement('div');
		wrapper.className = 'hint-message';
		const content = document.createElement('div');
		content.className = 'message-content';
		const p = document.createElement('p');
		p.textContent = (h.content || h.text || h.message || '') + '';
		content.appendChild(p);
		const timeEl = document.createElement('div');
		timeEl.className = 'message-time';
		if (h.time) {
			const t = Number(h.time)||0;
			try { timeEl.textContent = new Date(t*1000).toLocaleString(); } catch(e) { timeEl.textContent = String(t); }
		}
		wrapper.appendChild(content);
		wrapper.appendChild(timeEl);
		hintsContainer.appendChild(wrapper);
	}
}

async function doFetch(force) {
  try {
		const data = await fetchMessages(force ? '' : lastChecksum);
		if (!data) return;
		if (data.messages) {
			renderMessages(data.messages);
			lastChecksum = data.checksum || lastChecksum;
      const latestIncoming = getLatestIncomingTs(data.messages);
      const dot = document.getElementById('chatToggleNotificationDot');
      if (chatOpen) {
        if (dot) dot.style.display = 'none';
        if (latestIncoming > 0) lastSeenIncomingTs = latestIncoming;
      } else {
        if (latestIncoming > lastSeenIncomingTs) {
          if (dot) dot.style.display = 'block';
        }
      }
    }
		if (Array.isArray(data.hints)) {
			renderHintsArray(data.hints);
		}
  } catch (e) {
    console.warn('[chat] doFetch error', e);
  }
}

async function pollMessagesLoop() {
	while (true) {
		await doFetch(false);
		await new Promise(r => setTimeout(r, chatOpen ? 1500 : 10000));
	}
}

async function sendChatMessage() {
	const input = document.getElementById('chatInput');
	const content = (input && input.value || '').trim();
	if (!content) return;
	const now = Date.now();
	if (now - chatLastSendAt < chatCooldownMs) return;
	chatLastSendAt = now;
	const timerEl = document.getElementById('chatCooldownTimer');
	const sendBtnEl = document.getElementById('chatendButton');
	if (sendBtnEl) {
		sendBtnEl.disabled = true;
		sendBtnEl.style.display = 'none';
	}
	if (timerEl) timerEl.textContent = Math.ceil(chatCooldownMs/1000) + 's';
	if (chatCooldownInterval) clearInterval(chatCooldownInterval);
	chatCooldownInterval = setInterval(() => {
		const rem = Math.ceil((chatLastSendAt + chatCooldownMs - Date.now())/1000);
		if (!timerEl) return;
		if (rem <= 0) {
			timerEl.textContent = '';
			if (sendBtnEl) { sendBtnEl.disabled = false; sendBtnEl.style.display = ''; }
			clearInterval(chatCooldownInterval);
			chatCooldownInterval = null;
			return;
		}
		timerEl.textContent = rem + 's';
	}, 250);
	const to = 'admin@sudocrypt.com';
	const payload = { to, type: 'message', content };
	(function optimisticAppend() {
		const container = document.getElementById('chatContainer');
		if (!container) return;
		const message = document.createElement('div');
		message.className = 'chat-message user';
		const contentWrap = document.createElement('div');
		contentWrap.className = 'chat-message-content';
		const sender = document.createElement('div');
		sender.className = 'chat-message-label';
		sender.textContent = 'You';
		const text = document.createElement('div');
		text.className = 'chat-message-text';
		text.textContent = content;
		contentWrap.appendChild(sender);
		contentWrap.appendChild(text);
		message.appendChild(contentWrap);
		container.appendChild(message);
		container.scrollTop = container.scrollHeight;
	})();

	try {
		await fetch('/api/message/send', { method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin', body: JSON.stringify(payload) });
	} catch (e) {
		console.warn('Failed to send message', e);
	}
	if (input) input.value = '';
	if (typeof refreshChatContent === 'function') refreshChatContent();
}

document.addEventListener('DOMContentLoaded', function () {
	setupChatSignalHandlers();
	const btn = document.getElementById('chatendButton');
	const input = document.getElementById('chatInput');
	if (btn) {
		btn.disabled = false;
		btn.addEventListener('click', sendChatMessage);
	}
	if (input) {
		input.addEventListener('keydown', function (e) {
			if (e.key === 'Enter') sendChatMessage();
		});
	}
	pollMessagesLoop();
});

window.switchChatTab = function(tab) {
	const tabs = document.querySelectorAll('.chat-tab');
	tabs.forEach(t => { if (t.dataset && t.dataset.tab) { t.classList.toggle('active', t.dataset.tab === tab); } });
	const contents = document.querySelectorAll('.chat-tab-content');
	contents.forEach(c => { c.classList.toggle('active', c.id === (tab === 'leads' ? 'chatContent' : (tab === 'hints' ? 'hintsContent' : ''))); });
	const hintsContainer = document.getElementById('hintsContainer');
	if (hintsContainer) {
		if (tab === 'hints') {
			hintsContainer.style.display = '';
			loadHintsForCurrentLevel();
		} else {
			hintsContainer.style.display = 'none';
		}
	} else {
		if (tab === 'hints') loadHintsForCurrentLevel();
	}
}

async function loadHintsForCurrentLevel() {
	let hintsContainer = document.getElementById('hintsContainer');
	if (!hintsContainer) {
		const chatPopup = document.getElementById('chatPopup') || document.body;
		if (chatPopup) {
			hintsContainer = document.createElement('div');
			hintsContainer.id = 'hintsContainer';
			hintsContainer.className = 'chat-messages-container no-scrollbar';
			hintsContainer.style.display = 'none';
			chatPopup.appendChild(hintsContainer);
		}
	}
	if (!hintsContainer) return;
	hintsContainer.innerHTML = '';
	const levelType = new URLSearchParams((new URL(window.location.href)).search).get('type') || 'cryptic';
	try {
		const respMsgs = await fetch('/api/messages?type=' + encodeURIComponent(levelType), { credentials: 'same-origin' });
		if (respMsgs.ok) {
			const data = await respMsgs.json().catch(() => ({}));
			const hintsFromMsgs = Array.isArray(data.hints) ? data.hints : null;
			if (hintsFromMsgs !== null) {
				var hints = hintsFromMsgs;
			} else {
				const respLvl = await fetch('/api/play/current?type=' + encodeURIComponent(levelType), { credentials: 'same-origin' });
				if (!respLvl.ok) throw new Error('no level');
				const lvl = await respLvl.json();
				const levelID = lvl.ID || lvl.id || '';
				if (!levelID) throw new Error('no level');
				const resp = await fetch('/api/hints?level=' + encodeURIComponent(levelID), { credentials: 'same-origin' });
				if (!resp.ok) throw new Error('no hints');
				const js = await resp.json();
				var hints = Array.isArray(js.hints) ? js.hints : [];
			}
		} else {
			const respLvl = await fetch('/api/play/current?type=' + encodeURIComponent(levelType), { credentials: 'same-origin' });
			if (!respLvl.ok) throw new Error('no level');
			const lvl = await respLvl.json();
			const levelID = lvl.ID || lvl.id || '';
			if (!levelID) throw new Error('no level');
			const resp = await fetch('/api/hints?level=' + encodeURIComponent(levelID), { credentials: 'same-origin' });
			if (!resp.ok) throw new Error('no hints');
			const js = await resp.json();
			var hints = Array.isArray(js.hints) ? js.hints : [];
		}
		if (hints.length === 0) {
			hintsContainer.innerHTML = '<div class="empty-state"><div class="empty-icon">?</div><p>No hints available yet.</p></div>';
			return;
		}
		hints.sort((a,b)=> (Number(a.time||0) - Number(b.time||0)));
		for (const h of hints) {
			const wrapper = document.createElement('div');
			wrapper.className = 'hint-message';
			const content = document.createElement('div');
			content.className = 'message-content';
			const p = document.createElement('p');
			p.textContent = h.content || '';
			content.appendChild(p);
			const timeEl = document.createElement('div');
			timeEl.className = 'message-time';
			if (h.time) {
				const t = Number(h.time)||0;
				try { timeEl.textContent = new Date(t*1000).toLocaleString(); } catch(e) { timeEl.textContent = String(t); }
			}
			wrapper.appendChild(content);
			wrapper.appendChild(timeEl);
			hintsContainer.appendChild(wrapper);
		}
	} catch (e) {
		hintsContainer.innerHTML = '<div class="empty-state"><div class="empty-icon">?</div><p>No hints available yet.</p></div>';
	}
}

async function refreshChatContent() {
	await doFetch(true);
}


