const converter = new showdown.Converter();
const notyf = typeof Notyf !== 'undefined' ? new Notyf() : null
const inputEl = document.getElementById('inputField');
const displayEl = document.getElementById('displayBox');
const popupContainer = document.getElementById('popupContainer');
const popupContent = document.getElementById('popupContent');
const sourceHintField = document.getElementById('sourceHintField');
const answerField = document.getElementById('answerField');
const walkthroughField = document.getElementById('walkthroughField');
const addWalkthroughPartBtn = document.getElementById('addWalkthroughPartBtn');
const clearWalkthroughPartsBtn = document.getElementById('clearWalkthroughPartsBtn');
const levelName = document.getElementById("level_id")

function updateDisplay() {
    const inputValue = inputEl.value.trim();
    const html = converter.makeHtml(inputValue);
    displayEl.innerHTML = `${html}`;
    document.querySelectorAll("#displayBox a").forEach((x)=>{
        x.target="_blank"
    })
}

inputEl.addEventListener('input', updateDisplay);

popupContainer.addEventListener('click', function (e) {
    if (e.target === popupContainer) {
        closePopup()
    }
});

document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
        closePopup()
    }
});

function openPopup(levelNumber) {
    if (levelNumber == -1) {
        inputEl.value = ""
        displayEl.value = ""
        sourceHintField.value = ''
        answerField.value = ''
        if (walkthroughField) walkthroughField.value = ''
        const walkthroughPartsContainer = document.getElementById('walkthroughParts');
        if (walkthroughPartsContainer) {
            walkthroughPartsContainer.innerHTML = '';
            const ta = document.createElement('textarea');
            ta.className = 'walkthrough-part form-input';
            ta.setAttribute('data-index', '0');
            ta.placeholder = 'Walkthrough part 0';
            ta.style.minHeight = '120px';
            ta.style.resize = 'vertical';
            walkthroughPartsContainer.appendChild(ta);
        }
        popupContainer.style.display = 'flex'
        levelName.innerText = "New Level"
        document.getElementById("levelId").value = ""
        updateDisplay()
    } else {
        sourceHintField.value = levelsData[levelNumber]["sourcehint"]
        answerField.value = levelsData[levelNumber]["answer"]
        inputEl.value = levelsData[levelNumber]["markup"]
        try {
            const raw = levelsData[levelNumber]["walkthrough"] || '';
            let parts = [];
            try {
                const parsed = JSON.parse(raw || 'null');
                if (Array.isArray(parsed)) parts = parsed.map(p=>p+"");
            } catch(e) {
                if (raw && String(raw).trim() !== '') parts = [String(raw)];
            }
            const walkthroughPartsContainer = document.getElementById('walkthroughParts');
            if (walkthroughPartsContainer) walkthroughPartsContainer.innerHTML = '';
            if (parts.length === 0) parts = [''];
            parts.slice(0,10).forEach((p,i)=>{
                const ta = document.createElement('textarea');
                ta.className = 'walkthrough-part form-input';
                ta.setAttribute('data-index', String(i));
                ta.placeholder = 'Walkthrough part ' + String(i);
                ta.style.minHeight = '120px';
                ta.style.resize = 'vertical';
                ta.value = p || '';
                if (walkthroughPartsContainer) walkthroughPartsContainer.appendChild(ta);
            });
        } catch(e) {}
        popupContainer.style.display = 'flex'
        levelName.innerText = "Level " + String(levelNumber)
        document.getElementById("levelId").value = String(levelNumber)
        updateDisplay()
    }
}

function closePopup() {
    popupContainer.style.display = 'none';
}

