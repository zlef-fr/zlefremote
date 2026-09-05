// Room lifetime, in isolation: no sockets, no browser, just the bookkeeping.
//
//   node test/rooms.test.js
//
// The case that matters is the permanently-hosted room. A Zlef Home agent hosts
// its own room around the clock so a saved phone can connect at any hour, which
// means the room is idle almost all the time — and the idle sweep used to close
// it every 30 minutes.
const assert = require('assert');
const { Rooms } = require('../lib/rooms');

const OPEN = 1, CLOSED = 3;
const fakeWs = (state = OPEN) => ({ readyState: state, sent: [], send(s) { this.sent.push(s); }, close() { this.readyState = CLOSED; } });
const frames = (ws) => ws.sent.map((s) => JSON.parse(s).t);

function test(name, fn) {
  try { fn(); console.log(`  PASS  ${name}`); } catch (e) {
    console.log(`  FAIL  ${name}\n        ${e.message}`); process.exitCode = 1;
  }
}

test('a room whose host is still connected survives the idle sweep', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1', 'AGENTA');
  rooms.byCode.get(code).lastActivity = Date.now() - 6 * 60 * 60 * 1000; // idle 6 h
  rooms._sweep();
  assert.ok(rooms.byCode.has(code), 'the agent room was swept while its host held the socket');
  assert.deepStrictEqual(frames(host), [], 'a live host was told its room closed');
});

test('a room whose host is gone is reclaimed', () => {
  const rooms = new Rooms();
  const host = fakeWs(CLOSED);
  const { code } = rooms.createRoom(host, '10.0.0.2', 'DEADXX');
  rooms.byCode.get(code).lastActivity = Date.now() - 6 * 60 * 60 * 1000;
  rooms._sweep();
  assert.ok(!rooms.byCode.has(code), 'an orphaned room kept its code forever');
});

test('a fresh orphan is left alone until it goes idle', () => {
  const rooms = new Rooms();
  const { code } = rooms.createRoom(fakeWs(CLOSED), '10.0.0.3', 'FRESHX');
  rooms._sweep();
  assert.ok(rooms.byCode.has(code), 'a room was swept before its idle window elapsed');
});

test('the host still learns when its room is closed for another reason', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.4', 'AGENTB');
  rooms.closeRoom(code, 'idle');
  assert.deepStrictEqual(frames(host), ['closed'], 'the host was not notified');
  assert.ok(!rooms.byCode.has(code));
});

test('a phone joining and leaving does not take the room with it', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.5', 'AGENTC');
  const phone = fakeWs();
  const { id } = rooms.join(code, phone);
  rooms.leave(phone);
  assert.ok(rooms.byCode.has(code), 'the room died with its visitor');
  assert.deepStrictEqual(frames(host), ['peer', 'peer'], 'host missed the join/leave pair');
  assert.strictEqual(id, 1);
});
