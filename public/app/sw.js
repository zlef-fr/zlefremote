/* ZlefRemote PWA service worker.
 * Scope is the whole origin (served from /sw.js), but it ONLY ever touches the
 * remote app's own shell + static assets. The relay endpoints (/ws, /api/*) and
 * the marketing site are always passed straight through to the network.
 *
 * Bump CACHE whenever the shell HTML, any /app/* asset, or a ?vN query changes,
 * or returning installs keep serving the stale shell. */
const CACHE = 'zr-pwa-v10';

// Same-origin shell + assets (no query strings; matched with ignoreSearch).
const SHELL = [
  '/r',
  '/app/index.html',
  '/app/css/client.css',
  '/app/js/icons.js',
  '/app/js/i18n.js',
  '/app/js/crypto.js',
  '/app/js/conn.js',
  '/app/js/input.js',
  '/app/js/home.js',
  '/app/js/media-session.js',
  '/app/js/screen.js',
  '/app/js/app.js',
  '/app.webmanifest',
  '/app/icons/icon-192.png',
  '/app/icons/icon-512.png',
];
// Cross-origin design tokens — best-effort (don't fail install if offline).
const EXTRA = ['https://da.zlef.fr/tokens.css'];

self.addEventListener('install', (e) => {
  e.waitUntil((async () => {
    const c = await caches.open(CACHE);
    await c.addAll(SHELL);
    await Promise.allSettled(EXTRA.map((u) => c.add(u).catch(() => {})));
    self.skipWaiting();
  })());
});

self.addEventListener('activate', (e) => {
  e.waitUntil((async () => {
    const keys = await caches.keys();
    await Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)));
    await self.clients.claim();
  })());
});

function isAppAsset(url) {
  return url.pathname === '/app.webmanifest'
    || url.pathname.startsWith('/app/')
    || (url.origin === 'https://da.zlef.fr' && url.pathname === '/tokens.css');
}

self.addEventListener('fetch', (e) => {
  const req = e.request;
  if (req.method !== 'GET') return;                 // never touch POST etc.
  const url = new URL(req.url);

  // Live relay traffic + telemetry: always network, never cached.
  if (url.origin === location.origin && (url.pathname.startsWith('/api/') || url.pathname === '/ws')) return;

  // App navigations (/r, /r/<ROOM>) → network-first, fall back to cached shell
  // so the remote opens instantly and works offline until the WS is needed.
  if (req.mode === 'navigate') {
    if (url.origin === location.origin && (url.pathname === '/r' || url.pathname.startsWith('/r/'))) {
      e.respondWith((async () => {
        try {
          const net = await fetch(req);
          const c = await caches.open(CACHE);
          c.put('/r', net.clone());
          return net;
        } catch {
          return (await caches.match('/r', { ignoreSearch: true })) || Response.error();
        }
      })());
    }
    return; // marketing pages & everything else: untouched
  }

  // Static app assets → stale-while-revalidate.
  if (isAppAsset(url)) {
    e.respondWith((async () => {
      const cached = await caches.match(req, { ignoreSearch: true });
      const net = fetch(req).then((res) => {
        if (res && res.ok) caches.open(CACHE).then((c) => c.put(req, res.clone()));
        return res;
      }).catch(() => null);
      return cached || (await net) || Response.error();
    })());
  }
});
