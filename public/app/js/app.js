(() => {
  const t = ZRI18n.t;
  const $ = (id) => document.getElementById(id);
  const vibrate = (ms) => { try { navigator.vibrate && navigator.vibrate(ms); } catch {} };

  // settings (persisted per device)
  const cfg = {
    sensitivity: parseFloat(localStorage.getItem('zr_sens') || '1.6'),
    scrollSpeed: parseFloat(localStorage.getItem('zr_scroll') || '1.0'),
    natural: localStorage.getItem('zr_natural') !== '0',
  };
  const getCfg = () => cfg;

  let paired = false;
  const stickyMods = new Set();

  // ── small toast ────────────────────────────────────────────────────────────
  let toastTimer = null;
  function toast(msg) {
    const el = $('toast'); el.textContent = msg; el.hidden = false;
    requestAnimationFrame(() => el.classList.add('show'));
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { el.classList.remove('show'); setTimeout(() => (el.hidden = true), 300); }, 3200);
  }
  const goHome = () => { location.href = '/r/'; };

  // ── static labels ──────────────────────────────────────────────────────
  $('padHint').textContent = t('hint_pad');
  $('tlPad').textContent = t('tab_pad');
  $('tlKeys').textContent = t('tab_keys');
  $('tlMedia').textContent = t('tab_media');
  $('bLeft').textContent = t('btn_left');
  $('bRight').textContent = t('btn_right');
  $('bMid').textContent = t('btn_mid');
  $('bDrag').textContent = t('btn_drag');
  $('typer').placeholder = t('type_ph');
  $('setTitle').textContent = t('settings');
  $('lblSens').textContent = t('sensitivity');
  $('lblScroll').textContent = t('scroll_speed');
  $('lblNatural').textContent = t('natural_scroll');
  $('secureNote').textContent = '🔒 ' + t('secure');
  $('sens').value = cfg.sensitivity; $('scrollSp').value = cfg.scrollSpeed; $('natural').checked = cfg.natural;

  // ── command sender ─────────────────────────────────────────────────────
  const send = (cmd) => { if (paired) ZRConn.send(cmd); };

  // lock-screen media card (Media Session API) — drives the PC's media keys
  ZRMedia.config({ send, t });

  // ── connection state machine ───────────────────────────────────────────
  const ov = $('overlay'), ovTitle = $('ovTitle'), ovText = $('ovText'),
        ovBtn = $('ovBtn'), ovHome = $('ovHome'), ovSpin = $('ovSpin'), ovIcon = $('ovIcon'),
        dot = $('dot'), stLabel = $('stLabel');

  function showOverlay(title, text, opts = {}) {
    ov.hidden = false;
    ovTitle.textContent = title || ''; ovTitle.hidden = !title;
    ovText.textContent = text || '';
    ovSpin.hidden = !!opts.icon; ovIcon.hidden = !opts.icon;
    if (opts.icon) ovIcon.innerHTML = ZRIcon.svg(opts.icon, 1.5);
    if (opts.btn) { ovBtn.hidden = false; ovBtn.textContent = opts.btn; ovBtn.onclick = opts.onClick; }
    else ovBtn.hidden = true;
    // the "Devices" escape hatch shows whenever we're not actively connecting
    ovHome.hidden = !opts.home;
    if (opts.home) { ovHome.textContent = t('back_home'); ovHome.onclick = goHome; }
  }
  function hideOverlay() { ov.hidden = true; }
  function setStatus(label, cls) { stLabel.textContent = label; dot.className = 'dot ' + (cls || ''); }

  ZRConn.on('state', (s) => {
    switch (s) {
      case 'connecting': showOverlay(t('connecting'), '', {}); setStatus(t('connecting'), 'warn'); break;
      case 'linked': showOverlay(t('linking'), '', {}); setStatus(t('linking'), 'warn'); break;
      case 'reconnecting': setStatus(t('reconnecting'), 'warn'); break;
      case 'nokey': showOverlay('', t('nokey'), { icon: 'lock' }); setStatus(t('closed'), 'bad'); break;
    }
  });
  ZRConn.on('error', (e) => {
    // a saved (persistent) device whose room is gone = the computer is offline
    if (e === 'no_such_room' && ZRConn.isPersistent()) {
      showOverlay(t('offline_title'), t('offline_sub'),
        { icon: 'plug', btn: t('try_again'), onClick: () => location.reload(), home: true });
    } else {
      const msg = e === 'no_such_room' ? t('err_room') : e === 'room_full' ? t('err_full') : t('err_connect');
      showOverlay('', msg, { icon: 'warn', btn: t('reconnect'), onClick: () => location.reload(), home: true });
    }
    setStatus(t('closed'), 'bad'); paired = false; ZRMedia.stop();
  });
  ZRConn.on('closed', (reason) => {
    showOverlay('', reason === 'host_left' ? t('closed_host') : t('closed'),
      { icon: 'plug', btn: t('reconnect'), onClick: () => location.reload(), home: true });
    setStatus(t('closed'), 'bad'); paired = false; ZRMedia.stop();
  });

  // host → client commands (the welcome handshake)
  ZRConn.on('cmd', (c) => {
    if (c.t === 'welcome') {
      paired = true; ZRConn.markPaired();
      hideOverlay();
      setStatus(t('paired'), 'ok');
      $('hostName').textContent = c.name || '';
      vibrate(20);
      ZRMedia.start(c.name || '');
      // remember persistent computers so they reconnect from Home in one tap
      if (ZRConn.isPersistent()) {
        ZRHome.saveFromWelcome({
          name: c.name, os: c.os, key: ZRConn.getKeyB64(),
          room: ZRConn.getRoom(), persistent: true,
        });
        toast(t('saved_toast'));
      } else {
        toast(t('ephemeral_note'));
      }
    }
  });

  // ── trackpad ───────────────────────────────────────────────────────────
  ZRInput.attach($('pad'), (cmd) => { send(cmd); if (cmd.t === 'click') vibrate(8); }, getCfg);

  // mouse buttons
  document.querySelectorAll('.mbtn[data-click]').forEach((b) => {
    b.addEventListener('click', () => { send({ t: 'click', b: b.dataset.click }); vibrate(8); });
  });
  let dragLock = false;
  $('bDrag').addEventListener('click', () => {
    dragLock = !dragLock;
    $('bDrag').classList.toggle('active', dragLock);
    send({ t: dragLock ? 'down' : 'up', b: 'left' });
    vibrate(12);
  });

  // ── tabs ───────────────────────────────────────────────────────────────
  const views = { pad: $('viewPad'), keys: $('viewKeys'), media: $('viewMedia') };
  document.querySelectorAll('.tab').forEach((tab) => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.tab').forEach((x) => x.classList.remove('active'));
      tab.classList.add('active');
      const v = tab.dataset.view;
      for (const k in views) views[k].hidden = (k !== v);
      if (v === 'keys') setTimeout(() => $('typer').focus(), 60);
    });
  });

  // ── modifiers ──────────────────────────────────────────────────────────
  const MODS = [['ctrl', t('ctrl')], ['alt', t('alt')], ['shift', t('shift')], ['meta', t('win')]];
  const modRow = $('modRow');
  MODS.forEach(([k, label]) => {
    const b = document.createElement('button');
    b.className = 'modkey'; b.textContent = label; b.dataset.mod = k;
    b.addEventListener('click', () => {
      if (stickyMods.has(k)) stickyMods.delete(k); else stickyMods.add(k);
      b.classList.toggle('active', stickyMods.has(k));
      vibrate(8);
    });
    modRow.appendChild(b);
  });
  function clearMods() {
    stickyMods.clear();
    modRow.querySelectorAll('.modkey').forEach((b) => b.classList.remove('active'));
  }
  // press a key (special name or character) with any sticky modifiers
  function pressKey(k) {
    const mods = [...stickyMods];
    send({ t: 'key', k, mods });
    if (mods.length) clearMods();
    vibrate(8);
  }

  // ── special keys ─────────────────────────────────────────────────────────
  const SPECS = [
    [t('esc'), 'escape'], [t('tab'), 'tab'], [t('back'), 'backspace'], [t('enter'), 'enter'],
    ['↑', 'up'], ['↓', 'down'], ['←', 'left'], ['→', 'right'],
    ['Home', 'home'], ['End', 'end'], ['PgUp', 'pageup'], ['PgDn', 'pagedown'],
    ['Del', 'delete'], [t('space'), 'space'], ['F5', 'f5'], ['F11', 'f11'],
  ];
  const specKeys = $('specKeys');
  SPECS.forEach(([label, k]) => {
    const b = document.createElement('button');
    b.className = 'speckey'; b.textContent = label;
    b.addEventListener('click', () => pressKey(k));
    specKeys.appendChild(b);
  });

  // ── typing ───────────────────────────────────────────────────────────────
  const typer = $('typer');
  typer.addEventListener('beforeinput', (e) => {
    // Enter / Backspace map to host key presses, not literal text
    if (e.inputType === 'insertLineBreak' || e.inputType === 'insertParagraph') {
      e.preventDefault(); pressKey('enter'); return;
    }
    if (e.inputType === 'deleteContentBackward') {
      e.preventDefault(); pressKey('backspace'); return;
    }
  });
  typer.addEventListener('input', (e) => {
    if (e.inputType === 'insertText' || e.inputType === 'insertCompositionText') {
      const s = e.data;
      if (s == null) return;
      if (stickyMods.size && s.length === 1) { pressKey(s); }
      else send({ t: 'text', s });
      typer.value = ''; // keep the field from drifting out of sync
    }
  });
  // hardware keyboards (desktop test): send arrows etc.
  typer.addEventListener('keydown', (e) => {
    const map = { Enter: 'enter', Backspace: 'backspace', Tab: 'tab', Escape: 'escape',
      ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right', Delete: 'delete' };
    if (map[e.key]) { e.preventDefault(); pressKey(map[e.key]); }
  });

  // ── media ──────────────────────────────────────────────────────────────
  const MEDIA = [
    ['voldown', t('vol_down'), 'voldown'], ['vol', t('vol_up'), 'volup'], ['mute', t('mute'), 'mute'],
    ['play', t('play'), 'playpause'], ['prev', t('prev'), 'prev'], ['next', t('next'), 'next'],
  ];
  const mg = $('mediaGrid');
  MEDIA.forEach(([ic, label, k]) => {
    const b = document.createElement('button');
    b.className = 'mediakey';
    b.innerHTML = `<span class="mk-ic">${ZRIcon.svg(ic)}</span><span class="mk-l">${label}</span>`;
    b.addEventListener('click', () => { send({ t: 'media', k }); vibrate(10); });
    mg.appendChild(b);
  });

  // ── settings sheet ───────────────────────────────────────────────────────
  $('gear').addEventListener('click', () => { $('sheet').hidden = false; });
  $('setClose').addEventListener('click', () => { $('sheet').hidden = true; });
  $('sheet').addEventListener('click', (e) => { if (e.target === $('sheet')) $('sheet').hidden = true; });
  $('sens').addEventListener('input', (e) => { cfg.sensitivity = parseFloat(e.target.value); localStorage.setItem('zr_sens', e.target.value); });
  $('scrollSp').addEventListener('input', (e) => { cfg.scrollSpeed = parseFloat(e.target.value); localStorage.setItem('zr_scroll', e.target.value); });
  $('natural').addEventListener('change', (e) => { cfg.natural = e.target.checked; localStorage.setItem('zr_natural', e.target.checked ? '1' : '0'); });

  // lock-screen controls toggle — only shown where the browser supports it
  if (ZRMedia.supported()) {
    $('lockctlRow').hidden = false;
    $('lblLockctl').textContent = t('lockctl');
    $('lockctlNote').textContent = t('lockctl_note');
    $('lockctl').checked = ZRMedia.enabled();
    $('lockctl').addEventListener('change', (e) => { ZRMedia.setEnabled(e.target.checked); });
  }

  // keep the screen awake while controlling
  let wakeLock = null;
  async function keepAwake() { try { if ('wakeLock' in navigator) wakeLock = await navigator.wakeLock.request('screen'); } catch {} }

  // ── mode routing: home (device picker) vs control ──────────────────────────
  $('homeBtn').addEventListener('click', goHome);

  function showControl() {
    document.body.classList.add('mode-control');
    $('topbar').hidden = false; $('stage').hidden = false; $('tabs').hidden = false;
    document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible') keepAwake(); });
    keepAwake();
    ZRConn.start();
  }
  function showHomeScreen() {
    document.body.classList.add('mode-home');
    $('viewHome').hidden = false;
    ZRHome.init();
  }

  if (ZRConn.hasTarget()) showControl();
  else showHomeScreen();
})();
