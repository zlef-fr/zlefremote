// End-to-end check of the browser desktop remote (/d) against a REAL agent.
// Not part of CI — it needs a display, a browser and a running agent — but it
// is the exact script used to verify the client, so it stays in the repo.
//
//   # terminal 1: the machine to control (its own X display here)
//   Xvfb :96 -screen 0 1400x900x24 &
//   DISPLAY=:96 xterm -e "cat > /tmp/typed.txt" &
//   DISPLAY=:96 ./dist/zlefremote-agent-linux-amd64 -mode lan -port 9791 -machine
//
//   # terminal 2: the controlling side
//   Xvfb :97 -screen 0 1400x900x24 &
//   ZR_URL='https://192.168.1.24:9791/#k=…' ZR_HOST_DISPLAY=:96 \
//     DISPLAY=:97 NODE_PATH=/usr/lib/node_modules node test/web-remote.e2e.js
//
// A relay URL (https://remote.zlef.fr/r/AB12CD#k=…) works just as well.
const { chromium } = require('playwright');
const fs = require('fs');
const { execFileSync } = require('child_process');

const url = process.env.ZR_URL;
const hostDisplay = process.env.ZR_HOST_DISPLAY || ':96';
const typedFile = process.env.ZR_TYPED_FILE || '/tmp/typed.txt';
if (!url) { console.error('set ZR_URL to the agent pairing link'); process.exit(2); }

// /r/<room> (phone) and /d/<room> (desktop) are the same room and key
const [base, frag] = url.split('#');
const deskURL = base.includes('/r/')
  ? base.replace('/r/', '/d/') + '#' + frag
  : base.replace(/\/$/, '') + '/d#' + frag;

// execFile, not a shell string: the display comes from the environment and has
// no business being interpolated into a command line.
const hostPointer = () =>
  execFileSync('xdotool', ['getmouselocation'], { env: { ...process.env, DISPLAY: hostDisplay } })
    .toString().trim();

(async () => {
  const browser = await chromium.launch({
    headless: false,   // pointer lock needs a real window
    executablePath: process.env.ZR_CHROME || undefined,
  });
  // LAN mode serves a self-signed certificate (browsers need a secure context
  // for crypto.subtle); a user clicks "Advanced → Proceed", we set this flag.
  const page = await browser.newPage({ viewport: { width: 1280, height: 760 }, ignoreHTTPSErrors: true });
  const errors = [];
  page.on('pageerror', (e) => errors.push('pageerror: ' + e.message));
  page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });

  const fail = (msg) => { console.error('FAIL:', msg); process.exitCode = 1; };
  const ok = (msg) => console.log('ok —', msg);

  await page.goto(deskURL, { waitUntil: 'domcontentloaded' });

  const ctx = await page.evaluate(() => ({ secure: isSecureContext, subtle: typeof crypto.subtle }));
  ctx.secure && ctx.subtle === 'object' ? ok('secure context (WebCrypto available)') : fail('not a secure context: ' + JSON.stringify(ctx));

  await page.waitForSelector('#takeover:not([hidden])', { timeout: 25000 });
  ok('screen stream started — ' + (await page.textContent('#stats')));

  // clicking the picture takes control and warps the remote pointer there
  await page.mouse.click(640, 400);
  await page.waitForTimeout(1000);
  (await page.evaluate(() => document.pointerLockElement !== null)) ? ok('pointer locked') : fail('pointer lock not granted');
  const atClick = hostPointer();

  await page.mouse.move(760, 500);
  await page.waitForTimeout(700);
  const atMove = hostPointer();
  atMove !== atClick ? ok(`pointer follows (${atClick} → ${atMove})`) : fail('the host pointer did not move');

  const phrase = 'web-remote-' + Math.random().toString(36).slice(2, 8);
  await page.keyboard.type(phrase, { delay: 40 });
  await page.keyboard.press('Enter');
  await page.waitForTimeout(1500);
  const typed = fs.readFileSync(typedFile, 'utf8').trim().split('\n').pop();
  typed === phrase ? ok('typing arrives verbatim') : fail(`typed "${typed}", expected "${phrase}"`);

  // Right Ctrl on its own gives control back
  await page.keyboard.down('ControlRight');
  await page.waitForTimeout(120);
  await page.keyboard.up('ControlRight');
  await page.waitForTimeout(600);
  (await page.evaluate(() => document.pointerLockElement === null)) ? ok('Right Ctrl released control') : fail('Right Ctrl did not release control');

  errors.length ? fail('console errors: ' + errors.join(' | ')) : ok('no console errors');
  await browser.close();
})();
