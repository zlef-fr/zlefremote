// Connection layer. Two transports, one API:
//   • relay  — /r/<ROOM>#k=…  → wss://<host>/ws, JOIN a room on the relay
//   • direct — /#k=…          → ws(s)://<host>/ws, the agent's own LAN server
// Both carry {t:'data', payload:<sealed>} frames. send()/onCmd() speak plaintext
// objects; sealing/opening happens here.
const ZRConn = (() => {
  let ws = null, mode = 'direct', room = null, cbs = {}, state = 'init';
  let reconnectTimer = null, manualClose = false;
  let keyB64 = null, persistent = false;

  function on(ev, fn) { cbs[ev] = fn; }
  function emit(ev, d) { if (cbs[ev]) cbs[ev](d); }
  function setState(s) { state = s; emit('state', s); }

  function parseKey() {
    const m = location.hash.match(/k=([A-Za-z0-9\-_]+)/);
    return m ? m[1] : null;
  }
  // a persistent agent appends &p=1 — the device can be saved & reconnected.
  function parsePersistent() { return /[#&]p=1\b/.test(location.hash); }
  // is there a target to connect to at all? (relay room+key, or direct key)
  function hasTarget() { return !!parseKey(); }

  function detect() {
    const rm = location.pathname.match(/^\/r\/([A-Za-z0-9]{4,8})/i);
    if (rm) { mode = 'relay'; room = rm[1].toUpperCase(); }
    else { mode = 'direct'; }
  }

  async function start() {
    detect();
    const k = parseKey();
    if (!k) { setState('nokey'); return; }
    keyB64 = k;
    persistent = parsePersistent();
    try { await ZRCrypto.init(k); } catch { setState('nokey'); return; }
    connect();
  }

  function wsUrl() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    return `${proto}://${location.host}/ws`;
  }

  function connect() {
    setState('connecting');
    manualClose = false;
    ws = new WebSocket(wsUrl());

    ws.onopen = () => {
      if (mode === 'relay') ws.send(JSON.stringify({ t: 'join', room }));
      else afterLink(); // direct: agent is already listening
    };

    ws.onmessage = async (e) => {
      let msg; try { msg = JSON.parse(e.data); } catch { return; }
      switch (msg.t) {
        case 'joined': afterLink(); break;
        case 'data':
          try { emit('cmd', await ZRCrypto.open(msg.payload)); } catch { /* decryption failure — bad key or corrupted frame, drop silently */ }
          break;
        case 'closed': setState('closed'); emit('closed', msg.reason); break;
        case 'error': setState('error'); emit('error', msg.error); break;
        case 'pong': break;
      }
    };

    ws.onclose = () => {
      if (manualClose) return;
      if (state === 'paired' || state === 'linked') {
        setState('reconnecting');
        clearTimeout(reconnectTimer);
        reconnectTimer = setTimeout(connect, 1200);
      } else if (state === 'connecting') {
        setState('error'); emit('error', 'connect_failed');
      }
    };
    ws.onerror = () => { /* errors surface as close events; handled in ws.onclose */ };
  }

  async function afterLink() {
    setState('linked');
    // handshake: prove we hold the key. The host replies with a welcome.
    await send({ t: 'hello', v: 1, ua: navigator.userAgent.slice(0, 80) });
  }

  async function send(cmd) {
    if (!ws || ws.readyState !== 1) return;
    const payload = await ZRCrypto.seal(cmd);
    ws.send(JSON.stringify({ t: 'data', payload }));
  }

  function markPaired() { setState('paired'); }

  function close() { manualClose = true; clearTimeout(reconnectTimer); try { ws && ws.close(); } catch { /* ignore — socket may already be gone */ } }

  setInterval(() => { if (ws && ws.readyState === 1) try { ws.send(JSON.stringify({ t: 'ping' })); } catch { /* ignore — socket closed between check and send */ } }, 25000);

  return {
    start, send, on, close, markPaired, hasTarget,
    getState: () => state, getMode: () => mode,
    getRoom: () => room, getKeyB64: () => keyB64, isPersistent: () => persistent,
  };
})();
