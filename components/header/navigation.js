document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.getAttribute('data-action');
  if (action === 'login') {
    window.location.href = '/auth';
    return;
  }
  if (action === 'logout') {
    try {
      const res = await fetch('/logout', { method: 'GET', credentials: 'same-origin' });
      if (res.ok) {
        window.location.href = '/auth';
      } else {
        window.location.href = '/auth';
      }
    } catch (err) {
      window.location.href = '/auth';
    }
  }
});

document.addEventListener('DOMContentLoaded', () => {
  const hamburger = document.querySelector('.hamburger');
  const mobileMenu = document.getElementById('mobileMenu');
  if (!hamburger || !mobileMenu) return;

  const overlay = document.createElement('div');
  overlay.className = 'page-blur-overlay';
  document.body.appendChild(overlay);

  function openMenu(){
    hamburger.setAttribute('aria-expanded','true');
    mobileMenu.classList.add('open');
    mobileMenu.setAttribute('aria-hidden','false');
    overlay.classList.add('visible');
    document.documentElement.style.overflow = 'hidden';
  }
  function closeMenu(){
    hamburger.setAttribute('aria-expanded','false');
    mobileMenu.classList.remove('open');
    mobileMenu.setAttribute('aria-hidden','true');
    overlay.classList.remove('visible');
    document.documentElement.style.overflow = '';
  }

  hamburger.addEventListener('click', (e)=>{
    const expanded = hamburger.getAttribute('aria-expanded') === 'true';
    if (expanded) closeMenu(); else openMenu();
  });

  overlay.addEventListener('click', ()=>{ closeMenu(); });

  const closeBtn = mobileMenu.querySelector('.mobile-close');
  if (closeBtn) closeBtn.addEventListener('click', ()=> closeMenu());

  document.addEventListener('keydown', (e)=>{
    if (e.key === 'Escape') closeMenu();
  });

  mobileMenu.addEventListener('click', (e)=>{
    const link = e.target.closest('a');
    if (link) closeMenu();
  });
});
