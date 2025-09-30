document.addEventListener('DOMContentLoaded', () => {
  const nav = document.querySelector('[data-nav]');
  if (!nav) return;

  nav.addEventListener('click', async (e) => {
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
});
