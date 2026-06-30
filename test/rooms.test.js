'use strict';
// Tests for lib/rooms.js — run with: node --test test/
// Uses Node.js built-in test runner (node:test, available since Node 18).
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { Rooms, CODE_LEN, ALPHABET } = require('../lib/rooms');

// Minimal WebSocket mock: tracks sent messages and exposes readyState.
function fakeWs(ip = '1.2.3.4') {
  const ws = {
    _ip: ip,
    readyState: 1,
    _sent: [],
    send(data) { this._sent.push(JSON.parse(data)); },
    close() { this.readyState = 3; },
  };
  return ws;
}

test('randomCode has correct length and uses the allowed alphabet', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const r = rooms.createRoom(host, '10.0.0.1');
  assert.ok(!r.error, 'should succeed');
  assert.equal(r.code.length, CODE_LEN);
  const allowed = new Set(ALPHABET.split(''));
  for (const ch of r.code) assert.ok(allowed.has(ch), `unexpected char: ${ch}`);
});

test('createRoom → host gets hosted message', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const r = rooms.createRoom(host, '10.0.0.1');
  assert.ok(!r.error);
  assert.equal(host._sent.length, 0, 'createRoom does not send to host directly');
  assert.equal(host._room, r.code);
  assert.equal(host._role, 'host');
});

test('join → client gets joined; host gets peer-join', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const client = fakeWs('5.6.7.8');
  const j = rooms.join(code, client);
  assert.ok(!j.error);
  assert.equal(client._role, 'client');
  assert.equal(host._sent.length, 1);
  assert.equal(host._sent[0].t, 'peer');
  assert.equal(host._sent[0].event, 'join');
});

test('fromClient relays payload to host', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const client = fakeWs();
  rooms.join(code, client);
  host._sent = []; // clear peer-join message
  rooms.fromClient(client, 'CIPHERTEXT');
  assert.equal(host._sent.length, 1);
  assert.equal(host._sent[0].t, 'data');
  assert.equal(host._sent[0].payload, 'CIPHERTEXT');
});

test('fromHost broadcasts to all clients when no `to`', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const c1 = fakeWs(); const c2 = fakeWs();
  rooms.join(code, c1); rooms.join(code, c2);
  rooms.fromHost(host, 'PAYLOAD', null);
  assert.equal(c1._sent.length, 1);
  assert.equal(c2._sent.length, 1);
});

test('fromHost targets a specific client when `to` is set', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const c1 = fakeWs(); const c2 = fakeWs();
  const j1 = rooms.join(code, c1); rooms.join(code, c2);
  rooms.fromHost(host, 'PAYLOAD', j1.id);
  assert.equal(c1._sent.length, 1);
  assert.equal(c2._sent.length, 0);
});

test('leave client notifies host', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const client = fakeWs();
  rooms.join(code, client);
  host._sent = [];
  rooms.leave(client);
  assert.equal(host._sent.length, 1);
  assert.equal(host._sent[0].t, 'peer');
  assert.equal(host._sent[0].event, 'leave');
  assert.equal(rooms.hostCount(), 1, 'room still open after client leaves');
});

test('leave host closes the room and all clients', () => {
  const rooms = new Rooms();
  const host = fakeWs();
  const { code } = rooms.createRoom(host, '10.0.0.1');
  const client = fakeWs();
  rooms.join(code, client);
  rooms.leave(host);
  assert.equal(rooms.hostCount(), 0);
  assert.equal(client._sent.length, 1);
  assert.equal(client._sent[0].t, 'closed');
});

test('MAX_ROOMS_PER_IP enforced', () => {
  const rooms = new Rooms();
  const IP = '9.9.9.9';
  let last;
  for (let i = 0; i < 6; i++) {
    last = rooms.createRoom(fakeWs(IP), IP);
    assert.ok(!last.error, `room ${i + 1} should succeed`);
  }
  const over = rooms.createRoom(fakeWs(IP), IP);
  assert.equal(over.error, 'too_many_rooms');
});

test('desired code is reclaimed if previous host is gone', () => {
  const rooms = new Rooms();
  const host1 = fakeWs();
  const r1 = rooms.createRoom(host1, '10.0.0.1', 'AAAA22');
  assert.equal(r1.code, 'AAAA22');
  host1.readyState = 3; // simulate disconnect
  const host2 = fakeWs();
  const r2 = rooms.createRoom(host2, '10.0.0.2', 'AAAA22');
  assert.equal(r2.code, 'AAAA22', 'stale room should be reclaimed');
});

test('desired code rejected when live host holds it', () => {
  const rooms = new Rooms();
  const host1 = fakeWs();
  rooms.createRoom(host1, '10.0.0.1', 'BBBB22');
  const host2 = fakeWs();
  const r2 = rooms.createRoom(host2, '10.0.0.2', 'BBBB22');
  assert.equal(r2.error, 'room_taken');
});

test('invalid desired code formats are rejected', () => {
  const rooms = new Rooms();
  const cases = ['ABC', 'AB C123', 'AAAIIII', '0BBBBB']; // too short; space; I/0 ambiguous chars
  for (const bad of cases) {
    const r = rooms.createRoom(fakeWs(), '1.2.3.4', bad);
    assert.equal(r.error, 'bad_code', `expected bad_code for "${bad}"`);
  }
});
