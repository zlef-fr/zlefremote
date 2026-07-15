'use strict';
const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { WebSocketServer } = require('ws');
const { Rooms } = require('./lib/rooms');
const { landing, startPage, privacyPage } = require('./lib/pages');
const { pickLang } = require('./lib/i18n');

const PORT = parseInt(process.env.PORT || '10067', 10);
const PUBLIC_HOST = process.env.PUBLIC_HOST || 'remote.zlef.fr';
const ROOT = __dirname;
const MAX_FRAME = 64 * 1024; // 64 KB ceiling per relayed frame

const rooms = new Rooms();
let agentPings = 0;

const MIME = {
  '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8', '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon',
  '.woff2': 'font/woff2', '.webmanifest': 'application/manifest+json',
};

function distHave() {
  try { return new Set(fs.readdirSync(path.join(ROOT, 'dist'))); }
  catch { return new Set(); }
}

// ── agent release manifest (consumed by `zlefremote-agent -update`) ──────────
const AGENT_VERSION = '1.5.1';
const AGENT_ASSETS = {
  'linux-amd64': 'zlefremote-agent-linux-amd64',
  'windows-amd64': 'zlefremote-agent-windows-amd64.exe',
  'darwin-arm64': 'zlefremote-agent-darwin-arm64',
};
const _shaCache = new Map(); // file -> { mtimeMs, size, sha }
function fileSha(file) {
  try {
    const full = path.join(ROOT, 'dist', file);
    const st = fs.statSync(full);
    const c = _shaCache.get(file);
    if (c && c.mtimeMs === st.mtimeMs && c.size === st.size) return c.sha;
    const sha = crypto.createHash('sha256').update(fs.readFileSync(full)).digest('hex');
    _shaCache.set(file, { mtimeMs: st.mtimeMs, size: st.size, sha });
    return sha;
  } catch { return null; }
}

function send(res, code, body, type, extra) {
  res.writeHead(code, Object.assign({ 'Content-Type': type || 'text/plain; charset=utf-8' }, extra || {}));
  res.end(body);
}

