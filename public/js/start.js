// /start — progressive enhancement for the "send to desktop" actions.
(function () {
  const shareBtn = document.getElementById('shareBtn');
  const copyBtn = document.getElementById('copyBtn');

  // Native share sheet (phones) — only show if the browser supports it.
  if (shareBtn && navigator.share) {
    shareBtn.hidden = false;
    shareBtn.addEventListener('click', () => {
      navigator.share({
        title: 'ZlefRemote',
        text: shareBtn.dataset.msg,
        url: shareBtn.dataset.url,
      }).catch(() => {});
    });
  }

  if (copyBtn) {
    const label = copyBtn.querySelector('span');
    const original = label ? label.textContent : '';
    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(copyBtn.dataset.url);
      } catch {
        const ta = document.createElement('textarea');
        ta.value = copyBtn.dataset.url; document.body.appendChild(ta); ta.select();
        try { document.execCommand('copy'); } catch {}
        ta.remove();
      }
      copyBtn.classList.add('ok');
      if (label) label.textContent = copyBtn.dataset.copied;
      setTimeout(() => { copyBtn.classList.remove('ok'); if (label) label.textContent = original; }, 1800);
    });
  }
})();
