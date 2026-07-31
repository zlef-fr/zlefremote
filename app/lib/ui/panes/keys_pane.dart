import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../core/caps.dart';
import '../../core/i18n/i18n.dart';
import '../../core/session.dart';
import '../../core/settings.dart';
import '../theme.dart';
import '../widgets.dart';

/// The keyboard.
///
/// Text goes out as text (`{t:'text'}`), never as synthesised key codes, so the
/// computer's own layout does the work — AZERTY, accents, dead keys and emoji
/// all land correctly. Only the keys that have no character — arrows, Escape,
/// F-keys, chords — travel as key presses.
class KeysPane extends StatefulWidget {
  const KeysPane({super.key});

  @override
  State<KeysPane> createState() => _KeysPaneState();
}

class _KeysPaneState extends State<KeysPane> {
  final _controller = TextEditingController();
  final _focus = FocusNode();
  String _previous = '';
  final _mods = <String>{};

  /// What the user just typed, echoed locally. The field is a conduit, not a
  /// document — without this there is no local proof a keystroke left.
  final _echo = <_EchoChar>[];
  Timer? _echoTimer;

  ZrSession get _session => context.read<ZrSession>();

  @override
  void initState() {
    super.initState();
    // the field's border tracks focus, so the widget has to hear about it
    _focus.addListener(() {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _echoTimer?.cancel();
    _controller.dispose();
    _focus.dispose();
    super.dispose();
  }

  void _haptic() {
    if (context.read<ZrSettings>().haptics) HapticFeedback.selectionClick();
  }

  // ── typing ─────────────────────────────────────────────────────────────────

  void _onChanged(String value) {
    if (value == _previous) return;

    if (value.length > _previous.length && value.startsWith(_previous)) {
      final added = value.substring(_previous.length);
      if (added.contains('\n')) {
        for (final part in added.split('\n')) {
          if (part.isNotEmpty) _emitText(part);
          _pressKey('enter', echo: '↵');
        }
      } else {
        _emitText(added);
      }
    } else if (value.length < _previous.length && _previous.startsWith(value)) {
      for (var i = 0; i < _previous.length - value.length; i++) {
        _pressKey('backspace', echo: '⌫', control: true);
      }
    } else if (value.isNotEmpty) {
      // selection replace / autocorrect rewrite — send what is there now
      _emitText(value);
    }

    _previous = value;
    // the field is not a document: reset it before it grows into one, without
    // emitting the deletion we would otherwise diff.
    if (value.length > 120) {
      _previous = '';
      _controller.value = TextEditingValue.empty;
    }
  }

  void _emitText(String text) {
    if (_mods.isNotEmpty && text.length == 1) {
      // a modifier is armed → this is a chord, not text
      _pressKey(text);
      return;
    }
    _session.type(text);
    _pushEcho(text);
    _haptic();
  }

  void _pressKey(String name, {String? echo, bool control = false}) {
    final mods = _mods.toList();
    _session.key(name, mods: mods);
    if (mods.isNotEmpty) setState(_mods.clear);
    _pushEcho(echo ?? _glyphFor(name), control: control || echo != null);
    _haptic();
  }

  static String _glyphFor(String key) => switch (key) {
        'space' => '␣',
        'enter' => '↵',
        'tab' => '⇥',
        'escape' => 'esc',
        'up' => '↑',
        'down' => '↓',
        'left' => '←',
        'right' => '→',
        _ => key,
      };

  void _pushEcho(String text, {bool control = false}) {
    setState(() {
      for (final char in text.characters) {
        _echo.add(_EchoChar(char == ' ' ? '␣' : char, control));
      }
      if (_echo.length > 18) _echo.removeRange(0, _echo.length - 18);
    });
    _echoTimer?.cancel();
    _echoTimer = Timer(const Duration(milliseconds: 1700), () {
      if (mounted) setState(_echo.clear);
    });
  }

  // ── build ──────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return ListView(
      padding: const EdgeInsets.fromLTRB(Z.s3, 0, Z.s3, Z.s4),
      children: [
        _typer(l),
        const SizedBox(height: Z.s3),
        _modRow(l),
        const SizedBox(height: Z.s3),
        _specialKeys(l),
        const SizedBox(height: Z.s3),
        _dpadAndShortcuts(l),
        const SizedBox(height: Z.s4),
        const _ClipboardCard(),
      ],
    );
  }

