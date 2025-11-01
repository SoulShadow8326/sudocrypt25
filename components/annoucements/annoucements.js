const annToggle = document.getElementById('announcementsToggleBtn')
const annPopup = document.getElementById('announcementsPopup')
const annContainer = document.getElementById('announcementsContainer')
const annCircle = document.getElementById('announcementsCircle')
const annTemplate = typeof announcement_Item !== 'undefined' ? announcement_Item : ''
function renderAnnouncement(a) {
    return annTemplate.replace('{time}', a.time || '').replace('{announcement}', (a.text || ''))
}
let lastAnnChecksum = ''
async function fetchAnnouncementsChecksum() {
    try {
        const url = lastAnnChecksum ? ('/api/announcements?checksum=' + encodeURIComponent(lastAnnChecksum)) : '/api/announcements'
        const res = await fetch(url, { credentials: 'same-origin' })
        if (res.status === 304) return lastAnnChecksum
        if (!res.ok) return null
        const list = await res.json().catch(() => [])
        if (!Array.isArray(list)) return null
        const s = list.map(i => (i.id || '') + '|' + (i.text || '') + '|' + (i.time || '')).join('::')
        let h = 0
        for (let i = 0; i < s.length; i++) h = ((h << 5) - h) + s.charCodeAt(i) | 0
        return String(h)
    } catch (e) {
        return null
    }
}
async function loadAnnouncements() {
    if (!annContainer) return
    annContainer.innerHTML = ''
    try {
        const res = await fetch('/api/announcements')
        if (!res.ok) throw new Error('no api')
        const list = await res.json().catch(() => [])
        if (!Array.isArray(list) || list.length === 0) {
            document.querySelector('.announcements-empty') && (document.querySelector('.announcements-empty').style.display = '')
            return
        }
        for (const item of list) {
            const html = renderAnnouncement(item)
            const div = document.createElement('div')
            div.innerHTML = html
            annContainer.appendChild(div.firstElementChild)
        }
        document.querySelector('.announcements-empty') && (document.querySelector('.announcements-empty').style.display = 'none')
    } catch (err) {
        const sample = [{ time: 'just now', text: 'Welcome to Sudocrypt 2025' }]
        for (const item of sample) {
            const div = document.createElement('div')
            div.innerHTML = renderAnnouncement(item)
            annContainer.appendChild(div.firstElementChild)
        }
    }
}
annToggle && annToggle.addEventListener('click', (e) => {
    if (!annPopup) return
    const open = annPopup.classList.toggle('visible')
    annPopup.setAttribute('aria-hidden', (!open).toString())
    if (open) {
        annCircle && annCircle.classList.add('active')
        annToggle && annToggle.classList.remove('notify')
        loadAnnouncements()
    } else {
        annCircle && annCircle.classList.remove('active')
    }
})

async function pollAnnouncements() {
    try {
        const cs = await fetchAnnouncementsChecksum()
        if (cs && lastAnnChecksum && cs !== lastAnnChecksum) {
            const isOpen = annPopup && annPopup.classList.contains('visible')
            if (!isOpen) {
                annToggle && annToggle.classList.add('notify')
                annCircle && annCircle.classList.add('active')
            }
        }
        if (cs) lastAnnChecksum = cs
    } catch (e) {
    }
    setTimeout(pollAnnouncements, 15000)
}

pollAnnouncements()
