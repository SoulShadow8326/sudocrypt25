async function fetchCurrentLevel() {
    try {
        const url = '/api/play/current' + (window.location.search || '');
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            return null;
        }
    const data = await resp.json();
        return data;
    } catch (err) {
        console.error('[play.js] error fetching current level', err);
        return null;
    }
}

async function fetchLeaderboard() {
    try {
        const url = '/api/leaderboard';
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            return null;
        }
        const data = await resp.json();
        return data;
    } catch (err) {
        console.error('[play.js] error fetching leaderboard', err);
        return null;
    }
}

function renderMarkup(markup) {
    try {
        const el = document.getElementById('markup');
        if (!el) return;
        el.innerHTML = markup;
    } catch (err) {
        console.error('[play.js] render error', err);
    }
}

async function sha256Hex(message) {
    const enc = new TextEncoder();
    const data = enc.encode(message);
    const hash = await crypto.subtle.digest('SHA-256', data);
    const bytes = new Uint8Array(hash);
    let s = '';
    for (let i = 0; i < bytes.length; i++) {
        s += bytes[i].toString(16).padStart(2, '0');
    }
    return s;
}

async function initPlay() {
    const lvl = await fetchCurrentLevel();
    
    let isValidLevel = lvl && lvl.id && lvl.id !== "";
    
    if (lvl) {
        renderMarkup(lvl.markup || markupHTML || '');
        window.__currentLevelId = lvl.id;

        try {
            if (typeof lvl.leads_enabled !== 'undefined') {
                window.__leadsEnabledForCurrentLevel = !!lvl.leads_enabled;
            }
        } catch (e) {
        }
        
        try {
            var titleEl = document.querySelector('.title');
            if (titleEl && lvl.id && lvl.id !== "") {
                var parts = lvl.id.split('-');
                if (parts.length === 2) {
                    var num = parts[1];
                    titleEl.innerText = 'Level ' + num;
                } else {
                    titleEl.innerText = 'Level ' + lvl.id;
                }
            }
        } catch (e) {
        }
    }
    
    let shouldDisableInput = false;
    
    try {
        const levelsResp = await fetch('/api/levels', { credentials: 'same-origin' });
        if (!levelsResp.ok) {
            shouldDisableInput = true; 
        } else {
            const levels = await levelsResp.json();
            if (!Array.isArray(levels) || 
                levels.length === 0 || 
                !window.__currentLevelId || 
                window.__currentLevelId === "" || 
                !levels.includes(window.__currentLevelId)) {
                
                shouldDisableInput = true;
                
                const level_title = document.getElementById("level_title");
                if (level_title) level_title.style.display = 'none';
                
                renderMarkup('<p><em>No further levels are available currently. Thank you for playing!</em></p>');
                
                const chat = document.getElementById('chatToggleBtn');
                if (chat) chat.style.display = 'none';
            }
        }
    } catch (err) {
        shouldDisableInput = true; 
    }
    
    try {
        const input = document.getElementById('messageInput');
        const sendBtn = document.getElementById('sendButton');
        
        if (shouldDisableInput) {
            if (input) {
                input.disabled = true;
                input.placeholder = 'No levels available';
            }
            if (sendBtn) sendBtn.disabled = true;
            window.__leadsEnabledForCurrentLevel = false;
        } else {
            if (input) {
                input.disabled = false;
                input.placeholder = 'Enter your answer';
            }
            if (sendBtn) sendBtn.disabled = false;
            if (typeof lvl !== 'undefined' && typeof lvl.leads_enabled !== 'undefined') {
                window.__leadsEnabledForCurrentLevel = !!lvl.leads_enabled;
            } else {
                window.__leadsEnabledForCurrentLevel = true;
            }
        }
    } catch (e) {
        console.error('[play.js] error setting input state', e);
    }

    try {
        const input = document.getElementById('messageInput');
        if (input) {
            input.addEventListener('keydown', function (e) {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    submitAnswer();
                }
            });
        }
    } catch (e) {
    }
}

var lastSubmitAt = 0;
const submitCooldownMs = 3000;

window.addEventListener('load', initPlay);

window.__fetchCurrentLevel = fetchCurrentLevel;
window.__fetchLeaderboard = fetchLeaderboard;
window.__renderMarkup = renderMarkup;

async function submitAnswer() {
    try {
        const input = document.getElementById('messageInput');
        if (!input) return;
        const now = Date.now();
        if (now - lastSubmitAt < submitCooldownMs) {
            const n = new Notyf();
            n.error('Please wait before submitting again');
            return;
        }
        lastSubmitAt = now;
        const sendBtn = document.getElementById('sendButton');
        if (sendBtn) sendBtn.disabled = true;
        setTimeout(() => { if (sendBtn) sendBtn.disabled = false; }, submitCooldownMs);

        const params = new URLSearchParams((new URL(window.location.href)).search);
        const type = params.get('type') || 'cryptic';
        let ansRaw = input.value;

        const levelId = window.__currentLevelId || '';
        const typeWithLevel = levelId ? levelId : type;
        await post_log(typeWithLevel, ansRaw);

        if (type === 'cryptic') {
            const v = ansRaw.trim().toLowerCase();
            if (v === '') return;
            if (!/^[a-z]+$/.test(v)) {
                const n = new Notyf();
                n.error('Invalid answer format');
                return;
            }
            ansRaw = v;
        } else {
            ansRaw = ansRaw.trim();
            if (ansRaw === '') return;
        }
        
        const url = `/submit?answer=${encodeURIComponent(ansRaw)}&type=${encodeURIComponent(type)}`;
        const resp = await fetch(url, { credentials: 'same-origin' });
        let data = null;
        if (!resp.ok) {
            try {
                data = await resp.json();
            } catch (e) {
                data = null;
            }
            const n = new Notyf();
            if (resp.status === 403) {
                if (data && data.when) {
                    const msg = data.error || (data.when === 'before' ? 'The event has not commenced yet' : 'The event has concluded');
                    n.error(msg);
                    setTimeout(function () { window.location = '/timegate?toast=1&from=/submit&when=' + encodeURIComponent(data.when); }, 1200);
                    return;
                } else {
                    n.error('The event has concluded');
                    setTimeout(function () { window.location = '/timegate?toast=1&from=/submit'; }, 1200);
                    return;
                }
            }
            if (data && data.success === false) {
                n.error('incorrect');
            } else {
                n.error('submit failed');
            }
            return;
        }
        try {
            data = await resp.json();
        } catch (e) {
            data = null;
        }
        const n = new Notyf();
        if (data && data.success) {
            n.success('Correct');
            setTimeout(function () { window.location.reload(); }, 300);
        } else {
            n.error('incorrect');
        }
    } catch (err) {
        const n = new Notyf();
        n.error('submit failed');
    }
}

window.submit = submitAnswer;

async function post_log(type, ans){
	try{
		const response = await fetch('/api/attempt_logs', {
			method: 'POST',
			credentials: 'include',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				type: type,
				logs: ans
			})
		});
		if (!response.ok) {
			console.error('Error adding attempt_log!');	
		}
	}catch(error){
        console.error('Error fetching attempt logs:', error);
	}
}

function getCookie(name) {
	const nameEQ = name + "=";
	const decodedCookie = decodeURIComponent(document.cookie);
	const ca = decodedCookie.split(';');

	for (let i = 0; i < ca.length; i++) {
		let c = ca[i];
		while (c.charAt(0) === ' ') {
			c = c.substring(1);
		}
		if (c.indexOf(nameEQ) === 0) {
			return c.substring(nameEQ.length, c.length);
		}
	}
	return null; 
}
