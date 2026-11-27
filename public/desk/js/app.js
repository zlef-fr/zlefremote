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
    blackout: $('blackout'),
  };

  let caps = {}, hostName = '', state = 'connecting';
  let latency = 0, pingSeq = 0, pingAt = 0;
  let clipLast = '', clipPoll = 0, clipManual = false;
  let clipImgSeq = 0, clipImgLast = '', pasteWaiter = null;
  const clipAcks = new Map();   // image id → resolve(ack), see sendImage
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
    onPaste: onPasteChord,
  });

  if (!ZRConn.hasTarget()) {
    showIdle(t('nokey'), '', true);
  } else {
    ZRConn.start();
  }

  ZRConn.on('state', (s) => {
    state = s;
    if (s === 'reconnecting' || s === 'closed') {
      // a picture half-sent on a dropped link is lost: say so now, not in 15 s
      clipAcks.forEach((resolve) => resolve({ ok: false, err: 'timeout' }));
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
      case 'climgack': onImageAck(c); break;
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

    if (caps.screen) { if (!ZRDeskInput.blackedOut()) ZRDeskView.start(); showIdle(t('waiting'), '', false); }
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
        el.body.classList.remove('driving', 'peek');
        paintTakeover();
        if (!ZRDeskInput.blackedOut()) toast(t('released'));
        break;
      // the picture keeps nothing on screen and the stream pauses: the other
      // computer stops capturing until you come back
      case 'blackout':
        el.help.hidden = true;
        el.toasts.replaceChildren();
        el.blackout.hidden = false;
        requestAnimationFrame(() => el.body.classList.add('blacked-out'));
        ZRDeskView.stop();
        break;
      case 'unblackout':
        el.body.classList.remove('blacked-out');
        setTimeout(() => { if (!ZRDeskInput.blackedOut()) el.blackout.hidden = true; }, 260);
        if (caps.screen && state === 'paired') ZRDeskView.start();
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
      case 'clip': pushClip(); break;
      case 'help': el.help.hidden = false; buildHelp(); break;
      case 'quit': quit(); break;
      case 'capture': buildHelp(); break;
      case 'lockfail': toast('pointer lock refused', 'bad'); break;
    }
  }

  // clicking the picture is what takes control — remember where, so the remote
  // pointer lands under the local one once the lock is granted
  el.stage.addEventListener('mousedown', (e) => {
    if (ZRDeskInput.driving() || ZRDeskInput.blackedOut() || state !== 'paired') return;
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
  // Text and pictures, in four ways in:
  //   · Ctrl+V while in control: the browser's own paste event hands us the
  //     clipboard (text or image, no permission needed); we ship it, THEN let
  //     the Ctrl+V through, so the other computer pastes what you just copied.
  //   · a paste on this page while not in control (Ctrl+V on the picture).
  //   · Right Ctrl + C / the toolbar button: an explicit read, which may ask
  //     for the clipboard permission.
  //   · a 1.5 s poll of the TEXT clipboard once that permission is granted.
  //     Pictures are not polled: reading one re-encodes the whole bitmap, and
  //     the paths above already cover every way a picture gets pasted.
  // Nothing is dropped quietly: every refusal ends in a toast saying why.
  const CLIP_TEXT_MAX = 16 * 1024;         // agent's maxClipBytes (UTF-8 bytes)
  const CLIP_IMG_MAX = 12 * 1024 * 1024;   // agent's maxClipImageBytes
  const CLIP_IMG_CHUNK = 36000;            // base64 chars per frame, seals to ~49 KB of the relay's 64

  async function startClipboard() {
    ZRConn.send({ t: 'clipwatch', on: true });
    let granted = false;
    try {
      const st = await navigator.permissions.query({ name: 'clipboard-read' });
      granted = st.state === 'granted';
    } catch { granted = false; }
    if (!granted) { clipManual = true; toast(t('clip_manual')); return; }
    clipPoll = setInterval(pollClip, 1500);
  }

  async function pollClip() {
    let s = '';
    try { s = await navigator.clipboard.readText(); }
    catch { clearInterval(clipPoll); clipManual = true; return; }
    if (s && s !== clipLast && utf8Len(s) <= CLIP_TEXT_MAX) sendText(s, false);
  }

  // Right Ctrl + C / toolbar button: read everything the clipboard holds
  async function pushClip() {
    if (!caps.clip) return;
    let data;
    try { data = await readClipboard(); }
    catch { toast(t('clip_denied'), 'warn'); return; }
    if (!data.image && !data.text) { toast(t('clip_empty'), 'warn'); return; }
    await shipClip(data, true);
  }

  async function readClipboard() {
    if (navigator.clipboard.read) {
      const out = {};
      for (const item of await navigator.clipboard.read()) {
        const img = item.types.find((ty) => ty.startsWith('image/'));
        if (img && !out.image) out.image = await item.getType(img);
        if (item.types.includes('text/plain') && !out.text) out.text = await (await item.getType('text/plain')).text();
      }
      return out;
    }
    return { text: await navigator.clipboard.readText() };
  }

  // what a paste event carries: a picture (copied pixels, or an image FILE
  // copied in the file manager) wins over text, like it does in a chat app
  function fromPasteEvent(dt) {
    const out = {};
    if (!dt) return out;
    for (const it of dt.items || []) {
      if (it.kind === 'file' && it.type.startsWith('image/') && !out.image) out.image = it.getAsFile();
    }
    const text = dt.getData('text/plain');
    if (text) out.text = text;
    if (!out.image && !text && dt.files && dt.files.length) out.other = true;
    return out;
  }

  // explicit = the user asked for this send right now: say so when it lands
  async function shipClip(data, explicit) {
    if (data.image) return sendImage(data.image);
    if (data.text) return sendText(data.text, explicit, true);
    if (data.other) toast(t('clip_notimage'), 'warn');
  }

  function sendText(s, explicit, force) {
    if (!caps.clip) return;
    if (utf8Len(s) > CLIP_TEXT_MAX) { toast(t('clip_toolong'), 'warn'); return; }
    if (s === clipLast && !force) return;
    clipLast = s;
    clipImgLast = '';   // the host clipboard now holds text, not our last picture
    ZRConn.send({ t: 'clip', s });
    if (explicit) toast(t('clip_out'));
  }

  async function sendImage(blob) {
    if (!caps.clip || !('clipimg' in caps)) { toast(t('clip_img_old'), 'warn'); return; }
    if (!caps.clipimg) { toast(t('clip_img_host'), 'warn'); return; }
    let png;
    try { png = await asPNG(blob); }
    catch { toast(t('clip_img_bad'), 'bad'); return; }
    if (png.size > CLIP_IMG_MAX) { toast(t('clip_img_big', { mb: Math.round(CLIP_IMG_MAX / 1048576) }), 'warn'); return; }

    const b64 = await toBase64(png);
    const sig = png.size + ':' + b64.length + ':' + b64.slice(-64);
    if (sig === clipImgLast) return;   // already over there, nothing to wait for

    const id = ++clipImgSeq;
    const n = Math.ceil(b64.length / CLIP_IMG_CHUNK);
    const bar = n > 3 ? progressToast(t('clip_img_sending')) : null;
    const acked = new Promise((resolve) => {
      clipAcks.set(id, resolve);
      // a full 12 MB picture over a slow relay still fits in this budget
      setTimeout(() => resolve({ ok: false, err: 'timeout' }), 15000 + n * 250);
    });
    for (let i = 0; i < n; i++) {
      if (state !== 'paired') break;
      // don't bury the mouse and keyboard under a picture: keep the socket's
      // backlog short, so input frames slip in between chunks
      while (ZRConn.buffered() > 256 * 1024 && state === 'paired') await new Promise((r) => setTimeout(r, 20));
      await ZRConn.send({ t: 'climg', ci: id, cp: i, cn: n, cd: b64.slice(i * CLIP_IMG_CHUNK, (i + 1) * CLIP_IMG_CHUNK) });
      if (bar) bar.set((i + 1) / n);
    }
    const ack = state === 'paired' ? await acked : { ok: false, err: 'timeout' };
    clipAcks.delete(id);
    if (ack.ok) {
      clipImgLast = sig;
      clipLast = '';    // the host clipboard holds the picture now, not our last text
      const kb = Math.max(1, Math.round(png.size / 1024));
      const fr = ZRDeskI18n.lang === 'fr';
      const size = kb >= 1024
        ? (kb / 1024).toLocaleString(ZRDeskI18n.lang, { maximumFractionDigits: 1 }) + (fr ? ' Mo' : ' MB')
        : kb + (fr ? ' Ko' : ' KB');
      const msg = t('clip_img_out', { size });
      bar ? bar.done(msg) : toast(msg);
    } else {
      // unsupported can only mean the welcome lied (an agent swapped mid-session);
      // order/timeout both mean the chunks did not all arrive
      const err = { unsupported: 'clip_img_host', toobig: 'clip_img_err_toobig', format: 'clip_img_err_format',
        order: 'clip_img_err_lost', timeout: 'clip_img_err_lost' }[ack.err] || 'clip_img_err_write';
      const msg = t(err);
      bar ? bar.done(msg, 'bad') : toast(msg, 'bad');
    }
  }

  function onImageAck(c) {
    const resolve = clipAcks.get(c.ci);
    if (resolve) resolve({ ok: !!c.ok, err: c.err });
  }

  // the agent stores PNG (the one format every OS clipboard takes); a copied
  // JPEG/WebP/GIF file is redrawn into one here
  async function asPNG(blob) {
    if (blob.type === 'image/png') return blob;
    const bmp = await createImageBitmap(blob);
    const c = document.createElement('canvas');
    c.width = bmp.width; c.height = bmp.height;
    c.getContext('2d').drawImage(bmp, 0, 0);
    bmp.close && bmp.close();
    const out = await new Promise((r) => c.toBlob(r, 'image/png'));
    if (!out) throw new Error('encode');
    return out;
  }

  function toBase64(blob) {
    return new Promise((resolve, reject) => {
      const r = new FileReader();
      r.onload = () => resolve(String(r.result).slice(String(r.result).indexOf(',') + 1));
      r.onerror = () => reject(r.error);
      r.readAsDataURL(blob);
    });
  }

  function utf8Len(s) { return new TextEncoder().encode(s).length; }

  // Ctrl+V while in control (called by the input pump, which holds the key back
  // until this settles). The browser fires 'paste' right after the keydown we
  // let through; if it doesn't (nothing pasteable, or a browser that keeps the
  // shortcut), the key goes through on its own after a short wait.
  function onPasteChord() {
    if (!caps.clip) return Promise.resolve();
    return new Promise((resolve) => {
      const done = () => { if (pasteWaiter === entry) pasteWaiter = null; resolve(); };
      const entry = { done, timer: setTimeout(done, 250) };
      pasteWaiter = entry;
    });
  }

  document.addEventListener('paste', (e) => {
    if (state !== 'paired' || !caps.clip) return;
    e.preventDefault();
    const data = fromPasteEvent(e.clipboardData);
    const w = pasteWaiter;
    if (w) { clearTimeout(w.timer); pasteWaiter = null; }
    // while in control the paste is on its way to the other computer anyway:
    // only pictures (and failures) are worth a word; a bare paste on the page
    // is an explicit send and says so
    Promise.resolve(shipClip(data, !w)).catch(() => {}).finally(() => { if (w) w.done(); });
  });

  async function onRemoteClip(s) {
    if (!s || s === clipLast) return;
    clipLast = s;
    clipImgLast = '';
    try { await navigator.clipboard.writeText(s); toast(t('clip_in')); }
    catch { /* no permission: the host still has it, nothing to do here */ }
  }

  // ── help sheet ────────────────────────────────────────────────────────────
  function buildHelp() {
    const rows = [
      ['Right Ctrl', t('help_grab')],
      ['Ctrl + Esc', t('help_escape')],
      ['Right Ctrl × 2', t('help_blackout')],
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
    el.helpCapture.textContent = ZRDeskInput.keyboardLocked() ? t('help_capture_full')
      : ZRDeskInput.canLockKeys() ? t('help_capture_auto') : t('help_capture_part');
  }

  // ── toasts ────────────────────────────────────────────────────────────────
  // a toast with a progress bar, for sends that take a moment
  function progressToast(msg) {
    const n = document.createElement('div');
    n.className = 'toast progress';
    const label = document.createElement('span');
    label.textContent = msg;
    const track = document.createElement('i');
    const fill = document.createElement('b');
    track.appendChild(fill);
    n.append(label, track);
    el.toasts.appendChild(n);
    return {
      set(f) { fill.style.transform = `scaleX(${Math.min(1, f)})`; },
      done(text, kind) {
        label.textContent = text;
        n.classList.add('settled');
        if (kind) n.classList.add(kind);
        setTimeout(() => { n.classList.add('out'); setTimeout(() => n.remove(), 300); }, 2600);
      },
    };
  }

  function toast(msg, kind) {
    const n = document.createElement('div');
    n.className = 'toast' + (kind ? ' ' + kind : '');
    n.textContent = msg;
    el.toasts.appendChild(n);
    setTimeout(() => { n.classList.add('out'); setTimeout(() => n.remove(), 300); }, 2600);
    while (el.toasts.children.length > 4) el.toasts.firstChild.remove();
  }
})();