function submitForm() {
    const sourceHint = sourceHintField.value.trim();
    const answer = answerField.value.trim();
    let walkthrough = '';
    try {
        const walkthroughPartsContainer = document.getElementById('walkthroughParts');
        const partsEls = walkthroughPartsContainer ? Array.from(walkthroughPartsContainer.querySelectorAll('.walkthrough-part')) : [];
        const parts = partsEls.map(el=>String(el.value||'').trim()).filter(x=>x!=='').slice(0,10);
        walkthrough = JSON.stringify(parts);
    } catch(e) { walkthrough = JSON.stringify([]); }
    var levelId = document.getElementById("levelId").value.trim()
    if (/^[0-9]+$/.test(levelId)) {
        levelId = "cryptic-" + levelId
    }
    if (levelId === "") {
        levelId = "cryptic-0"
    }
    fetch("/set_level?source=" + encodeURIComponent(sourceHint) + "&answer=" + encodeURIComponent(answer) + "&markup=" + encodeURIComponent(inputEl.value.trim()) + "&walkthrough=" + encodeURIComponent(walkthrough) + "&levelid=" + String(levelId)).then((x) => {
        window.location = "/admin"
    })
}

function addWalkthroughPart() {
    try {
        const walkthroughPartsContainer = document.getElementById('walkthroughParts');
        const existing = walkthroughPartsContainer ? walkthroughPartsContainer.querySelectorAll('.walkthrough-part') : [];
        if (existing.length >= 10) return;
        const i = existing.length;
        const ta = document.createElement('textarea');
        ta.className = 'walkthrough-part form-input';
        ta.setAttribute('data-index', String(i));
        ta.placeholder = 'Walkthrough part ' + String(i);
        ta.style.minHeight = '120px';
        ta.style.resize = 'vertical';
        if (walkthroughPartsContainer) walkthroughPartsContainer.appendChild(ta);
    } catch(e) {}
}

function clearWalkthroughParts() {
    try {
        const walkthroughPartsContainer = document.getElementById('walkthroughParts');
        if (!walkthroughPartsContainer) return;
        const existing = walkthroughPartsContainer.querySelectorAll('.walkthrough-part');
        if (!existing || existing.length <= 1) return;
        const last = existing[existing.length - 1];
        last.parentNode.removeChild(last);
    } catch(e) {}
}

if (addWalkthroughPartBtn) addWalkthroughPartBtn.addEventListener('click', addWalkthroughPart);
else window.addEventListener('load', function(){ const b = document.getElementById('addWalkthroughPartBtn'); if (b) b.addEventListener('click', addWalkthroughPart); });
if (clearWalkthroughPartsBtn) clearWalkthroughPartsBtn.addEventListener('click', clearWalkthroughParts);
else window.addEventListener('load', function(){ const b = document.getElementById('clearWalkthroughPartsBtn'); if (b) b.addEventListener('click', clearWalkthroughParts); });

function deleteLevel() {
    var levelId = document.getElementById("levelId").value.trim()
    if (/^[0-9]+$/.test(levelId)) {
        levelId = "cryptic-" + levelId
    }
    if (levelId === "") {
        levelId = "cryptic-0"
    }
    fetch("/delete_level?level=" + encodeURIComponent(levelId)).then(() => {
        window.location = "/admin"
    })
}

document.addEventListener('click', async function(e) {
    const t = e.target && e.target.closest ? e.target.closest('.toggle-leads') : null;
    if (t) {
        e.stopPropagation();
        e.preventDefault();
        const lvl = t.getAttribute('data-level');
        const enable = t.classList.contains('on') ? false : true;
        try {
            const res = await fetch('/api/admin/levels/leads', {method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({action:'set', level: lvl, enabled: enable})});
            if (res && res.ok) {
                if (enable) {
                    t.classList.remove('off'); t.classList.add('on'); t.textContent = 'On';
                    if (notyf) notyf.success('Leads enabled for ' + lvl)
                } else {
                    t.classList.remove('on'); t.classList.add('off'); t.textContent = 'Off';
                    if (notyf) notyf.success('Leads disabled for ' + lvl)
                }
            } else {
                if (notyf) notyf.error('Failed to update leads')
            }
        } catch (err) {
            if (notyf) notyf.error('Failed to update leads')
        }
    }
}, true);

