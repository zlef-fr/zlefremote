import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/caps.dart';
import '../core/i18n/i18n.dart';
import '../core/native.dart';
import '../core/settings.dart';
import 'theme.dart';
import 'widgets.dart';

/// The preference controls, shared by the standalone Settings screen and the
/// in-session sheet — one definition, so the two can never drift.
class SettingsControls extends StatefulWidget {
  const SettingsControls({super.key, this.inSession = false});

  /// Inside a live session the phone-side surfaces are actionable right now,
  /// so their permission state is checked and offered here.
  final bool inSession;

  @override
  State<SettingsControls> createState() => _SettingsControlsState();
}

class _SettingsControlsState extends State<SettingsControls> {
  ZrCapState _notifications = ZrCapState.ready;

  @override
  void initState() {
    super.initState();
    _refreshPermissions();
  }

  Future<void> _refreshPermissions() async {
    final granted = await ZrNative.instance.hasNotificationPermission();
    if (mounted) {
      setState(() => _notifications =
          granted ? ZrCapState.ready : ZrCapState.needsPermission);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final settings = context.watch<ZrSettings>();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ZSectionTitle(l.t('section_pointer')),
        ZRow(
          title: l.t('pointer_speed'),
          trailing: Text(settings.pointerSpeed.toStringAsFixed(1),
              style: Z.mono.copyWith(fontSize: 15)),
        ),
        ZSlider(
          value: settings.pointerSpeed,
          min: 0.6,
          max: 3.5,
          onChanged: (v) => settings.pointerSpeed = v,
        ),
        ZRow(
          title: l.t('scroll_speed'),
          trailing: Text(settings.scrollSpeed.toStringAsFixed(1),
              style: Z.mono.copyWith(fontSize: 15)),
        ),
        ZSlider(
          value: settings.scrollSpeed,
          min: 0.3,
          max: 3.0,
          onChanged: (v) => settings.scrollSpeed = v,
        ),
        ZRow(
          title: l.t('natural_scroll'),
          note: l.t('natural_scroll_note'),
          trailing: ZToggle(
            value: settings.naturalScroll,
            onChanged: (v) => settings.naturalScroll = v,
          ),
        ),
        ZSectionTitle(l.t('section_phone')),
        ZRow(
          title: l.t('haptics'),
          note: l.t('haptics_note'),
          trailing: ZToggle(
            value: settings.haptics,
            onChanged: (v) => settings.haptics = v,
          ),
        ),
        ZRow(
          title: l.t('keep_awake'),
          note: l.t('keep_awake_note'),
          trailing: ZToggle(
            value: settings.keepAwake,
            onChanged: (v) {
              settings.keepAwake = v;
              ZrNative.instance.setKeepAwake(v);
            },
          ),
        ),
        ZRow(
          title: l.t('lock_controls'),
          note: l.t('lock_controls_note'),
          dimmed: _notifications != ZrCapState.ready,
          trailing: ZToggle(
            value: settings.lockScreenControls &&
                _notifications == ZrCapState.ready,
            onChanged: _notifications == ZrCapState.ready
                ? (v) => settings.lockScreenControls = v
                : null,
          ),
        ),
        if (_notifications != ZrCapState.ready) ...[
          const SizedBox(height: Z.s2),
          ZCapabilityNotice(
            cap: ZrCap.lockScreenControls,
            state: _notifications,
            compact: true,
            actionLabel: l.t('grant_permission'),
            action: () async {
              await ZrNative.instance.requestNotificationPermission();
              await _refreshPermissions();
            },
          ),
        ],
        ZRow(
          title: l.t('volume_keys'),
          note: l.t('volume_keys_note'),
          trailing: ZToggle(
            value: settings.volumeKeys,
            onChanged: (v) {
              settings.volumeKeys = v;
              ZrNative.instance.setVolumeKeyCapture(v);
            },
          ),
        ),
      ],
    );
  }
}
