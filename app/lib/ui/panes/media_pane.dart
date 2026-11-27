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

/// Playback, volume and brightness — the things you reach for without looking
/// at the computer at all.
class MediaPane extends StatefulWidget {
  const MediaPane({super.key});

  @override
  State<MediaPane> createState() => _MediaPaneState();
}

class _MediaPaneState extends State<MediaPane> {
  /// -1 = every screen at once.
  int _brightTarget = -1;
  double? _brightValue;
  Timer? _brightThrottle;
  DateTime _brightSentAt = DateTime.fromMillisecondsSinceEpoch(0);

  ZrSession get _session => context.read<ZrSession>();

  @override
  void dispose() {
    _brightThrottle?.cancel();
    super.dispose();
  }

  void _haptic() {
    if (context.read<ZrSettings>().haptics) HapticFeedback.selectionClick();
  }

  void _media(String key) {
    _session.media(key);
    _haptic();
  }

  /// The agent shells out to an OS tool per value, so a dragged slider must not
  /// become a queue of stale writes: throttle to a trailing send, and the last
  /// position always lands.
  void _sendBrightness(double value) {
    setState(() => _brightValue = value);
    final now = DateTime.now();
    final since = now.difference(_brightSentAt).inMilliseconds;
    if (since < 120) {
      _brightThrottle?.cancel();
      _brightThrottle =
          Timer(Duration(milliseconds: 130 - since), () => _sendBrightness(value));
      return;
    }
    _brightSentAt = now;
    _brightThrottle?.cancel();
    _session.setBrightness(value.round(), screen: _brightTarget);
  }

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    return ListView(
      padding: const EdgeInsets.fromLTRB(Z.s3, 0, Z.s3, Z.s5),
      children: [
        _transport(l),
        const SizedBox(height: Z.s4),
        _brightness(l, session),
      ],
    );
  }

  Widget _transport(L10n l) => ZCard(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l.t('media_transport').toUpperCase(), style: Z.eyebrow),
            const SizedBox(height: Z.s3),
            Row(
              children: [
                Expanded(
                  child: _MediaKey(
                    icon: Icons.volume_down_rounded,
                    label: l.t('vol_down'),
                    onTap: () => _media('voldown'),
                  ),
                ),
                const SizedBox(width: Z.s2),
                Expanded(
                  child: _MediaKey(
                    icon: Icons.volume_off_rounded,
                    label: l.t('mute'),
                    onTap: () => _media('mute'),
                  ),
                ),
                const SizedBox(width: Z.s2),
                Expanded(
                  child: _MediaKey(
                    icon: Icons.volume_up_rounded,
                    label: l.t('vol_up'),
                    onTap: () => _media('volup'),
                  ),
                ),
              ],
            ),
            const SizedBox(height: Z.s2),
            Row(
              children: [
                Expanded(
                  child: _MediaKey(
                    icon: Icons.skip_previous_rounded,
                    label: l.t('previous'),
                    onTap: () => _media('prev'),
                  ),
                ),
                const SizedBox(width: Z.s2),
                Expanded(
                  flex: 2,
                  child: _MediaKey(
                    icon: Icons.play_arrow_rounded,
                    label: l.t('play_pause'),
                    primary: true,
                    onTap: () => _media('playpause'),
                  ),
                ),
                const SizedBox(width: Z.s2),
                Expanded(
                  child: _MediaKey(
                    icon: Icons.skip_next_rounded,
                    label: l.t('next'),
                    onTap: () => _media('next'),
                  ),
                ),
              ],
            ),
          ],
        ),
      );

  Widget _brightness(L10n l, ZrSession session) {
    final state = session.caps[ZrCap.brightness];
    if (state != ZrCapState.ready) {
      return ZCapabilityNotice(cap: ZrCap.brightness, state: state);
    }

    final perScreen = session.caps[ZrCap.brightnessPerScreen];
    final method = session.caps[ZrCap.brightnessMethod];
    final screens = session.caps.brightScreens;
    final value = _brightValue ??
        (session.caps.brightness >= 0 ? session.caps.brightness.toDouble() : 70);
    final activeBackend = session.caps.brightBackends
        .where((b) => b.id == session.caps.activeBackend)
        .firstOrNull;

    return ZCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(l.t('brightness').toUpperCase(), style: Z.eyebrow),
              ),
              Text('${value.round()}%', style: Z.mono.copyWith(fontSize: 15)),
            ],
          ),
          if (method == ZrCapState.ready) ...[
            const SizedBox(height: Z.s3),
            Text(l.t('bright_method'), style: Z.label),
            const SizedBox(height: Z.s2),
            Wrap(
              spacing: Z.s2,
              runSpacing: Z.s2,
              children: [
                for (final backend in session.caps.brightBackends)
                  ZChip(
                    label: backend.label.isNotEmpty ? backend.label : backend.id,
                    selected: backend.id == session.caps.activeBackend,
                    onTap: () {
                      session.setBrightnessBackend(backend.id);
                      setState(() => _brightValue = null);
                      _haptic();
                    },
                  ),
              ],
            ),
            if (activeBackend?.software == true) ...[
              const SizedBox(height: Z.s2),
              Text(l.t('bright_software_note'),
                  style: Z.bodySoft.copyWith(fontSize: 14, color: Z.warning)),
            ],
          ],
          if (perScreen == ZrCapState.ready) ...[
            const SizedBox(height: Z.s3),
            Wrap(
              spacing: Z.s2,
              runSpacing: Z.s2,
              children: [
                ZChip(
                  label: l.t('bright_all'),
                  selected: _brightTarget == -1,
                  onTap: () => setState(() {
                    _brightTarget = -1;
                    _brightValue = screens.isNotEmpty && screens.first.level >= 0
                        ? screens.first.level.toDouble()
                        : _brightValue;
                  }),
                ),
                for (var i = 0; i < screens.length; i++)
                  ZChip(
                    label: screens[i].name.isNotEmpty
                        ? screens[i].name
                        : '${l.t('bright_screen')} ${i + 1}',
                    selected: _brightTarget == i,
                    onTap: () => setState(() {
                      _brightTarget = i;
                      if (screens[i].level >= 0) {
                        _brightValue = screens[i].level.toDouble();
                      }
                    }),
                  ),
              ],
            ),
          ],
          const SizedBox(height: Z.s2),
          Row(
            children: [
              const Icon(Icons.brightness_low_rounded,
                  size: 18, color: Z.inkMuted),
              Expanded(
                child: ZSlider(
                  value: value,
                  min: 5,
                  max: 100,
                  onChanged: _sendBrightness,
                  onChangeEnd: (v) {
                    _sendBrightness(v);
                    _haptic();
                  },
                ),
              ),
              const Icon(Icons.brightness_high_rounded,
                  size: 18, color: Z.inkMuted),
            ],
          ),
          if (perScreen != ZrCapState.ready) ...[
            const SizedBox(height: Z.s3),
            ZCapabilityNotice(
              cap: ZrCap.brightnessPerScreen,
              state: perScreen,
              compact: true,
            ),
          ],
        ],
      ),
    );
  }
}

class _MediaKey extends StatefulWidget {
  const _MediaKey({
    required this.icon,
    required this.label,
    required this.onTap,
    this.primary = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool primary;

  @override
  State<_MediaKey> createState() => _MediaKeyState();
}

class _MediaKeyState extends State<_MediaKey> {
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
          height: 76,
          decoration: BoxDecoration(
            color: widget.primary
                ? (_down ? Z.oliveMid : Z.olive)
                : (_down ? Z.surface3 : Z.surface2),
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: widget.primary ? Z.oliveMid : Z.line),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(widget.icon,
                  size: widget.primary ? 32 : 26,
                  color: widget.primary ? Z.oliveBright : Z.ink),
              const SizedBox(height: 2),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 4),
                child: Text(
                  widget.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.center,
                  style: Z.label.copyWith(
                    fontSize: 12.5,
                    color: widget.primary ? Z.oliveBright : Z.inkSoft,
                  ),
                ),
              ),
            ],
          ),
        ),
      );
}