const _turnOnAll = document.getElementById('turnOnAllLeads')
if (_turnOnAll) _turnOnAll.addEventListener('click', async ()=>{
    try {
        const res = await fetch('/api/admin/levels/leads', {method:'POST', credentials:'same-origin', headers:{'Content-Type':'application/json'}, body: JSON.stringify({action:'all', enabled:true})});
        if (res && res.ok) {
            if (notyf) notyf.success('Turned on leads for all levels')
            setTimeout(()=> window.location = '/admin', 350)
        } else {
            if (notyf) notyf.error('Failed to turn on all leads')
        }
    } catch(e) { if (notyf) notyf.error('Failed to turn on all leads') }
})
const _turnOffAll = document.getElementById('turnOffAllLeads')
if (_turnOffAll) _turnOffAll.addEventListener('click', async ()=>{
    try {
        const res = await fetch('/api/admin/levels/leads', {method:'POST', credentials:'same-origin', headers:{'Content-Type':'application/json'}, body: JSON.stringify({action:'all', enabled:false})});
        if (res && res.ok) {
            if (notyf) notyf.success('Turned off leads for all levels')
            setTimeout(()=> window.location = '/admin', 350)
        } else {
            if (notyf) notyf.error('Failed to turn off all leads')
        }
    } catch(e) { if (notyf) notyf.error('Failed to turn off all leads') }
})
const _toggleAI = document.getElementById('toggleAILeads')
if (_toggleAI) {
    let currentAILeads = true
    async function refreshAILeadsButton() {
        try {
            const resp = await fetch('/api/messages', { credentials: 'same-origin' })
            if (resp && resp.ok) {
                const dd = await resp.json().catch(()=>null)
                if (dd && typeof dd.ai_leads !== 'undefined') currentAILeads = !!dd.ai_leads
            }
        } catch (e) {}
        _toggleAI.textContent = currentAILeads ? 'AI Leads: ON' : 'AI Leads: OFF'
        _toggleAI.classList.toggle('on', currentAILeads)
        _toggleAI.classList.toggle('off', !currentAILeads)
    }
    refreshAILeadsButton()
    _toggleAI.addEventListener('click', async ()=>{
        const enable = !currentAILeads
        try {
            const res = await fetch('/api/admin/ai_leads', {method:'POST', credentials:'same-origin', headers:{'Content-Type':'application/json'}, body: JSON.stringify({enabled: enable})})
            if (res && res.ok) {
                currentAILeads = enable
                if (notyf) notyf.success('AI leads ' + (enable ? 'enabled' : 'disabled'))
                _toggleAI.textContent = currentAILeads ? 'AI Leads: ON' : 'AI Leads: OFF'
                setTimeout(()=> window.location = '/admin', 350)
            } else {
                if (notyf) notyf.error('Failed to update AI leads')
            }
        } catch(e) { if (notyf) notyf.error('Failed to update AI leads') }
    })
}

async function fetchAdminUsers() {
    try {
        const res = await fetch('/api/admin/users', {credentials: 'same-origin'})
        if (!res.ok) return []
        const list = await res.json().catch(()=>[])
        return Array.isArray(list) ? list : []
    } catch(e) { return [] }
}

