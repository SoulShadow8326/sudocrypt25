async function fetchCurrentLevel() {
    try {
        const url = '/api/play/current' + (window.location.search || '');
        console.log('[play.js] fetching current level from', url);
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            console.warn('[play.js] current level fetch failed', resp.status, await resp.text());
            return null;
        }
        const data = await resp.json();
        console.log('[play.js] current level data:', data);
        return data;
    } catch (err) {
        console.error('[play.js] error fetching current level', err);
        return null;
    }
}

async function fetchLeaderboard() {
    try {
        const url = '/api/leaderboard';
        console.log('[play.js] fetching leaderboard from', url);
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            console.warn('[play.js] leaderboard fetch failed', resp.status, await resp.text());
            return null;
        }
        const data = await resp.json();
        console.log('[play.js] leaderboard data:', data);
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
            console.log('[play.js] available levels:', levels);
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
        console.log('[play.js] public_hash:', lvl.public_hash || lvl.PublicHash || '(none)');
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
        console.log('[play.js] top leaderboard entries:', lb.slice(0, 10));
    }
}

window.addEventListener('load', initPlay);

window.__fetchCurrentLevel = fetchCurrentLevel;
window.__fetchLeaderboard = fetchLeaderboard;
window.__renderMarkup = renderMarkup;

async function submitAnswer() {
    try {
        const input = document.getElementById('messageInput');
        if (!input) return;
        const ans = input.value.trim();
        if (ans === '') return;
        const params = new URLSearchParams((new URL(window.location.href)).search);
        const type = params.get('type') || 'cryptic';
        const url = `/submit?answer=${encodeURIComponent(ans)}&type=${encodeURIComponent(type)}`;
        const resp = await fetch(url, { credentials: 'same-origin' });
        if (!resp.ok) {
            const txt = await resp.text();
            const n = new Notyf();
            n.error('submit failed');
            return;
        }
        const data = await resp.json();
        const n = new Notyf();
        if (data && data.success) {
            n.success('Correct');
            input.value = '';
            const lvl = await fetchCurrentLevel();
            if (lvl) renderMarkup(lvl.markup || markupHTML || '');
            fetchLeaderboard();
        } else {
            n.error('Incorrect');
        }
    } catch (err) {
        const n = new Notyf();
        n.error('Error');
    }
}

window.submit = submitAnswer;
