// E2EE: AES-256-GCM. The key lives only in the URL fragment (never sent to any
// server). Every command is sealed before it touches the WebSocket; the relay
// only ever moves opaque "iv.ciphertext" strings.
const ZRCrypto = (() => {
  let key = null;

  const b64u = {
    enc(buf) {
      let s = btoa(String.fromCharCode(...new Uint8Array(buf)));
      return s.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    },
    dec(str) {
      str = str.replace(/-/g, '+').replace(/_/g, '/');
      while (str.length % 4) str += '=';
      const bin = atob(str);
      const out = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
      return out;
    },
  };

  async function init(rawKeyB64u) {
    const raw = b64u.dec(rawKeyB64u);
    if (raw.length !== 32) throw new Error('bad key length');
    key = await crypto.subtle.importKey('raw', raw, 'AES-GCM', false, ['encrypt', 'decrypt']);
  }

  // deriveRoom maps a base64url key to its stable 6-char relay room — byte-for-
  // byte identical to the agent (agent/identity.go) so a saved phone holding the
  // key alone can recompute the exact room a persistent agent will host on.
  const ROOM_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
  async function deriveRoom(rawKeyB64u) {
    const raw = b64u.dec(rawKeyB64u);
    if (raw.length !== 32) throw new Error('bad key length');
    const domain = new TextEncoder().encode('zlefremote-room-v1\0');
    const buf = new Uint8Array(domain.length + raw.length);
    buf.set(domain, 0); buf.set(raw, domain.length);
    const h = new Uint8Array(await crypto.subtle.digest('SHA-256', buf));
    let room = '';
    for (let i = 0; i < 6; i++) room += ROOM_ALPHABET[h[i] & 31];
    return room;
  }

  async function seal(obj) {
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const pt = new TextEncoder().encode(JSON.stringify(obj));
    const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, pt);
    return b64u.enc(iv) + '.' + b64u.enc(ct);
  }

  async function open(frame) {
    const dot = frame.indexOf('.');
    if (dot < 0) throw new Error('bad frame');
    const iv = b64u.dec(frame.slice(0, dot));
    const ct = b64u.dec(frame.slice(dot + 1));
    const pt = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, ct);
    return JSON.parse(new TextDecoder().decode(pt));
  }

  return { init, seal, open, deriveRoom, ready: () => !!key };
})();
