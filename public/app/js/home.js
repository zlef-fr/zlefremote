// Home screen — the saved/new device picker. This is the PWA's entry point:
// a list of remembered computers you reconnect to in one tap, plus an "add a
// device" flow (scan the agent QR, or paste its pairing link).
//
// A device is reconnectable only when its agent runs with --remember: the agent
// then derives a STABLE relay room from its persisted key, and the pairing URL
// carries &p=1. We store just the key (+ os/name) and re-derive the room on each
// reconnect, so a saved phone always finds the computer at the same address.
const ZRHome = (() => {
  const t = ZRI18n.t;
  const $ = (id) => document.getElementById(id);
  const KEY = 'zr_devices';

  // ── store ────────────────────────────────────────────────────────────────
  function load() {
    try { const a = JSON.parse(localStorage.getItem(KEY) || '[]'); return Array.isArray(a) ? a : []; }
    catch { return []; } // corrupted localStorage entry — start fresh
  }
  function save(list) { localStorage.setItem(KEY, JSON.stringify(list)); }
  function list() { return load().sort((a, b) => (b.lastUsed || 0) - (a.lastUsed || 0)); }
  function devId(d) { return d.persistent ? 'p:' + d.room : 'e:' + d.room + ':' + (d.key || '').slice(0, 8); }

  function upsert(dev) {
    const all = load();
    const id = devId(dev);
    const i = all.findIndex((d) => devId(d) === id);
    if (i >= 0) all[i] = Object.assign(all[i], dev, { lastUsed: Date.now() });
    else all.push(Object.assign({ addedAt: Date.now(), lastUsed: Date.now() }, dev));
    save(all);
    return id;
  }
  function removeById(id) { save(load().filter((d) => devId(d) !== id)); }
  function renameById(id, name) {
    const all = load(); const d = all.find((x) => devId(x) === id);
    if (d) { d.name = name; save(all); }
  }
  function touchById(id) {
    const all = load(); const d = all.find((x) => devId(x) === id);
    if (d) { d.lastUsed = Date.now(); save(all); }
  }

  // ── helpers ────────────────────────────────────────────────────────────────
  const OS = { linux: '🐧', windows: '🪟', win: '🪟', darwin: '🍎', mac: '🍎' };
  function osGlyph(os) { return OS[(os || '').toLowerCase()] || '🖥'; }
  function timeAgo(ms) {
    if (!ms) return '';
    const s = Math.floor((Date.now() - ms) / 1000);
    if (s < 45) return t('just_now');
    const m = Math.floor(s / 60); if (m < 60) return m + ' min';
    const h = Math.floor(m / 60); if (h < 24) return h + ' h';
    const d = Math.floor(h / 24); return d + (ZRI18n.lang === 'fr' ? ' j' : ' d');
  }

  // build a device URL and navigate (re-derives the stable room from the key)
  async function connectTo(dev) {
    let room = dev.room;
    if (dev.persistent && dev.key) {
      try { room = await ZRCrypto.deriveRoom(dev.key); } catch { /* bad key in storage — fall back to stored room code */ }
    }
    touchById(devId(dev));
    location.href = `/r/${room}#k=${dev.key}` + (dev.persistent ? '&p=1' : '');
  }

  // ── render ─────────────────────────────────────────────────────────────────
  let menuOpenId = null;
  function render() {
    $('homeTitle').textContent = t('home_title');
    $('homeSub').textContent = t('home_sub');
    $('addLabel').textContent = t('add_device');

    const wrap = $('devList');
    const devs = list();
    wrap.innerHTML = '';

    if (!devs.length) {
      const e = document.createElement('div');
      e.className = 'dev-empty';
      const ic = document.createElement('div');
      ic.className = 'empty-ic';
      ic.innerHTML = ZRIcon.svg('cursor', 1.4); // static SVG path only
      const h2 = document.createElement('h2');
      h2.textContent = t('empty_title');
      const p = document.createElement('p');
      p.textContent = t('empty_sub');
      e.appendChild(ic); e.appendChild(h2); e.appendChild(p);
      wrap.appendChild(e);
    }

    devs.forEach((d) => {
      const id = devId(d);
      const card = document.createElement('div');
      card.className = 'devcard reveal-in';

      // Main connect button — built with DOM methods so user-derived values
      // (name, os, lastUsed) are always set as text, never injected as HTML.
      const mainBtn = document.createElement('button');
      mainBtn.className = 'devmain';
      mainBtn.setAttribute('aria-label', t('connect'));

      const avSpan = document.createElement('span');
      avSpan.className = 'dev-av';
      avSpan.textContent = osGlyph(d.os);

      const metaSpan = document.createElement('span');
      metaSpan.className = 'dev-meta';

      const nameSpan = document.createElement('span');
      nameSpan.className = 'dev-name';
      nameSpan.textContent = d.name || t('unknown_device');

      const subSpan = document.createElement('span');
      subSpan.className = 'dev-sub';
      subSpan.textContent = d.lastUsed ? t('last_used') + ' · ' + timeAgo(d.lastUsed) : '';

      metaSpan.appendChild(nameSpan);
      metaSpan.appendChild(subSpan);

      const goSpan = document.createElement('span');
      goSpan.className = 'dev-go';
      goSpan.innerHTML = ZRIcon.svg('cursor', 1.5); // static SVG path only

      mainBtn.appendChild(avSpan);
      mainBtn.appendChild(metaSpan);
      mainBtn.appendChild(goSpan);

      const moreBtn = document.createElement('button');
      moreBtn.className = 'dev-more';
      moreBtn.setAttribute('aria-label', t('settings'));
      moreBtn.textContent = '⋯';

      const menu = document.createElement('div');
      menu.className = 'dev-menu';
      menu.hidden = true;

      const renameBtn = document.createElement('button');
      renameBtn.dataset.act = 'rename';
      renameBtn.textContent = t('rename');

      const removeBtn = document.createElement('button');
      removeBtn.dataset.act = 'remove';
      removeBtn.className = 'danger';
      removeBtn.textContent = t('remove');

      menu.appendChild(renameBtn);
      menu.appendChild(removeBtn);

      card.appendChild(mainBtn);
      card.appendChild(moreBtn);
      card.appendChild(menu);

      mainBtn.addEventListener('click', () => connectTo(d));
      moreBtn.addEventListener('click', (ev) => {
        ev.stopPropagation();
        const wasOpen = !menu.hidden;
        document.querySelectorAll('.dev-menu').forEach((m) => (m.hidden = true));
        menu.hidden = wasOpen;
      });
      renameBtn.addEventListener('click', () => {
        const name = prompt(t('rename_ph'), d.name || '');
        if (name != null && name.trim()) { renameById(id, name.trim()); render(); }
      });
      removeBtn.addEventListener('click', () => {
        if (confirm(t('remove_q'))) { removeById(id); render(); }
      });
      wrap.appendChild(card);
    });
  }
  document.addEventListener('click', () => document.querySelectorAll('.dev-menu').forEach((m) => (m.hidden = true)));

  // ── add device ─────────────────────────────────────────────────────────────
  let scanStream = null, scanRAF = null, detector = null;

  function openAdd() {
    $('addTitle').textContent = t('add_title');
    $('scanBtnLabel').textContent = t('add_scan');
    $('scanHint').textContent = t('add_scan_hint');
    $('pasteL').textContent = t('add_paste');
    $('pasteInput').placeholder = t('add_paste_ph');
    $('pasteGoLabel').textContent = t('add_go');
    $('addErr').hidden = true;
    $('addSheet').hidden = false;
    // hide the scan button on browsers without BarcodeDetector + camera
    const canScan = ('BarcodeDetector' in window) && navigator.mediaDevices && navigator.mediaDevices.getUserMedia;
    $('scanBtn').hidden = !canScan;
    if (!canScan) { $('scanHint').textContent = t('scan_unsupported'); }
  }
  function closeAdd() { stopScan(); $('addSheet').hidden = true; }

  function showErr(msg) { const e = $('addErr'); e.textContent = msg; e.hidden = false; }

  function parseLink(s) {
    if (!s) return null;
    s = s.trim();
    const km = s.match(/[#&]k=([A-Za-z0-9\-_]+)/);
    if (!km) return null;
    const key = km[1];
    const rm = s.match(/\/r\/([A-Za-z0-9]{4,8})/i);
    const persistent = /[#&]p=1\b/.test(s);
    return { key, room: rm ? rm[1].toUpperCase() : null, persistent };
  }

  async function go(target) {
    let room = target.room;
    if (!room && target.persistent) { try { room = await ZRCrypto.deriveRoom(target.key); } catch { /* bad key — fall through to showErr below */ } }
    if (!room) { showErr(t('add_bad')); return; }
    location.href = `/r/${room}#k=${target.key}` + (target.persistent ? '&p=1' : '');
  }

  async function startScan() {
    try {
      detector = new BarcodeDetector({ formats: ['qr_code'] });
      scanStream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
    } catch (e) {
      showErr(t('scan_denied')); return;
    }
    const v = $('scanVideo');
    v.srcObject = scanStream; await v.play().catch(() => {});
    $('scanbox').classList.add('live');
    const tick = async () => {
      if (!scanStream) return;
      try {
        const codes = await detector.detect(v);
        for (const c of codes) {
          const target = parseLink(c.rawValue || '');
          if (target) { stopScan(); return go(target); }
        }
      } catch { /* BarcodeDetector may throw on certain frames; keep scanning */ }
      scanRAF = requestAnimationFrame(tick);
    };
    tick();
  }
  function stopScan() {
    if (scanRAF) cancelAnimationFrame(scanRAF), (scanRAF = null);
    if (scanStream) { scanStream.getTracks().forEach((t) => t.stop()); scanStream = null; }
    $('scanbox').classList.remove('live');
  }

  function wireAdd() {
    $('addDevice').addEventListener('click', openAdd);
    $('addClose').addEventListener('click', closeAdd);
    $('addSheet').addEventListener('click', (e) => { if (e.target === $('addSheet')) closeAdd(); });
    $('scanBtn').addEventListener('click', startScan);
    $('pasteGo').addEventListener('click', () => {
      const target = parseLink($('pasteInput').value);
      if (!target) { showErr(t('add_bad')); return; }
      go(target);
    });
    $('pasteInput').addEventListener('keydown', (e) => { if (e.key === 'Enter') $('pasteGo').click(); });
  }

  // called by app.js after a successful pair to remember a persistent device
  function saveFromWelcome(info) {
    if (!info || !info.persistent || !info.key) return null;
    return upsert({ name: info.name || t('new_device'), os: info.os || '', key: info.key, room: info.room, persistent: true });
  }

  // ── install affordance (InstallKit) ───────────────────────────────────────
  // InstallKit runs in manual mode (no FAB). We surface a single "Install app"
  // pill in the home header, only when the app is installable and not already
  // installed — and let InstallKit render the exact per-device steps on tap.
  function setupInstall() {
    const b = $('installBtn');
    if (!b) return;
    const standalone = (matchMedia && matchMedia('(display-mode: standalone)').matches) || navigator.standalone;
    if (standalone) return; // already installed → no prompt
    b.textContent = t('install_app');
    b.onclick = () => { try { window.InstallKit && window.InstallKit.open(); } catch { /* InstallKit not yet loaded */ } };
    const reveal = () => { try { if (window.InstallKit && window.InstallKit.canInstall()) b.hidden = false; } catch { /* InstallKit not yet loaded */ } };
    reveal();
    setTimeout(reveal, 900); // InstallKit boots async (deferred script)
    try { window.InstallKit && window.InstallKit.on && window.InstallKit.on('available', reveal); } catch { /* InstallKit not yet loaded */ }
    window.addEventListener('appinstalled', () => { b.hidden = true; });
  }

  // ── lock-screen controls promo card ───────────────────────────────────────
  // Surfaced on Home so the feature is discoverable without first connecting.
  // The toggle just persists the preference (via ZRMedia); it takes effect the
  // next time you connect to a computer.
  function setupLockCard() {
    const card = $('lockCard');
    if (!card || typeof ZRMedia === 'undefined' || !ZRMedia.supported()) return;
    $('lcIcon').innerHTML = ZRIcon.svg('lock', 1.2);
    $('lcTitle').textContent = t('lockctl');
    $('lcDesc').textContent = t('lockctl_home');
    const tog = $('lcToggle');
    const sync = (on) => {
      tog.setAttribute('aria-checked', on ? 'true' : 'false');
      card.classList.toggle('on', on);
    };
    sync(ZRMedia.enabled());
    tog.addEventListener('click', () => {
      const on = tog.getAttribute('aria-checked') !== 'true';
      ZRMedia.setEnabled(on);
      sync(on);
    });
    card.hidden = false;
  }

  function init() {
    wireAdd();
    render();
    setupInstall();
    setupLockCard();
    // PWA shortcut / deep link: /r/?add=1 opens the add sheet straight away
    if (/[?&]add=1\b/.test(location.search)) openAdd();
  }

  return { init, render, saveFromWelcome, openAdd };
})();
