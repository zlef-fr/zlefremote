// Lock-screen presence via the Media Session API.
//
// A web app can't draw over the Android lock screen (that needs a native
// showWhenLocked activity). The one lock-screen surface the browser DOES give
// us is the media notification: hold an (essentially silent) audio session and
// the OS renders a "now playing" card on the lock screen + shade. We point its
// play / prev / next (and seek = volume) buttons at ZlefRemote's media keys, so
// you can pause or skip your computer's media from the lock screen without
// unlocking — the closest a PWA gets to "the app is there on the lock screen".
//
// Opt-in: holding an audio session steals audio focus (it pauses music playing
// on the phone itself), so it's a setting, default OFF. Toggle in Settings.
const ZRMedia = (() => {
  const SUPPORTED = 'mediaSession' in navigator;
  let send = () => {};
  let tr = (k) => k;
  let audio = null;
  let blobUrl = null;
  let active = false;        // a session is currently held
  let host = '';
  let primed = false;        // user gesture has unlocked audio playback

  // setting: lock-screen controls (default OFF — opt-in)
  const enabled = () => localStorage.getItem('zr_lockctl') === '1';

  function config({ send: s, t }) { if (s) send = s; if (t) tr = t; }

  // A few hundred ms of 16-bit PCM silence, looped. Built at runtime so the
  // header/sample values are guaranteed correct (silence for 16-bit = 0, and an
  // ArrayBuffer is zero-initialised, so we only write the WAV header).
  function silentWavUrl(seconds = 0.4) {
    const sr = 8000, n = Math.floor(sr * seconds), dataLen = n * 2;
    const buf = new ArrayBuffer(44 + dataLen);
    const v = new DataView(buf);
    let p = 0;
    const str = (s) => { for (let i = 0; i < s.length; i++) v.setUint8(p++, s.charCodeAt(i)); };
    const u32 = (x) => { v.setUint32(p, x, true); p += 4; };
    const u16 = (x) => { v.setUint16(p, x, true); p += 2; };
    str('RIFF'); u32(36 + dataLen); str('WAVE');
    str('fmt '); u32(16); u16(1); u16(1); u32(sr); u32(sr * 2); u16(2); u16(16);
    str('data'); u32(dataLen);
    return URL.createObjectURL(new Blob([buf], { type: 'audio/wav' }));
  }

  function ensureAudio() {
    if (audio) return audio;
    blobUrl = silentWavUrl();
    audio = new Audio(blobUrl);
    audio.loop = true;
    audio.volume = 0.0001;          // inaudible but a real, non-zero stream
    audio.preload = 'auto';
    audio.setAttribute('playsinline', '');
    // If the OS/another app pauses our stream while we want it held, resume it
    // so the lock-screen card never silently disappears.
    audio.addEventListener('pause', () => { if (active) tryPlay(); });
    return audio;
  }

  function tryPlay() {
    const a = ensureAudio();
    const pr = a.play();
    if (pr && pr.then) pr.then(() => { primed = true; }).catch(() => { /* autoplay blocked — will retry on first user gesture */ });
  }

  // Browsers gate audio start on a user gesture. Catch the first interaction in
  // control mode and use it to unlock playback, so the card appears as soon as
  // the user touches the trackpad.
  function armGesturePrime() {
    if (primed) return;
    const go = () => { if (active && !primed) tryPlay(); cleanup(); };
    const cleanup = () => {
      document.removeEventListener('pointerdown', go, true);
      document.removeEventListener('touchend', go, true);
      document.removeEventListener('keydown', go, true);
    };
    document.addEventListener('pointerdown', go, true);
    document.addEventListener('touchend', go, true);
    document.addEventListener('keydown', go, true);
  }

  function setMeta() {
    if (!SUPPORTED) return;
    const origin = location.origin;
    navigator.mediaSession.metadata = new MediaMetadata({
      title: 'ZlefRemote',
      artist: host ? (tr('to') + ' ' + host) : tr('paired'),
      album: 'ZlefRemote',
      artwork: [
        { src: origin + '/app/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
        { src: origin + '/app/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
      ],
    });
    navigator.mediaSession.playbackState = 'playing';
  }

  // Map the lock-screen transport buttons to PC media keys. We always snap our
  // own playbackState back to 'playing' so the card (and its buttons) persist —
  // play/pause here is a momentary "toggle the computer's playback", not a
  // toggle of our silent holder stream.
  function setHandlers() {
    if (!SUPPORTED) return;
    const fire = (k) => { try { send({ t: 'media', k }); } catch { /* connection may have dropped between lock-screen tap and send */ } };
    const keep = () => { navigator.mediaSession.playbackState = 'playing'; };
    const set = (action, fn) => {
      try { navigator.mediaSession.setActionHandler(action, fn); } catch { /* action not supported by this browser/OS */ }
    };
    set('play', () => { fire('playpause'); tryPlay(); keep(); });
    set('pause', () => { fire('playpause'); tryPlay(); keep(); });
    set('previoustrack', () => { fire('prev'); keep(); });
    set('nexttrack', () => { fire('next'); keep(); });
    // Bonus: where the OS surfaces seek buttons, use them for volume.
    set('seekbackward', () => { fire('voldown'); keep(); });
    set('seekforward', () => { fire('volup'); keep(); });
  }

  function clearHandlers() {
    if (!SUPPORTED) return;
    for (const a of ['play', 'pause', 'previoustrack', 'nexttrack', 'seekbackward', 'seekforward']) {
      try { navigator.mediaSession.setActionHandler(a, null); } catch { /* action may not be supported; ignore */ }
    }
  }

  // Begin holding a lock-screen session for the given computer.
  function start(hostName) {
    host = hostName || host;
    if (!SUPPORTED || !enabled()) return;
    active = true;
    setMeta();
    setHandlers();
    tryPlay();
    armGesturePrime();
  }

  function stop() {
    active = false;
    clearHandlers();
    if (SUPPORTED) { try { navigator.mediaSession.playbackState = 'none'; navigator.mediaSession.metadata = null; } catch { /* ignore */ } }
    if (audio) { try { audio.pause(); } catch { /* ignore */ } }
  }

  // Settings toggle flips the feature live during a session.
  function setEnabled(on) {
    localStorage.setItem('zr_lockctl', on ? '1' : '0');
    if (on) start(host); else stop();
  }

  return { config, start, stop, setEnabled, enabled, supported: () => SUPPORTED };
})();