function safeStatic(res, baseDir, rel) {
  const full = path.normalize(path.join(baseDir, rel));
  if (full !== baseDir && !full.startsWith(baseDir + path.sep)) return send(res, 403, 'forbidden');
  fs.readFile(full, (err, buf) => {
    if (err) return send(res, 404, 'not found');
    const ext = path.extname(full).toLowerCase();
    send(res, 200, buf, MIME[ext] || 'application/octet-stream',
      { 'Cache-Control': ext === '.html' ? 'no-cache' : 'public, max-age=3600' });
  });
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const p = url.pathname;

  if (p === '/healthz') return send(res, 200, `ok (${agentPings} agent pings)`);

  // anonymous agent usage ping (version/os/arch/mode — no personal data)
  if (p === '/api/agent/ping' && req.method === 'POST') {
    let body = '';
    req.on('data', (c) => { body += c; if (body.length > 2048) req.destroy(); });
    req.on('end', () => {
      try {
        const d = JSON.parse(body || '{}');
        agentPings++;
        console.log(`agent ping #${agentPings} v=${String(d.version).slice(0, 16)} os=${String(d.os).slice(0, 12)}/${String(d.arch).slice(0, 12)} mode=${String(d.mode).slice(0, 8)}`);
      } catch {}
      res.writeHead(204); res.end();
    });
    return;
  }

  // landing (SSR, i18n)
  if (p === '/' || p === '/index.html') {
    const lang = pickLang(req);
    return send(res, 200, landing(lang, distHave()), 'text/html; charset=utf-8', { 'Cache-Control': 'no-cache' });
  }

  // /start — mobile funnel: preview the remote + send the agent to your desktop
  if (p === '/start') {
    const lang = pickLang(req);
    return send(res, 200, startPage(lang), 'text/html; charset=utf-8', { 'Cache-Control': 'no-cache' });
  }

  // /privacy — privacy policy (i18n, indexable)
  if (p === '/privacy') {
    const lang = pickLang(req);
    return send(res, 200, privacyPage(lang), 'text/html; charset=utf-8', { 'Cache-Control': 'no-cache' });
  }

  // relay client app — /r (home/device picker), /r/ and /r/<room> serve the
  // phone remote SPA (the PWA start_url is /r/, and saved-device nav lands here)
  if (p === '/r' || p === '/r/' || /^\/r\/[A-Z0-9]{4,8}$/i.test(p)) {
    return safeStatic(res, path.join(ROOT, 'public', 'app'), 'index.html');
  }

  // The apt repo is now zlef-wide at apt.zlef.fr. Permanently redirect the old
  // remote.zlef.fr/apt/* paths there so existing sources.list entries keep
  // working (apt follows redirects; the signing key is the same).
  if (p === '/apt' || p.startsWith('/apt/')) {
    res.writeHead(301, { Location: 'https://apt.zlef.fr' + p.slice('/apt'.length) });
    return res.end();
  }

  // agent release manifest — version + per-asset sha256 + download URLs
  if (p === '/api/agent/version') {
    const assets = {};
    for (const [k, f] of Object.entries(AGENT_ASSETS)) {
      const sha = fileSha(f);
      if (sha) assets[k] = { file: f, sha256: sha, url: `https://${PUBLIC_HOST}/download/${f}` };
    }
    return send(res, 200, JSON.stringify({ version: AGENT_VERSION, assets }),
      'application/json; charset=utf-8', { 'Cache-Control': 'no-cache' });
  }

  // agent binary downloads
  if (p.startsWith('/download/')) {
    const file = decodeURIComponent(p.slice('/download/'.length));
    const ok = /^zlefremote-agent-[a-z0-9.\-]+$/i.test(file)
      || /^zlefremote-xfce-plugin(?:-[a-z0-9.\-]+)?\.tar\.gz$/i.test(file);
    if (!ok) return send(res, 400, 'bad name');
    return safeStatic(res, path.join(ROOT, 'dist'), file);
  }

  // PWA service worker — served from root so its scope is the whole origin
  // (it only intercepts the remote app's own routes; everything else passes
  // through). no-cache so a new worker is picked up immediately.
  if (p === '/sw.js') {
    return fs.readFile(path.join(ROOT, 'public', 'app', 'sw.js'), (err, buf) => {
      if (err) return send(res, 404, 'not found');
      send(res, 200, buf, 'text/javascript; charset=utf-8',
        { 'Cache-Control': 'no-cache', 'Service-Worker-Allowed': '/' });
    });
  }

  // PWA manifest for the installable remote app
  if (p === '/app.webmanifest') {
    return fs.readFile(path.join(ROOT, 'public', 'app', 'app.webmanifest'), (err, buf) => {
      if (err) return send(res, 404, 'not found');
      send(res, 200, buf, 'application/manifest+json; charset=utf-8', { 'Cache-Control': 'no-cache' });
    });
  }

  // self-hosted social card
  if (p === '/og.png') return safeStatic(res, path.join(ROOT, 'public'), 'og.png');

  // static: /app/* , /css/* , /js/*
  if (p.startsWith('/app/')) return safeStatic(res, path.join(ROOT, 'public', 'app'), p.slice('/app/'.length));
  if (p.startsWith('/css/') || p.startsWith('/js/') || p.startsWith('/i18n/'))
    return safeStatic(res, path.join(ROOT, 'public'), p.replace(/^\//, ''));

  send(res, 404, 'not found');
});

// ── WebSocket relay ─────────────────────────────────────────────────────────
const wss = new WebSocketServer({ server, path: '/ws', maxPayload: MAX_FRAME });

function clientIp(req) {
  return (req.headers['cf-connecting-ip']
    || (req.headers['x-forwarded-for'] || '').split(',')[0].trim()
    || req.socket.remoteAddress || 'unknown');
}

wss.on('connection', (ws, req) => {
  // browser clients must come from our own origin; agents send no Origin header
  const origin = req.headers.origin;
  if (origin && origin !== `https://${PUBLIC_HOST}` && !/^https?:\/\/(localhost|127\.|192\.168\.|10\.|172\.)/.test(origin)) {
    try { ws.close(1008, 'origin'); } catch {} return;
  }
  ws._ip = clientIp(req);
  ws._t0 = Date.now();

  ws.on('message', (raw) => {
    let msg;
    try { msg = JSON.parse(raw.toString()); } catch { return; }
    switch (msg.t) {
      case 'host': {
        if (ws._room) return;
        // optional desired code: persistent agents derive a stable room from
        // their key so saved phones reconnect to the same address.
        const r = rooms.createRoom(ws, ws._ip, msg.room);
        if (r.error) return ws.send(JSON.stringify({ t: 'error', error: r.error }));
        ws.send(JSON.stringify({ t: 'hosted', room: r.code }));
        break;
      }
      case 'join': {
        if (ws._room) return;
        const r = rooms.join((msg.room || '').toUpperCase(), ws);
        if (r.error) return ws.send(JSON.stringify({ t: 'error', error: r.error }));
        ws.send(JSON.stringify({ t: 'joined', room: (msg.room || '').toUpperCase(), id: r.id }));
        break;
      }
      case 'data': {
        // blind relay of opaque ciphertext
        if (ws._role === 'client') rooms.fromClient(ws, msg.payload);
        else if (ws._role === 'host') rooms.fromHost(ws, msg.payload, msg.to);
        break;
      }
      case 'ping': ws.send(JSON.stringify({ t: 'pong' })); break;
    }
  });

  ws.on('close', () => rooms.leave(ws));
  ws.on('error', () => rooms.leave(ws));
});

// keepalive: drop dead sockets
setInterval(() => {
  for (const ws of wss.clients) {
    if (ws._alive === false) { try { ws.terminate(); } catch {} continue; }
    ws._alive = false;
    try { ws.ping(); } catch {}
  }
}, 30000).unref?.();
wss.on('connection', (ws) => { ws._alive = true; ws.on('pong', () => { ws._alive = true; }); });

server.listen(PORT, () => console.log(`ZlefRemote relay on :${PORT} (${PUBLIC_HOST}) — ${rooms.hostCount()} rooms`));