function renderAdminUser(u) {
    const email = u.email || ''
    const name = u.name || ''
    const cryptic = (typeof u.cryptic === 'number') ? u.cryptic : 0
    const ctf = (typeof u.ctf === 'number') ? u.ctf : 0
    const el = document.createElement('div')
    el.style.padding = '12px'
    el.style.borderRadius = '8px'
    el.style.background = 'rgba(255,255,255,0.02)'
    el.style.display = 'flex'
    el.style.justifyContent = 'space-between'
    el.style.alignItems = 'center'
    el.innerHTML = `
        <div style="flex:1">
            <div style="font-size:14px;color:rgba(255,255,255,0.9);font-weight:600">${escapeHtml(name) || escapeHtml(email)}</div>
            <div style="font-size:12px;color:rgba(255,255,255,0.6)">${escapeHtml(email)}</div>
            <div style="margin-top:6px;font-size:13px;color:rgba(255,255,255,0.8)">Current Cryptic Level: ${cryptic} &nbsp; • &nbsp; Current CTF Level: ${ctf}</div>
        </div>
        <div style="display:flex;flex-direction:column;gap:6px;margin-left:12px">
            <button class="btn-primary user-reset-cryptic" data-email="${escapeHtml(email)}">Reset cryptic lvl</button>
            <button class="btn-primary user-reset-ctf" data-email="${escapeHtml(email)}" style="background:#444">Reset CTF lvl</button>
            <button class="btn-primary user-delete" data-email="${escapeHtml(email)}" style="background:#7a1a1a">Delete user</button>
        </div>
    `
    return el
}

async function reloadAdminUsers() {
    const container = document.getElementById('adminUsersList')
    if (!container) return
    container.innerHTML = ''
    const list = await fetchAdminUsers()
    if (!list || list.length === 0) {
        container.innerHTML = '<div style="color:rgba(255,255,255,0.6)">No users found</div>'
        return
    }
    for (const u of list) {
        container.appendChild(renderAdminUser(u))
    }
}

async function postAdminUserAction(email, action) {
    try {
        const res = await fetch('/api/admin/user', {method: 'POST', credentials: 'same-origin', headers: {'Content-Type':'application/json'}, body: JSON.stringify({email: email, action: action})})
        return res && res.ok
    } catch(e) { return false }
}

function showAdminConfirm(message) {
    return new Promise((resolve) => {
        const modal = document.getElementById('adminConfirmModal')
        const msg = document.getElementById('adminConfirmMessage')
        const ok = document.getElementById('adminConfirmOk')
        const cancel = document.getElementById('adminConfirmCancel')
        if (!modal || !msg || !ok || !cancel) return resolve(false)
        msg.innerText = message
        modal.style.display = 'flex'
        const cleanup = () => {
            modal.style.display = 'none'
            ok.removeEventListener('click', onOk)
            cancel.removeEventListener('click', onCancel)
        }
        const onOk = () => { cleanup(); resolve(true) }
        const onCancel = () => { cleanup(); resolve(false) }
        ok.addEventListener('click', onOk)
        cancel.addEventListener('click', onCancel)
    })
}

document.addEventListener('click', async function(e){
    const t = e.target
    if (t && t.classList) {
        if (t.classList.contains('user-reset-cryptic')) {
            const email = t.getAttribute('data-email')
            if (!email) return
            t.disabled = true
            const ok = await postAdminUserAction(email, 'reset_cryptic')
            t.disabled = false
            if (ok) { if (notyf) notyf.success('Reset cryptic level'); reloadAdminUsers() } else { if (notyf) notyf.error('Failed') }
        } else if (t.classList.contains('user-reset-ctf')) {
            const email = t.getAttribute('data-email')
            if (!email) return
            t.disabled = true
            const ok = await postAdminUserAction(email, 'reset_ctf')
            t.disabled = false
            if (ok) { if (notyf) notyf.success('Reset CTF level'); reloadAdminUsers() } else { if (notyf) notyf.error('Failed') }
        } else if (t.classList.contains('user-delete')) {
            const email = t.getAttribute('data-email')
            if (!email) return
            const confirmed = await showAdminConfirm('Delete user ' + email + '?')
            if (!confirmed) return
            t.disabled = true
            const ok = await postAdminUserAction(email, 'delete')
            t.disabled = false
            if (ok) { if (notyf) notyf.success('Deleted user'); reloadAdminUsers() } else { if (notyf) notyf.error('Failed') }
        }
    }
})

window.addEventListener('load', ()=>{ reloadAdminUsers() })