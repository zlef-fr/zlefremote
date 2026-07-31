import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

/// End-to-end encryption for the ZlefRemote link.
///
/// One 256-bit key per computer. It is minted by the agent, travels only in the
/// fragment of the pairing URL (`#k=…`, never sent to any server) and is stored
/// on this phone in the Android Keystore. Every command is sealed before it
/// touches the socket, so the relay — and anyone on the Wi-Fi — only ever moves
/// opaque `iv.ciphertext` strings.
///
/// Wire format is shared byte-for-byte with `agent/crypto.go` and the web client
/// `public/app/js/crypto.js`: AES-256-GCM, 12-byte IV, frame =
/// `base64url(iv) + "." + base64url(ciphertext‖tag)`, no padding, no AAD.
/// A wrong key simply fails to open — that is also the authentication gate.
class ZrCrypto {
  ZrCrypto._(this._key, this.keyB64);

  static final _gcm = AesGcm.with256bits();
  static const _macLen = 16; // GCM tag, appended to the ciphertext by Go/WebCrypto

  final SecretKey _key;

  /// The raw key, base64url (unpadded) — what travels in a pairing link.
  final String keyB64;

  /// Builds a sealer from a base64url key. Throws when the key is not 32 bytes.
  static Future<ZrCrypto> fromKeyB64(String keyB64) async {
    final raw = b64uDecode(keyB64);
    if (raw.length != 32) {
      throw const FormatException('link key must be 32 bytes');
    }
    return ZrCrypto._(SecretKey(raw), keyB64);
  }

  /// Seals a command object into a wire frame.
  Future<String> seal(Object command) async {
    final nonce = _gcm.newNonce();
    final box = await _gcm.encrypt(
      utf8.encode(jsonEncode(command)),
      secretKey: _key,
      nonce: nonce,
    );
    final ct = Uint8List(box.cipherText.length + box.mac.bytes.length)
      ..setAll(0, box.cipherText)
      ..setAll(box.cipherText.length, box.mac.bytes);
    return '${b64uEncode(nonce)}.${b64uEncode(ct)}';
  }

  /// Opens a wire frame. Throws on a wrong key, a tampered frame or bad JSON —
  /// callers drop the frame silently, exactly like the agent does.
  Future<Map<String, dynamic>> open(String frame) async {
    final dot = frame.indexOf('.');
    if (dot < 0) throw const FormatException('bad frame');
    final nonce = b64uDecode(frame.substring(0, dot));
    final sealed = b64uDecode(frame.substring(dot + 1));
    if (sealed.length < _macLen) throw const FormatException('short frame');
    final cut = sealed.length - _macLen;
    final clear = await _gcm.decrypt(
      SecretBox(
        sealed.sublist(0, cut),
        nonce: nonce,
        mac: Mac(sealed.sublist(cut)),
      ),
      secretKey: _key,
    );
    final decoded = jsonDecode(utf8.decode(clear));
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('frame is not an object');
    }
    return decoded;
  }

  // ── room derivation ────────────────────────────────────────────────────────
  // A `--remember` agent derives its relay room from its persistent key instead
  // of taking a random one, so a saved phone that holds only the key can
  // recompute where that computer will be waiting. Must stay identical to
  // agent/identity.go and crypto.js — changing it strands every saved device.

  static const _roomAlphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
  static const _roomDomain = 'zlefremote-room-v1\x00';

  static Future<String> deriveRoom(String keyB64) async {
    final raw = b64uDecode(keyB64);
    if (raw.length != 32) {
      throw const FormatException('link key must be 32 bytes');
    }
    final input = <int>[...utf8.encode(_roomDomain), ...raw];
    final digest = await Sha256().hash(input);
    final out = StringBuffer();
    for (var i = 0; i < 6; i++) {
      out.write(_roomAlphabet[digest.bytes[i] & 31]); // 32 symbols → low 5 bits
    }
    return out.toString();
  }

  // ── base64url, unpadded (Go's RawURLEncoding) ─────────────────────────────

  static String b64uEncode(List<int> bytes) =>
      base64Url.encode(bytes).replaceAll('=', '');

  static Uint8List b64uDecode(String s) {
    final padded = s.padRight(s.length + ((4 - s.length % 4) % 4), '=');
    return base64Url.decode(padded);
  }
}

/// Top-level aliases — the screen-frame decoder needs raw base64url too.
Uint8List b64uDecode(String s) => ZrCrypto.b64uDecode(s);
String b64uEncode(List<int> b) => ZrCrypto.b64uEncode(b);
