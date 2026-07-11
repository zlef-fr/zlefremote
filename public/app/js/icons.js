// Shared line-icons for the remote client (no emoji). Mirrors the landing set.
const ZRIcon = (() => {
  const P = {
    cursor: '<path d="M3 3l7.2 17.4 2.3-7 7-2.3z"/>',
    keyboard: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M6 9h.01M10 9h.01M14 9h.01M18 9h.01M7 13h10"/>',
    sliders: '<line x1="21" x2="14" y1="6" y2="6"/><line x1="10" x2="3" y1="6" y2="6"/><line x1="21" x2="12" y1="12" y2="12"/><line x1="8" x2="3" y1="12" y2="12"/><line x1="21" x2="16" y1="18" y2="18"/><line x1="12" x2="3" y1="18" y2="18"/><circle cx="12" cy="6" r="2"/><circle cx="10" cy="12" r="2"/><circle cx="14" cy="18" r="2"/>',
    vol: '<path d="M11 5 6 9H3v6h3l5 4z"/><path d="M15.5 8.5a5 5 0 0 1 0 7"/><path d="M18.5 6a9 9 0 0 1 0 12"/>',
    voldown: '<path d="M11 5 6 9H3v6h3l5 4z"/><path d="M15.5 8.5a5 5 0 0 1 0 7"/>',
    mute: '<path d="M11 5 6 9H3v6h3l5 4z"/><path d="m16 9 5 6M21 9l-5 6"/>',
    play: '<path d="M6 4l14 8-14 8z"/><path d="M6 4v16"/>',
    prev: '<path d="M19 5 9 12l10 7z"/><path d="M5 5v14"/>',
    next: '<path d="M5 5l10 7L5 19z"/><path d="M19 5v14"/>',
    lock: '<rect x="4" y="10.5" width="16" height="10" rx="2"/><path d="M8 10.5V7a4 4 0 0 1 8 0v3.5"/>',
    warn: '<path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/><path d="M12 9v4M12 17h.01"/>',
    plug: '<path d="M12 22v-5M9 8V2M15 8V2M7 8h10v3a5 5 0 0 1-10 0z"/>',
    monitor: '<rect x="2" y="3.5" width="20" height="13" rx="2"/><path d="M8 21h8M12 16.5V21"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    sundim: '<circle cx="12" cy="12" r="4"/><path d="M12 4h.01M12 20h.01M6.3 6.3h.01M17.7 17.7h.01M4 12h.01M20 12h.01M6.3 17.7h.01M17.7 6.3h.01"/>',
  };
  function svg(name, sw) {
    return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${sw || 1.7}" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${P[name] || ''}</svg>`;
  }
  // hydrate any <span data-ico="name"> already in the DOM
  function hydrate(root = document) {
    root.querySelectorAll('[data-ico]').forEach((el) => { el.innerHTML = svg(el.dataset.ico); });
  }
  return { svg, hydrate };
})();
ZRIcon.hydrate();
