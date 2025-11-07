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
            // clear existing parts
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