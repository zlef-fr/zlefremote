// Desktop remote — orchestration. Joins the same room as the phone client
// (ZRConn/ZRCrypto are shared), feeds frames to the view, hands input to the
// pump, and keeps the bar, toasts and clipboard in sync.
(() => {
  const t = ZRDeskI18n.t;
  const $ = (id) => document.getElementById(id);

  const el = {
    body: document.body,
    bar: $('bar'), dot: $('dot'), host: $('hostName'), stats: $('stats'),
    stage: $('stage'), canvas: $('screen'),
    idle: $('idle'), idleTitle: $('idleTitle'), idleText: $('idleText'), spin: $('spin'),
    takeover: $('takeover'), takeBtn: $('takeBtn'), takeHint: $('takeHint'),
    help: $('help'), helpList: $('helpList'), helpClose: $('helpClose'),
    helpTitle: $('helpTitle'), helpCapture: $('helpCapture'),
    toasts: $('toasts'),
    btnQuality: $('btnQuality'), lblQuality: $('lblQuality'), btnMonitor: $('btnMonitor'),
    btnClip: $('btnClip'), btnCad: $('btnCad'), btnFull: $('btnFull'),
    btnHelp: $('btnHelp'), btnQuit: $('btnQuit'),
  };

  let caps = {}, hostName = '', state = 'connecting';
  let latency = 0, pingSeq = 0, pingAt = 0;
  let clipLast = '', clipPoll = 0, clipManual = false;
  let pendingClick = null;

  // ── boot ──────────────────────────────────────────────────────────────────
  el.takeBtn.textContent = t('take');
  el.takeHint.textContent = t('take_hint');
  el.btnCad.textContent = 'Ctrl+Alt+Del';
  el.btnHelp.textContent = '?';
  el.btnQuit.textContent = t('quit');
  el.helpTitle.textContent = t('help_title');
  el.helpClose.textContent = t('ok');
  el.btnClip.textContent = t('clip');
  syncFullLabel();
  buildHelp();

  ZRDeskView.config({
    canvas: el.canvas,
    send: (c) => ZRConn.send(c),
    onStats: paintStats,
  });
  ZRDeskInput.config({
    stage: el.stage,
    send: (c) => ZRConn.send(c),
    view: ZRDeskView,
    onAction: action,
  });

  if (!ZRConn.hasTarget()) {
    showIdle(t('nokey'), '', true);
  } else {
    ZRConn.start();
  }

  ZRConn.on('state', (s) => {
    state = s;
    if (s === 'reconnecting' || s === 'closed') {
      ZRDeskInput.releaseAll();
      ZRDeskView.reset();
    }
    paintState();
  });
  ZRConn.on('error', (e) => {
    if (e === 'no_such_room') showIdle(t('noroom'), '', true);
    else if (e === 'host_left') showIdle(t('hostgone'), '', true);
  });
  ZRConn.on('closed', () => showIdle(t('closed'), '', true));
  ZRConn.on('cmd', onCmd);

  function onCmd(c) {
    switch (c.t) {
      case 'welcome': onWelcome(c); break;
      case 'f': ZRDeskView.frame(c); hideIdle(); break;
      case 'viewerr': if (c.reason === 'unsupported') showIdle(t('noscreen'), '', true); break;
      case 'pong': if (c.i === pingSeq && pingAt) { latency = Math.round(performance.now() - pingAt); pingAt = 0; } break;
      case 'clip': onRemoteClip(c.s || ''); break;
    }
  }

  function onWelcome(c) {
    ZRConn.markPaired();
    state = 'paired';
    hostName = c.name || '';
    caps = c.cap || {};
    el.host.textContent = hostName;
    document.title = hostName ? `${hostName} — ZlefRemote` : 'ZlefRemote';
    ZRDeskInput.setKeyHold(!!caps.keyhold);

    ZRDeskView.setScreens(c.screens || []);
    el.btnMonitor.hidden = (c.screens || []).length < 2;
    paintMonitor();

    if (caps.screen) { ZRDeskView.start(); showIdle(t('waiting'), '', false); }
    else showIdle(t('noscreen'), '', true);

    el.btnClip.hidden = !caps.clip;
    if (caps.clip) startClipboard();
    paintState();
    if (!pingTimer) pingTimer = setInterval(ping, 2000);
  }

  // ── connection state → bar + overlay ──────────────────────────────────────
  let pingTimer = 0;
  function ping() {
    if (state !== 'paired') return;
    pingSeq++; pingAt = performance.now();
    ZRConn.send({ t: 'ping', i: pingSeq });
  }

  function paintState() {
    const live = state === 'paired';
    el.dot.className = 'dot ' + (live ? 'live' : state === 'reconnecting' ? 'warn' : 'bad');
    if (state === 'connecting') showIdle(t('connecting'), '', false);
    else if (state === 'linked') showIdle(t('linked'), '', false);
    else if (state === 'reconnecting') showIdle(t('reconnecting'), '', false);
    paintTakeover();
    paintStats(ZRDeskView.stats());
  }

  function paintStats(s) {
    const bits = [];
    if (state === 'paired' && ZRDeskView.hasImage()) {
      bits.push(`${s.fps} fps · ${s.kbps} kB/s`);
      if (latency) bits.push(`${latency} ms`);
    } else {
      bits.push('— fps');
    }
    bits.push(t('q_' + ZRDeskView.getPreset()));
    const screens = ZRDeskView.getScreens();
    if (screens.length > 1) bits.push(`${t('monitor')} ${ZRDeskView.getDisplay() + 1}/${screens.length}`);
    if (ZRDeskInput.raw()) bits.push('raw');
    el.stats.textContent = bits.join('   ·   ');
    if (s && s.stale && state === 'paired') el.dot.className = 'dot warn';
  }

  function paintTakeover() {
    const canDrive = state === 'paired';
    el.takeover.hidden = ZRDeskInput.driving() || !canDrive || !ZRDeskView.hasImage();
  }

  function paintMonitor() {
    const screens = ZRDeskView.getScreens();
    if (screens.length > 1) el.btnMonitor.textContent = `${t('monitor')} ${ZRDeskView.getDisplay() + 1}/${screens.length}`;
  }

  function showIdle(title, text, bad) {
    el.idle.hidden = false;
    el.idleTitle.textContent = title;
    el.idleText.textContent = text || '';
    el.idleText.className = bad ? 'bad' : '';
    el.spin.hidden = !!bad;
  }
  function hideIdle() { el.idle.hidden = true; paintTakeover(); }

  // ── actions (chords + toolbar) ────────────────────────────────────────────
  function action(a) {
    switch (a) {
      case 'grab':
        el.body.classList.add('driving');
        if (pendingClick) { ZRDeskInput.warpFromClient(pendingClick.x, pendingClick.y); pendingClick = null; }
        paintTakeover();
        toast(t('taken'));
        break;
      case 'ungrab':
        el.body.classList.remove('driving');
        paintTakeover();
        toast(t('released'));
        break;
      case 'fullscreen': toggleFullscreen(); break;
      case 'quality':
        ZRDeskView.cyclePreset();
        el.lblQuality.textContent = t('q_' + ZRDeskView.getPreset());
        paintStats(ZRDeskView.stats());
        break;
      case 'monitor':
        ZRDeskView.nextDisplay();
        ZRDeskInput.center();
        paintMonitor();
        paintStats(ZRDeskView.stats());
        break;
      case 'raw':
        ZRDeskInput.setRaw(!ZRDeskInput.raw());
        toast(ZRDeskInput.raw() ? t('raw_on') : t('raw_off'), ZRDeskInput.raw() ? 'warn' : '');
        paintStats(ZRDeskView.stats());
        break;
      case 'clip': pushClip(true); break;
      case 'help': el.help.hidden = false; buildHelp(); break;
      case 'quit': quit(); break;
      case 'capture': buildHelp(); break;
      case 'lockfail': toast('pointer lock refused', 'bad'); break;
    }
  }

  // clicking the picture is what takes control — remember where, so the remote
  // pointer lands under the local one once the lock is granted
  el.stage.addEventListener('mousedown', (e) => {
    if (ZRDeskInput.driving() || state !== 'paired') return;
    pendingClick = { x: e.clientX, y: e.clientY };
    ZRDeskInput.take();
  });
  el.takeBtn.addEventListener('click', (e) => { e.stopPropagation(); ZRDeskInput.take(); });

  el.btnQuality.addEventListener('click', () => action('quality'));
  el.btnMonitor.addEventListener('click', () => action('monitor'));
  el.btnClip.addEventListener('click', () => action('clip'));
  el.btnCad.addEventListener('click', () => ZRConn.send({ t: 'key', k: 'delete', mods: ['ctrl', 'alt'] }));
  el.btnFull.addEventListener('click', () => action('fullscreen'));
  el.btnHelp.addEventListener('click', () => action('help'));
  el.btnQuit.addEventListener('click', quit);
  el.helpClose.addEventListener('click', () => { el.help.hidden = true; });
  el.help.addEventListener('click', (e) => { if (e.target === el.help) el.help.hidden = true; });
  el.lblQuality.textContent = t('q_' + ZRDeskView.getPreset());

  // the bar hides while driving; nudging the cursor to the top brings it back
  window.addEventListener('mousemove', (e) => {
    if (!el.body.classList.contains('driving')) return;
    el.body.classList.toggle('peek', e.clientY <= 4);
  });

  function toggleFullscreen() {
    if (document.fullscreenElement) document.exitFullscreen();
    else document.documentElement.requestFullscreen().catch(() => {});
  }
  document.addEventListener('fullscreenchange', syncFullLabel);
  function syncFullLabel() { el.btnFull.textContent = document.fullscreenElement ? t('unfull') : t('full'); }

  function quit() {
    ZRDeskInput.release();
    ZRDeskView.stop();
    ZRConn.close();
    showIdle(t('closed'), '', true);
    state = 'closed';
    paintState();
  }

  // ── clipboard ─────────────────────────────────────────────────────────────
  // Reading the clipboard needs a permission in Chromium and is not offered at
  // all in some browsers, so: try to poll, and fall back to an explicit
  // "send mine now" gesture (Right Ctrl + C / the toolbar button).
  async function startClipboard() {
    ZRConn.send({ t: 'clipwatch', on: true });
    let granted = false;
    try {
      const st = await navigator.permissions.query({ name: 'clipboard-read' });
      granted = st.state === 'granted';
    } catch { granted = false; }
    if (!granted) { clipManual = true; toast(t('clip_manual')); return; }
    clipPoll = setInterval(() => pushClip(false), 1500);
  }

  async function pushClip(explicit) {
    if (!caps.clip) return;
    let s = '';
    try { s = await navigator.clipboard.readText(); }
    catch {
      if (explicit) toast(t('clip_denied'), 'warn');
      clearInterval(clipPoll);
      clipManual = true;
      return;
    }
    if (!s || s === clipLast || s.length > 16 * 1024) return;
    clipLast = s;
    ZRConn.send({ t: 'clip', s });
    if (explicit) toast(t('clip_out'));
  }

  async function onRemoteClip(s) {
    if (!s || s === clipLast) return;
    clipLast = s;
    try { await navigator.clipboard.writeText(s); toast(t('clip_in')); }
    catch { /* no permission: the host still has it, nothing to do here */ }
  }

  // ── help sheet ────────────────────────────────────────────────────────────
  function buildHelp() {
    const rows = [
      ['Right Ctrl', t('help_grab')],
      ['Right Ctrl + F', t('help_full')],
      ['Right Ctrl + P', t('help_quality')],
      ['Right Ctrl + M', t('help_monitor')],
      ['Right Ctrl + K', t('help_raw')],
      ['Right Ctrl + C', t('help_clip')],
      ['Right Ctrl + Del', t('help_cad')],
      ['Right Ctrl + Tab', t('help_alttab')],
      ['Right Ctrl + S', t('help_super')],
      ['Right Ctrl + Esc', t('help_esc')],
      ['Right Ctrl + Q', t('help_quit')],
    ];
    el.helpList.replaceChildren(...rows.map(([k, v]) => {
      const li = document.createElement('li');
      const a = document.createElement('span'); a.textContent = k;
      const b = document.createElement('span'); b.textContent = v;
      li.append(a, b);
      return li;
    }));
    el.helpCapture.textContent = ZRDeskInput.keyboardLocked() ? t('help_capture_full') : t('help_capture_part');
  }

  // ── toasts ────────────────────────────────────────────────────────────────
  function toast(msg, kind) {
    const n = document.createElement('div');
    n.className = 'toast' + (kind ? ' ' + kind : '');
    n.textContent = msg;
    el.toasts.appendChild(n);
    setTimeout(() => { n.classList.add('out'); setTimeout(() => n.remove(), 300); }, 2600);
    while (el.toasts.children.length > 4) el.toasts.firstChild.remove();
  }
})();
