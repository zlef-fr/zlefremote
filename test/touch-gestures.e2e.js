// Every multi-touch gesture, driven against a REAL agent with synthetic
// touches, asserted on what the agent actually injected.
//
// Not part of CI — it needs a browser and a running agent — but it is the check
// that proves the whole chain: recognizer → intent → per-OS chord → injector.
// The stub injector is enough (and safer): it logs what it would have done.
//
//   # terminal 1: the computer being controlled
//   cd agent && go build -o /tmp/zr-agent .        # stub backend, no CGO
//   /tmp/zr-agent -mode lan -port 9791 -machine -no-telemetry > /tmp/zr-agent.log 2>&1 &
//
//   # terminal 2
//   ZR_URL="$(grep -o 'url=.*' /tmp/zr-agent.log | cut -d= -f2-)" \
//     ZR_AGENT_LOG=/tmp/zr-agent.log \
//     NODE_PATH=/usr/lib/node_modules node test/touch-gestures.e2e.js
//
// A relay URL (https://remote.zlef.fr/r/AB12CD#k=…) works just as well.
const { chromium } = require('playwright');
const fs = require('fs');

const url = process.env.ZR_URL;
const logFile = process.env.ZR_AGENT_LOG || '/tmp/zr-agent.log';

const mark = () => fs.readFileSync(logFile, 'utf8').length;
const since = (off) => fs.readFileSync(logFile, 'utf8').slice(off);

(async () => {
  const browser = await chromium.launch();
  const ctx = await browser.newContext({
    hasTouch: true, isMobile: true, viewport: { width: 412, height: 915 },
    ignoreHTTPSErrors: true,
  });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (m.type() === 'error') console.log('  [console]', m.text()); });
  await page.goto(url, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // touch helpers: raw CDP so we control every finger independently
  const cdp = await ctx.newCDPSession(page);
  const send = (type, points) =>
    cdp.send('Input.dispatchTouchEvent', { type, touchPoints: points });
  const pt = (id, x, y) => ({ x, y, id });

  const pad = await page.locator('#pad').boundingBox();
  if (!pad) throw new Error('trackpad not visible — did pairing fail?');
  const cx = pad.x + pad.width / 2, cy = pad.y + pad.height / 2;

  async function gesture(name, fingers, steps, dx, dy, stepMs = 16) {
    const off = mark();
    let pts = fingers.map((f, i) => pt(i, cx + f[0], cy + f[1]));
    await send('touchStart', pts);
    for (let s = 0; s < steps; s++) {
      pts = pts.map((p, i) => pt(i, p.x + (Array.isArray(dx) ? dx[i] : dx), p.y + (Array.isArray(dy) ? dy[i] : dy)));
      await send('touchMove', pts);
      await page.waitForTimeout(stepMs);
    }
    await send('touchEnd', []);
    await page.waitForTimeout(350);
    const out = since(off).split('\n').filter((l) => l.includes('[stub]'));
    console.log(`\n▸ ${name}\n${out.map((l) => '   ' + l.replace(/^.*\[stub\] /, '')).join('\n') || '   (nothing)'}`);
    return out.join('\n');
  }

  const results = {};
  results.threeLeft = await gesture('three fingers left', [[-40, 0], [0, 0], [40, 0]], 6, -14, 0);
  results.threeRight = await gesture('three fingers right', [[-40, 0], [0, 0], [40, 0]], 6, 14, 0);
  results.threeUp = await gesture('three fingers up', [[-40, 0], [0, 0], [40, 0]], 6, 0, -14);
  results.threeDown = await gesture('three fingers down', [[-40, 0], [0, 0], [40, 0]], 6, 0, 14);
  results.flick = await gesture('two-finger flick right', [[-30, 0], [30, 0]], 6, 18, 0);
  results.scroll = await gesture('two-finger scroll down', [[-30, -60], [30, -60]], 8, 0, 14, 40);
  results.pinch = await gesture('pinch out', [[-20, 0], [20, 0]], 8, [-14, 14], 0);
  results.pinchIn = await gesture('pinch in', [[-90, 0], [90, 0]], 8, [14, -14], 0);

  // three-finger tap → middle click
  {
    const off = mark();
    const pts = [pt(0, cx - 40, cy), pt(1, cx, cy), pt(2, cx + 40, cy)];
    await send('touchStart', pts);
    await page.waitForTimeout(60);
    await send('touchEnd', []);
    await page.waitForTimeout(300);
    results.threeTap = since(off);
    console.log('\n▸ three-finger tap\n   ' +
      (results.threeTap.split('\n').filter((l) => l.includes('[stub]')).map((l) => l.replace(/^.*\[stub\] /, '')).join('\n   ') || '(nothing)'));
  }

  await page.screenshot({ path: '/tmp/zr-client.png' });
  await browser.close();

  const expect = (name, hay, needle) => {
    const ok = hay.includes(needle);
    console.log(`${ok ? '  PASS' : '  FAIL'}  ${name} → ${needle}`);
    return ok;
  };
  console.log('\n── verdict ──');
  let all = true;
  all &= expect('3 left = previous app', results.threeLeft, 'key "tab" mods=[alt shift]');
  all &= expect('3 right = next app', results.threeRight, 'key "tab" mods=[alt]');
  all &= expect('3 up = overview', results.threeUp, 'key "meta" mods=[]');
  all &= expect('3 down = show desktop', results.threeDown, 'key "d" mods=[meta]');
  all &= expect('2 flick right = back', results.flick, 'key "left" mods=[alt]');
  all &= expect('2 drag = scroll', results.scroll, 'scroll');
  all &= expect('pinch out = zoom in', results.pinch, 'key "=" mods=[ctrl]');
  all &= expect('pinch in = zoom out', results.pinchIn, 'key "-" mods=[ctrl]');
  all &= expect('3 tap = middle click', results.threeTap, 'click middle');
  const flickScrolled = /scroll [+-]\d+/.test(results.flick);
  console.log(`${flickScrolled ? '  FAIL' : '  PASS'}  the flick did not also scroll`);
  all &= !flickScrolled;
  process.exit(all ? 0 : 1);
})();