  Widget _typer(L10n l) => Container(
        decoration: BoxDecoration(
          color: Z.surface1,
          borderRadius: BorderRadius.circular(Z.rLg),
          border: Border.all(color: _focus.hasFocus ? Z.oliveMid : Z.line),
        ),
        padding: const EdgeInsets.all(Z.s3),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            TextField(
              controller: _controller,
              focusNode: _focus,
              onChanged: _onChanged,
              autofocus: true,
              maxLines: 2,
              minLines: 1,
              autocorrect: false,
              enableSuggestions: false,
              keyboardType: TextInputType.multiline,
              style: Z.body,
              decoration: InputDecoration(
                isDense: true,
                border: InputBorder.none,
                hintText: l.t('keys_type_ph'),
                hintStyle: Z.bodySoft.copyWith(color: Z.inkFaint, fontSize: 15),
              ),
            ),
            AnimatedSize(
              duration: Z.fast,
              curve: Z.ease,
              child: _echo.isEmpty
                  ? const SizedBox(width: double.infinity, height: 0)
                  : Padding(
                      padding: const EdgeInsets.only(top: Z.s2),
                      child: Wrap(
                        spacing: 4,
                        children: [
                          for (final char in _echo)
                            Text(
                              char.text,
                              style: Z.mono.copyWith(
                                fontSize: 19,
                                color: char.control ? Z.grapeSoft : Z.oliveSoft,
                              ),
                            ),
                        ],
                      ),
                    ),
            ),
          ],
        ),
      );

  Widget _modRow(L10n l) {
    const mods = [
      ('ctrl', 'mod_ctrl'),
      ('alt', 'mod_alt'),
      ('shift', 'mod_shift'),
      ('meta', 'mod_meta'),
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            for (final (id, key) in mods) ...[
              Expanded(
                child: _Key(
                  label: l.t(key),
                  active: _mods.contains(id),
                  onTap: () {
                    setState(() =>
                        _mods.contains(id) ? _mods.remove(id) : _mods.add(id));
                    _haptic();
                  },
                ),
              ),
              if (id != 'meta') const SizedBox(width: Z.s2),
            ],
          ],
        ),
        const SizedBox(height: Z.s2),
        Text(l.t('mods_hint'),
            style: Z.mono.copyWith(fontSize: 11.5, color: Z.inkFaint)),
      ],
    );
  }

  Widget _specialKeys(L10n l) {
    final keys = <(String, String)>[
      (l.t('key_esc'), 'escape'),
      (l.t('key_tab'), 'tab'),
      (l.t('key_backspace'), 'backspace'),
      (l.t('key_enter'), 'enter'),
      (l.t('key_home'), 'home'),
      (l.t('key_end'), 'end'),
      (l.t('key_pgup'), 'pageup'),
      (l.t('key_pgdn'), 'pagedown'),
      (l.t('key_delete'), 'delete'),
      (l.t('key_space'), 'space'),
      ('F5', 'f5'),
      ('F11', 'f11'),
    ];
    return GridView.count(
      crossAxisCount: 4,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      mainAxisSpacing: Z.s2,
      crossAxisSpacing: Z.s2,
      childAspectRatio: 2.1,
      children: [
        for (final (label, key) in keys)
          _Key(label: label, onTap: () => _pressKey(key)),
      ],
    );
  }

  /// Side by side when there is room, stacked when there isn't. Beside the
  /// d-pad a phone in portrait leaves the shortcut chips ~200pt, which is not
  /// enough for "Sélectionner tout" — and a chip whose label is cut in half is
  /// worse than a chip on its own line.
  Widget _dpadAndShortcuts(L10n l) => LayoutBuilder(
        builder: (context, constraints) {
          final shortcuts = Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(l.t('shortcuts').toUpperCase(), style: Z.eyebrow),
              const SizedBox(height: Z.s2),
              Wrap(
                spacing: Z.s2,
                runSpacing: Z.s2,
                children: [
                  _shortcut(l.t('sc_copy'), 'c', ['ctrl']),
                  _shortcut(l.t('sc_paste'), 'v', ['ctrl']),
                  _shortcut(l.t('sc_cut'), 'x', ['ctrl']),
                  _shortcut(l.t('sc_undo'), 'z', ['ctrl']),
                  _shortcut(l.t('sc_selectall'), 'a', ['ctrl']),
                  _shortcut(l.t('sc_switch'), 'tab', ['alt']),
                  _shortcut(l.t('sc_close'), 'f4', ['alt']),
                  _shortcut(l.t('sc_lock'), 'l', ['meta']),
                ],
              ),
            ],
          );
          final dpad = _Dpad(onPress: _pressKey);
          if (constraints.maxWidth < 520) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Center(child: dpad),
                const SizedBox(height: Z.s3),
                shortcuts,
              ],
            );
          }
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              dpad,
              const SizedBox(width: Z.s3),
              Expanded(child: shortcuts),
            ],
          );
        },
      );

  Widget _shortcut(String label, String key, List<String> mods) => ZChip(
        label: label,
        selected: false,
        onTap: () {
          _session.key(key, mods: mods);
          _pushEcho(label, control: true);
          _haptic();
        },
      );
}

class _EchoChar {
  const _EchoChar(this.text, this.control);
  final String text;
  final bool control;
}

class _Key extends StatefulWidget {
  const _Key({required this.label, required this.onTap, this.active = false});

  final String label;
  final VoidCallback onTap;
  final bool active;

