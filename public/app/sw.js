/* ZlefRemote — service-worker tombstone.
 *
 * The phone client used to be an installable PWA. It is a native Android app
 * now (fr.zlef.remote, /app/zlefremote.apk), so this file no longer caches
 * anything: it exists only to retire the workers already installed on people's
 * phones. Without it, a browser holding the old zr-pwa-v15 cache would keep
 * serving its shell forever and those users would never see the site change.
 *
 * It claims control, deletes every cache we ever created, unregisters itself
 * and reloads, once, the pages the old worker was serving. The web client itself still works — it is
 * the no-install fallback, and the UI the agent embeds for LAN mode — it simply
 * is not a PWA any more.
 */

self.addEventListener('install', () => self.skipWaiting());

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Only pages the OLD worker controlled get reloaded: they may be running a
    // frozen shell. Taken BEFORE claim() — after it, every open page counts as
    // controlled, including a fresh one served straight from the network, and
    // reloading that one is how a stray register() turned into an endless loop.
    const stale = await self.clients.matchAll({ type: 'window' });
    const names = await caches.keys();
    await Promise.all(
      names.filter((n) => n.startsWith('zr-')).map((n) => caches.delete(n)),
    );
    await self.clients.claim();
    await self.registration.unregister();
    for (const client of stale) {
      client.navigate(client.url).catch(() => {});
    }
  })());
});

// never intercept anything while we wind down
self.addEventListener('fetch', () => {});
