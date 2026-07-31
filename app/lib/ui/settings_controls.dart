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
  ZrCapState _background = ZrCapState.ready;

  @override
  void initState() {
    super.initState();
    _refreshPermissions();
  }

  Future<void> _refreshPermissions() async {
    final granted = await ZrNative.instance.hasNotificationPermission();
    final restricted = await ZrNative.instance.isBackgroundRestricted();
    if (mounted) {
      setState(() {
        _notifications =
            granted ? ZrCapState.ready : ZrCapState.needsPermission;
        _background =
            restricted ? ZrCapState.needsPermission : ZrCapState.ready;
      });
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
        if (_background != ZrCapState.ready) ...[
          const SizedBox(height: Z.s2),
          ZCapabilityNotice(
            cap: ZrCap.backgroundSession,
            state: _background,
            actionLabel: l.t('open_app_settings'),
            action: () async {
              await ZrNative.instance.openAppSettings();
              await _refreshPermissions();
            },
          ),
        ],
        // Unrestricting the lock screen is a security decision, so it is
        // confirmed once and then kept visible while it is on.
        ZRow(
          title: l.t('lock_full'),
          note: l.t('lock_full_note'),
          dimmed: !settings.lockScreenControls,
          trailing: ZToggle(
            value: settings.lockFullControl,
            onChanged: settings.lockScreenControls
                ? (v) async {
                    if (!v) {
                      settings.lockFullControl = false;
                      return;
                    }
                    final ok = await _confirmFullControl(context);
                    if (ok) settings.lockFullControl = true;
                  }
                : null,
          ),
        ),
        if (settings.lockFullControl && settings.lockScreenControls)
          Padding(
            padding: const EdgeInsets.only(bottom: Z.s2),
            child: ZCard(
              padding: const EdgeInsets.all(Z.s3),
              child: Row(
                children: [
                  const Icon(Icons.lock_open_rounded,
                      size: 18, color: Z.warning),
                  const SizedBox(width: Z.s2),
                  Expanded(
                    child: Text(l.t('lock_full_warning'),
                        style: Z.bodySoft
                            .copyWith(fontSize: 14.5, color: Z.warning)),
                  ),
                ],
              ),
            ),
          ),
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


/// One explicit confirmation before the whole remote becomes reachable without
/// a PIN. Destructive styling on purpose: this is not a preference, it is a
/// decision about who can drive the computer.
Future<bool> _confirmFullControl(BuildContext context) async {
  final l = L10n.of(context);
  final ok = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      backgroundColor: Z.surface1,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(Z.rLg),
        side: const BorderSide(color: Z.lineStrong),
      ),
      title: Text(l.t('lock_full_confirm_title'), style: Z.title),
      content: Text(l.t('lock_full_confirm_body'), style: Z.bodySoft),
      actions: [
        ZButton(
          label: l.t('cancel'),
          kind: ZButtonKind.secondary,
          size: ZButtonSize.sm,
          onPressed: () => Navigator.of(context).pop(false),
        ),
        ZButton(
          label: l.t('lock_full_confirm_ok'),
          kind: ZButtonKind.destructive,
          size: ZButtonSize.sm,
          onPressed: () => Navigator.of(context).pop(true),
        ),
      ],
    ),
  );
  return ok ?? false;
}