  @override
  State<_Key> createState() => _KeyState();
}

class _KeyState extends State<_Key> {
  bool _down = false;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTapDown: (_) => setState(() => _down = true),
        onTapUp: (_) => setState(() => _down = false),
        onTapCancel: () => setState(() => _down = false),
        onTap: widget.onTap,
        behavior: HitTestBehavior.opaque,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          height: Z.tap,
          alignment: Alignment.center,
          padding: const EdgeInsets.symmetric(horizontal: Z.s2),
          decoration: BoxDecoration(
            color: widget.active ? Z.olive : (_down ? Z.surface3 : Z.surface2),
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: widget.active ? Z.oliveMid : Z.line),
          ),
          child: Text(
            widget.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: Z.label.copyWith(
              fontSize: 15,
              color: widget.active ? Z.oliveBright : Z.ink,
            ),
          ),
        ),
      );
}

/// Inverted-T arrow cluster — the shape every keyboard already has, so the
/// thumb finds it without looking.
class _Dpad extends StatelessWidget {
  const _Dpad({required this.onPress});
  final void Function(String key) onPress;

  @override
  Widget build(BuildContext context) {
    Widget arrow(IconData icon, String key) => _DpadKey(icon: icon, onTap: () => onPress(key));
    // three 46pt keys + two 8pt gaps = 154; the box has to allow for that or
    // the row overflows by exactly 2 pixels.
    return SizedBox(
      width: 46 * 3 + Z.s2 * 2,
      child: Column(
        children: [
          arrow(Icons.keyboard_arrow_up_rounded, 'up'),
          const SizedBox(height: Z.s2),
          Row(
            children: [
              arrow(Icons.keyboard_arrow_left_rounded, 'left'),
              const SizedBox(width: Z.s2),
              arrow(Icons.keyboard_arrow_down_rounded, 'down'),
              const SizedBox(width: Z.s2),
              arrow(Icons.keyboard_arrow_right_rounded, 'right'),
            ],
          ),
        ],
      ),
    );
  }
}

class _DpadKey extends StatelessWidget {
  const _DpadKey({required this.icon, required this.onTap});
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: Container(
          width: 46,
          height: 46,
          decoration: BoxDecoration(
            color: Z.surface2,
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: Z.line),
          ),
          child: Icon(icon, color: Z.ink, size: 24),
        ),
      );
}

/// The shared clipboard — new on the phone, and a good example of the rule:
/// when the computer can't do it, the card stays and says why.
class _ClipboardCard extends StatelessWidget {
  const _ClipboardCard();

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    final state = session.caps[ZrCap.clipboard];

    if (state != ZrCapState.ready) {
      return ZCapabilityNotice(cap: ZrCap.clipboard, state: state);
    }

    final text = session.hostClipboard;
    return ZCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.content_paste_rounded,
                  size: 18, color: Z.oliveSoft),
              const SizedBox(width: Z.s2),
              Expanded(
                child: Text(l.t('clipboard'),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Z.body.copyWith(fontWeight: FontWeight.w600)),
              ),
            ],
          ),
          const SizedBox(height: Z.s3),
          Text(l.t('clip_from_host'), style: Z.label),
          const SizedBox(height: Z.s1),
          Container(
            width: double.infinity,
            constraints: const BoxConstraints(maxHeight: 96),
            padding: const EdgeInsets.all(Z.s3),
            decoration: BoxDecoration(
              color: Z.surface2,
              borderRadius: BorderRadius.circular(Z.rMd),
              border: Border.all(color: Z.line),
            ),
            child: SingleChildScrollView(
              child: Text(
                text?.isNotEmpty == true ? text! : l.t('clip_empty'),
                style: Z.mono.copyWith(
                  color: text?.isNotEmpty == true ? Z.ink : Z.inkFaint,
                ),
              ),
            ),
          ),
          const SizedBox(height: Z.s3),
          Row(
            children: [
              Expanded(
                child: ZButton(
                  label: l.t('clip_copy_here'),
                  kind: ZButtonKind.ghost,
                  size: ZButtonSize.sm,
                  icon: Icons.download_rounded,
                  expand: true,
                  onPressed: text?.isNotEmpty == true
                      ? () async {
                          await Clipboard.setData(ClipboardData(text: text!));
                          if (context.mounted) zToast(context, l.t('clip_copied'));
                        }
                      : null,
                ),
              ),
              const SizedBox(width: Z.s2),
              Expanded(
                child: ZButton(
                  label: l.t('clip_send'),
                  kind: ZButtonKind.ghost,
                  size: ZButtonSize.sm,
                  icon: Icons.upload_rounded,
                  expand: true,
                  onPressed: () async {
                    final data = await Clipboard.getData(Clipboard.kTextPlain);
                    final value = data?.text ?? '';
                    if (value.isEmpty) return;
                    session.pushClipboard(value);
                    if (context.mounted) zToast(context, l.t('clip_sent'));
                  },
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
