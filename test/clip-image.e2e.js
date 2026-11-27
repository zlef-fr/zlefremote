// End-to-end check of clipboard IMAGES (and text) through the browser desktop
// remote, against a REAL agent writing a REAL X clipboard. Not part of CI — it
// needs two displays, two browsers and a running agent — but it is the exact
// script used to verify the feature, so it stays in the repo.
//
//   # the machine to control: its own X display, xclip, a real agent
//   Xvfb :96 -screen 0 1400x900x24 &
//   DISPLAY=:96 ./zlefremote-agent -mode lan -port 9791 -machine
//
//   # the controlling side
//   Xvfb :97 -screen 0 1400x900x24 &
//   ZR_URL='https://127.0.0.1:9791/#k=…' ZR_HOST_DISPLAY=:96 \
//     DISPLAY=:97 NODE_PATH=/usr/lib/node_modules node test/clip-image.e2e.js
//
// On the host display it opens a second browser on a page that records every
// paste it receives — that is the "app you paste into" over there. What it
// proves: an image copied on THIS computer, then Ctrl+V while in control,
// lands as that same image in the app on the OTHER one (and text the same way);
// and a computer whose agent can't take images says so instead of nothing.
const { chromium } = require('playwright');

const url = process.env.ZR_URL;
const hostDisplay = process.env.ZR_HOST_DISPLAY || ':96';
if (!url) { console.error('set ZR_URL to the agent pairing link'); process.exit(2); }
const [base, frag] = url.split('#');
const deskURL = base.includes('/r/') ? base.replace('/r/', '/d/') + '#' + frag : base.replace(/\/$/, '') + '/d#' + frag;

const TARGET = `data:text/html,<title>target</title><body style="margin:0;background:%23222">
<div id=t contenteditable style="position:fixed;inset:0;color:%23fff">paste here</div>
<script>
window.got = [];
document.addEventListener('paste', async (e) => {
  const dt = e.clipboardData, rec = { types: [...dt.types], text: dt.getData('text/plain') };
  const f = [...dt.items].find((i) => i.kind === 'file' && i.type.startsWith('image/'));
  if (f) { const b = await createImageBitmap(f.getAsFile()); rec.img = b.width + 'x' + b.height; }
  window.got.push(rec);
});
document.getElementById('t').focus();
</script>`;

const fail = (msg) => { console.error('FAIL:', msg); process.exitCode = 1; };
const ok = (msg) => console.log('ok -', msg);

(async () => {
  // ── the other computer: a browser on its display, focused paste target ──
  const hostBrowser = await chromium.launch({ headless: false, executablePath: process.env.ZR_CHROME || undefined, env: { ...process.env, DISPLAY: hostDisplay } });
  const target = await (await hostBrowser.newContext({ viewport: { width: 1400, height: 900 } })).newPage();
  await target.goto(TARGET);
  await target.click('#t');
  const pasted = () => target.evaluate(() => window.got);

  // ── this computer ──
  const browser = await chromium.launch({ headless: false, executablePath: process.env.ZR_CHROME || undefined });   // pointer lock needs a window
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 760 }, ignoreHTTPSErrors: true });
  const origin = new URL(deskURL).origin;
  // ZR_NO_READ=1: never grant clipboard-read, the common case on a first visit.
  // Ctrl+V must still carry the clipboard (the browser's paste event needs no
  // permission); only the explicit button has to say it was refused.
  const noRead = !!process.env.ZR_NO_READ;
  await ctx.grantPermissions(noRead ? ['clipboard-write'] : ['clipboard-read', 'clipboard-write'], { origin });
  const page = await ctx.newPage();
  const errors = [];
  page.on('pageerror', (e) => errors.push('pageerror: ' + e.message));
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
  const toasts = () => page.$$eval('.toast', (ns) => ns.map((n) => n.textContent));

  await page.goto(deskURL, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#takeover:not([hidden])', { timeout: 25000 });
  ok('paired, screen streaming');

  // a 900x600 picture with real detail (a gradient + noise ≈ 1.5 MB of PNG:
  // dozens of chunks, so the chunking and the progress toast both run)
  const copyImage = (w, h) => page.evaluate(async ([w, h]) => {
    const c = document.createElement('canvas'); c.width = w; c.height = h;
    const g = c.getContext('2d'), id = g.createImageData(w, h);
    for (let i = 0; i < id.data.length; i += 4) {
      id.data[i] = (i / 4) % w % 256; id.data[i + 1] = Math.random() * 255; id.data[i + 2] = 128; id.data[i + 3] = 255;
    }
    g.putImageData(id, 0, 0);
    const blob = await new Promise((r) => c.toBlob(r, 'image/png'));
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })]);
    return blob.size;
  }, [w, h]);

  // take control
  await page.mouse.click(640, 400);
  await page.waitForTimeout(800);
  (await page.evaluate(() => document.pointerLockElement !== null)) ? ok('in control') : fail('pointer lock not granted');
  // put the host pointer over the target, then click it so it has focus
  await page.mouse.click(640, 400);
  await page.waitForTimeout(400);

  // 1. image, Ctrl+V while in control
  const size = await copyImage(900, 600);
  await page.keyboard.press('Control+V');
  let got = [];
  for (let i = 0; i < 60 && !got.length; i++) { await page.waitForTimeout(250); got = await pasted(); }
  const last = got[got.length - 1] || {};
  last.img === '900x600'
    ? ok(`Ctrl+V pasted the copied image over there (${Math.round(size / 1024)} KB → ${last.img})`)
    : fail('image paste on the host: ' + JSON.stringify(got));
  console.log('   toasts:', JSON.stringify(await toasts()));

  // 2. text, Ctrl+V while in control: the host must paste the NEW text,
  // which only holds if the text crossed before the key did
  const phrase = 'clip-' + Math.random().toString(36).slice(2, 8);
  await page.evaluate((s) => navigator.clipboard.writeText(s), phrase);
  await page.keyboard.press('Control+V');
  let n0 = got.length;
  for (let i = 0; i < 40; i++) { await page.waitForTimeout(150); got = await pasted(); if (got.length > n0) break; }
  const t2 = got[got.length - 1] || {};
  t2.text === phrase && !t2.img ? ok('Ctrl+V right after copying text pastes that text') : fail('text paste: ' + JSON.stringify(t2));

  // 3. image again via the toolbar button (explicit read), not in control
  await page.keyboard.down('ControlRight'); await page.keyboard.up('ControlRight');
  await page.waitForTimeout(500);
  await copyImage(320, 200);
  await page.click('#btnClip', { force: true });
  await page.waitForTimeout(2500);
  const tb = await toasts();
  if (noRead) {
    tb.some((s) => /refused|refusée/.test(s)) ? ok('toolbar without permission says so: ' + tb.pop()) : fail('toolbar without permission: ' + JSON.stringify(tb));
  } else tb.some((s) => /image (sent|envoyée)/.test(s)) ? ok('toolbar button sends an image: ' + tb.filter((s) => /image/.test(s)).pop()) : fail('toolbar image: ' + JSON.stringify(tb));

  errors.length ? fail('page errors: ' + errors.join(' | ')) : ok('no page errors');
  await page.screenshot({ path: '/tmp/clip-image-e2e.png' });
  await browser.close();
  await hostBrowser.close();
})().catch((e) => { console.error(e); process.exit(1); });
