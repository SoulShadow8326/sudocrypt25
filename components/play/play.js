async function fetchCurrentLevel() {
    try {
        const url = '/api/play/current' + (window.location.search || '');
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            console.warn('[play.js] current level fetch failed', resp.status, await resp.text());
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
            console.warn('[play.js] leaderboard fetch failed', resp.status, await resp.text());
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

async function initPlay() {
    try {
        const levelsResp = await fetch('/api/levels', { credentials: 'same-origin' });
        if (!levelsResp.ok) {
            console.warn('[play.js] levels list fetch failed', levelsResp.status);
        } else {
            const levels = await levelsResp.json();
            if (!Array.isArray(levels) || levels.length === 0) {
                const level_title = document.getElementById("level_title");
                if (level_title) level_title.style.display = 'none';
                renderMarkup('<p><em>No levels are available currently. Please check back later.</em></p>');
                const input = document.getElementById('messageInput');
                const sendBtn = document.getElementById('sendButton');
                if (input) input.disabled = true;
                if (sendBtn) sendBtn.disabled = true;
                const chat = document.getElementById('chatToggleBtn');
                if (chat) chat.style.display = 'none';
                return;
            }
        }
    } catch (err) {
        console.error('[play.js] error fetching levels list', err);
    }

    const lvl = await fetchCurrentLevel();
    if (lvl) {
        renderMarkup(lvl.markup || markupHTML || '');
        try {
            var titleEl = document.querySelector('.title');
            if (titleEl && lvl.id) {
                var parts = lvl.id.split('-');
                if (parts.length === 2) {
                    var num = parts[1];
                    titleEl.innerText = 'Level ' + num;
                } else {
                    titleEl.innerText = 'Level ' + lvl.id;
                }
            }
        } catch (e) {
            console.warn('[play.js] failed to update title', e);
        }
    }
    const lb = await fetchLeaderboard();
    if (lb) {
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
        console.warn('[play.js] failed to attach enter handler', e);
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
