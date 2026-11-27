import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../core/i18n/i18n.dart';
import '../core/session.dart';
import '../core/settings.dart';
import 'theme.dart';
import 'widgets.dart';

/// What the remote shows in front of a locked phone.
///
/// The activity is `showWhenLocked`, so picking the phone up puts this on
/// screen the way a navigation app does — no unlocking, no media card
/// borrowed from whatever music player happens to own the lock screen.
///
/// It is deliberately **not** the whole app. Anyone can pick up a locked phone,
/// so the surface carries the controls you reach for blind — playback, volume,
/// brightness-free, nothing destructive — and stops there. The trackpad, the
/// keyboard, the saved-computer list and settings all stay behind the lock,
/// because those are the ones that could be used against you.
class LockSurface extends StatelessWidget {
  const LockSurface({super.key});

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    final haptics = context.read<ZrSettings>().haptics;

    void send(String key) {
      session.media(key);
      if (haptics) HapticFeedback.mediumImpact();
    }

    return Container(
      color: Z.bg,
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(Z.s4),
          child: Column(
            children: [
              Row(
                children: [
                  ZStatusDot(session.isPaired ? ZTone.good : ZTone.working),
                  const SizedBox(width: Z.s2),
                  Expanded(
                    child: Text(
                      session.hostName.isNotEmpty
                          ? session.hostName
                          : l.t('unknown_computer'),
                      style: Z.body.copyWith(fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const Icon(Icons.lock_rounded, size: 15, color: Z.inkFaint),
                ],
              ),
              const Spacer(),
              Row(
                children: [
                  Expanded(
                    child: _LockKey(
                      icon: Icons.volume_down_rounded,
                      onTap: () => send('voldown'),
                    ),
                  ),
                  const SizedBox(width: Z.s3),
                  Expanded(
                    flex: 2,
                    child: _LockKey(
                      icon: Icons.play_arrow_rounded,
                      primary: true,
                      onTap: () => send('playpause'),
                    ),
                  ),
                  const SizedBox(width: Z.s3),
                  Expanded(
                    child: _LockKey(
                      icon: Icons.volume_up_rounded,
                      onTap: () => send('volup'),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: Z.s3),
              Row(
                children: [
                  Expanded(
                    child: _LockKey(
                      icon: Icons.skip_previous_rounded,
                      onTap: () => send('prev'),
                    ),
                  ),
                  const SizedBox(width: Z.s3),
                  Expanded(
                    child: _LockKey(
                      icon: Icons.volume_off_rounded,
                      onTap: () => send('mute'),
                    ),
                  ),
                  const SizedBox(width: Z.s3),
                  Expanded(
                    child: _LockKey(
                      icon: Icons.skip_next_rounded,
                      onTap: () => send('next'),
                    ),
                  ),
                ],
              ),
              const Spacer(),
              Text(
                l.t('lock_unlock_for_more'),
                textAlign: TextAlign.center,
                style: Z.mono.copyWith(fontSize: 12, color: Z.inkFaint),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Big, spaced, unlabelled: this is the surface you use without looking.
class _LockKey extends StatefulWidget {
  const _LockKey({required this.icon, required this.onTap, this.primary = false});

  final IconData icon;
  final VoidCallback onTap;
  final bool primary;

  @override
  State<_LockKey> createState() => _LockKeyState();
}

class _LockKeyState extends State<_LockKey> {
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
          height: 96,
          decoration: BoxDecoration(
            color: widget.primary
                ? (_down ? Z.oliveMid : Z.olive)
                : (_down ? Z.surface3 : Z.surface2),
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: widget.primary ? Z.oliveMid : Z.line),
          ),
          child: Icon(
            widget.icon,
            size: widget.primary ? 44 : 34,
            color: widget.primary ? Z.oliveBright : Z.ink,
          ),
        ),
      );
}
