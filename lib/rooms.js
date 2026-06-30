'use strict';
// Blind E2EE relay. A room links one HOST (the desktop agent) to one or more
// CLIENTS (phones). The relay only ever moves opaque ciphertext between them; it
// holds no encryption key and cannot read a single keystroke or mouse move.

const ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'; // no 0/O/1/I ambiguity
const CODE_LEN = 6;
const MAX_ROOMS = 800;
const MAX_CLIENTS_PER_ROOM = 4;
const MAX_ROOMS_PER_IP = 6;
const IDLE_MS = 30 * 60 * 1000; // close a room after 30 min with no traffic

function randomCode() {
  let s = '';
  for (let i = 0; i < CODE_LEN; i++) s += ALPHABET[Math.floor(Math.random() * ALPHABET.length)];
  return s;
}

class Rooms {
  constructor() {
    this.byCode = new Map();      // code -> room
    this.ipHostCount = new Map(); // ip -> number of rooms hosted
    setInterval(() => this._sweep(), 60 * 1000).unref?.();
  }

  _sweep() {
    const now = Date.now();
    for (const [code, room] of this.byCode) {
      if (now - room.lastActivity > IDLE_MS) this.closeRoom(code, 'idle');
    }
  }

  hostCount() { return this.byCode.size; }

  // A host may request a specific `desired` code (a persistent agent derives a
  // stable room from its saved key, so remembered phones reconnect to the same
  // address). The relay reclaims a desired code held by a dead/stale host, but
  // refuses one held by a different live host (collision) → 'room_taken'.
  createRoom(hostWs, ip, desired) {
    if (this.byCode.size >= MAX_ROOMS) return { error: 'relay_full' };
    const hosted = this.ipHostCount.get(ip) || 0;
    if (hosted >= MAX_ROOMS_PER_IP) return { error: 'too_many_rooms' };

    let code;
    if (desired) {
      code = String(desired).toUpperCase();
      if (!/^[A-Z0-9]{4,8}$/.test(code) || /[O01I]/.test(code)) return { error: 'bad_code' };
      const existing = this.byCode.get(code);
      if (existing) {
        // reclaim if the previous host is gone (restart/idle), else collision.
        if (existing.host && existing.host.readyState === 1) return { error: 'room_taken' };
        this.closeRoom(code, 'reclaimed');
      }
    } else {
      do { code = randomCode(); } while (this.byCode.has(code));
    }

    const room = {
      code, host: hostWs, ip,
      clients: new Map(), nextId: 1,
      createdAt: Date.now(), lastActivity: Date.now(),
    };
    this.byCode.set(code, room);
    this.ipHostCount.set(ip, hosted + 1);
    hostWs._room = code;
    hostWs._role = 'host';
    return { code };
  }

  join(code, clientWs) {
    const room = this.byCode.get(code);
    if (!room) return { error: 'no_such_room' };
    if (room.clients.size >= MAX_CLIENTS_PER_ROOM) return { error: 'room_full' };
    const id = room.nextId++;
    room.clients.set(id, clientWs);
    clientWs._room = code;
    clientWs._role = 'client';
    clientWs._cid = id;
    room.lastActivity = Date.now();
    // tell the host a peer joined (so it can send a welcome/handshake). We also
    // forward the client's real IP so the host owner can see who is connected —
    // metadata only; the relay stays blind to the E2EE ciphertext itself.
    this._send(room.host, { t: 'peer', event: 'join', id, ip: clientWs._ip || '' });
    return { id };
  }

  // client -> host
  fromClient(clientWs, payload) {
    const room = this.byCode.get(clientWs._room);
    if (!room) return;
    room.lastActivity = Date.now();
    this._send(room.host, { t: 'data', from: clientWs._cid, payload });
  }

  // host -> a specific client (or broadcast if no `to`)
  fromHost(hostWs, payload, to) {
    const room = this.byCode.get(hostWs._room);
    if (!room) return;
    room.lastActivity = Date.now();
    if (to == null) {
      for (const ws of room.clients.values()) this._send(ws, { t: 'data', payload });
    } else {
      const ws = room.clients.get(to);
      if (ws) this._send(ws, { t: 'data', payload });
    }
  }

  leave(ws) {
    const code = ws._room;
    if (!code) return;
    const room = this.byCode.get(code);
    if (!room) return;
    if (ws._role === 'host') {
      this.closeRoom(code, 'host_left');
    } else {
      room.clients.delete(ws._cid);
      room.lastActivity = Date.now();
      this._send(room.host, { t: 'peer', event: 'leave', id: ws._cid });
    }
  }

  closeRoom(code, reason) {
    const room = this.byCode.get(code);
    if (!room) return;
    for (const ws of room.clients.values()) {
      this._send(ws, { t: 'closed', reason });
      try { ws.close(); } catch { /* ignore — socket may already be gone */ }
    }
    if (reason !== 'host_left') {
      this._send(room.host, { t: 'closed', reason });
      try { room.host.close(); } catch { /* ignore — socket may already be gone */ }
    }
    const hosted = this.ipHostCount.get(room.ip) || 1;
    if (hosted <= 1) this.ipHostCount.delete(room.ip);
    else this.ipHostCount.set(room.ip, hosted - 1);
    this.byCode.delete(code);
  }

  _send(ws, obj) {
    if (ws && ws.readyState === 1) {
      try { ws.send(JSON.stringify(obj)); } catch { /* ignore — race between readyState check and send */ }
    }
  }
}

module.exports = { Rooms, CODE_LEN, ALPHABET };
