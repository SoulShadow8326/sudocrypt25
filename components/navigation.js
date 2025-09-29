function initAnnouncements() {
    const annPopup = document.getElementById('announcementsPopup')
    const annToggle = document.getElementById('announcementsToggleBtn')
    if (!annPopup || !annToggle) return

    annToggle.addEventListener('click', (e) => {
        e.stopPropagation()
        const open = annPopup.classList.toggle('show')
        annPopup.setAttribute('aria-hidden', open ? 'false' : 'true')
        const annCircle = document.getElementById('announcementsCircle')
        if (open) annCircle && annCircle.classList.add('active')
        else annCircle && annCircle.classList.remove('active')
    })

    document.addEventListener('click', (e) => {
        if (!annPopup.contains(e.target) && !annToggle.contains(e.target)) {
            annPopup.classList.remove('show')
            annPopup.setAttribute('aria-hidden', 'true')
            const annCircle = document.getElementById('announcementsCircle')
            annCircle && annCircle.classList.remove('active')
        }
    })
}

function initActiveNav() {
    const links = Array.from(document.querySelectorAll('.nav-links a'))
    if (!links.length) return

    function setActive(href) {
        links.forEach(l => l.classList.toggle('active', l.getAttribute('href') === href))
        try { localStorage.setItem('activeNav', href) } catch (e) {}
    }

    const path = window.location.pathname
    const saved = (() => { try { return localStorage.getItem('activeNav') } catch(e){ return null } })()
    const byPath = links.find(l => l.getAttribute('href') === path)
    if (byPath) setActive(byPath.getAttribute('href'))
    else if (saved) setActive(saved)
    else setActive('/')

    links.forEach(l => {
        l.addEventListener('click', (e) => {
            const href = l.getAttribute('href')
            setActive(href)
        })
    })
}

document.addEventListener('DOMContentLoaded', () => {
    initAnnouncements()
    initActiveNav()
})
