import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../core/caps.dart';
import '../core/conn.dart';
import '../core/devices.dart';
import '../core/i18n/i18n.dart';
import '../core/native.dart';
import '../core/session.dart';
import '../core/settings.dart';
import 'panes/keys_pane.dart';
import 'panes/media_pane.dart';
import 'panes/screen_pane.dart';
import 'lock_surface.dart';
import 'panes/touchpad_pane.dart';
import 'session_settings_sheet.dart';
import 'theme.dart';
import 'widgets.dart';

/// The connected surface: one computer, four ways to drive it.
///
/// The mode dock sits at the bottom, under the thumb, and the pane above it
/// fills everything else — a phone remote is held one-handed, so nothing that
/// matters lives at the top of the screen.
class ControlScreen extends StatefulWidget {
  const ControlScreen({super.key, required this.device, this.session});

  final ZrDevice device;

  /// Test seam: supply a session and the screen drives that one instead of
  /// dialling a computer. Nothing in the app passes it.
  @visibleForTesting
  final ZrSession? session;

  @override
  State<ControlScreen> createState() => _ControlScreenState();
}

enum ZrMode { pad, screen, keys, media }

class _ControlScreenState extends State<ControlScreen> {
  late final ZrSession _session;
  late final ZrSettings _settings;
  StreamSubscription<ZrNativeEvent>? _nativeSub;
  ZrMode _mode = ZrMode.pad;
  bool _agentBannerDismissed = false;
  bool _backgroundBannerDismissed = false;
  bool _backgroundRestricted = false;
  bool _locked = false;

  @override
  void initState() {
    super.initState();
    _settings = context.read<ZrSettings>();
    final injected = widget.session;
    _session = injected ??
        ZrSession(
          device: widget.device,
          onDeviceLearned: (updated) =>
              context.read<ZrDeviceStore>().upsert(updated),
        );
    _session.addListener(_onSessionChanged);
    _session.quality = _settings.viewQuality;
    if (injected == null) unawaited(_session.start());
    _startNativeSurfaces();
  }

  /// The foreground service is what keeps this session alive when the screen
  /// goes off or the user switches apps — the browser used to drop the socket.
  /// It also carries the media notification.
  Future<void> _startNativeSurfaces() async {
    final native = ZrNative.instance;
    if (_settings.lockScreenControls) {
      await native.startSession(_hostLabel);
    }
    await native.setKeepAwake(_settings.keepAwake);
    await native.setVolumeKeyCapture(_settings.volumeKeys);
    // the platform channel is absent under test and on a plain Flutter host;
    // an unhandled stream error there would fail the whole screen.
    _nativeSub = native.events.listen(_onNativeEvent, onError: (_) {});
    // A background-restricted app keeps its foreground service but loses its
    // socket the moment the screen goes off. Say so up front — a session that
    // dies in your pocket is the one failure the user can't diagnose.
    final restricted = await native.isBackgroundRestricted();
    if (mounted && restricted) setState(() => _backgroundRestricted = true);
  }

  void _onNativeEvent(ZrNativeEvent event) {
    switch (event.kind) {
      case ZrNativeEventKind.media:
        if (event.value != null) _session.media(event.value!);
      case ZrNativeEventKind.volumeUp:
        _session.media('volup');
      case ZrNativeEventKind.volumeDown:
        _session.media('voldown');
      case ZrNativeEventKind.stopRequested:
        if (mounted) Navigator.of(context).maybePop();
      case ZrNativeEventKind.keyguard:
        // The activity shows over the keyguard, so "locked" is not "hidden" —
        // it decides which surface is safe to put in front of whoever picked
        // the phone up.
        final locked = event.value == 'locked';
        if (mounted && locked != _locked) setState(() => _locked = locked);
    }
  }

  String get _hostLabel => _session.hostName.isNotEmpty
      ? _session.hostName
      : (widget.device.name.isNotEmpty ? widget.device.name : 'ZlefRemote');

  void _onSessionChanged() {
    if (!mounted) return;
    ZrNative.instance.updateSession(
      host: _hostLabel,
      connected: _session.isPaired,
    );
  }

  @override
  void dispose() {
    _nativeSub?.cancel();
    ZrNative.instance
      ..stopSession()
      ..setKeepAwake(false)
      ..setVolumeKeyCapture(false);
    _session.removeListener(_onSessionChanged);
    // an injected session belongs to whoever made it
    if (widget.session == null) _session.dispose();
    super.dispose();
  }

  void _setMode(ZrMode mode) {
    if (mode == _mode) return;
    HapticFeedback.selectionClick();
    setState(() => _mode = mode);
    // the stream is expensive on both ends: run it only while it is on screen
    if (mode == ZrMode.screen) {
      if (_session.caps.ready(ZrCap.screen)) _session.startView();
    } else {
      _session.stopView();
    }
  }

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider.value(
      value: _session,
      child: Scaffold(
        backgroundColor: Z.bg,
        resizeToAvoidBottomInset: true,
        // Over the keyguard the user gets what they chose: the playback-only
        // surface by default, or the whole remote once they have enabled it and
        // accepted what that means.
        body: _locked &&
                _settings.lockScreenControls &&
                !_settings.lockFullControl
            ? const LockSurface()
            : SafeArea(
          child: Column(
            children: [
              _TopBar(
                device: widget.device,
                onSettings: () => showSessionSettings(context, _session),
              ),
              Expanded(
                child: Stack(
                  children: [
                    _Panes(
                      mode: _mode,
                      bannerDismissed: _agentBannerDismissed,
                      onDismissBanner: () =>
                          setState(() => _agentBannerDismissed = true),
                      backgroundRestricted:
                          _backgroundRestricted && !_backgroundBannerDismissed,
                      onDismissBackground: () =>
                          setState(() => _backgroundBannerDismissed = true),
                    ),
                    const _ConnectionOverlay(),
                  ],
                ),
              ),
              _ModeDock(mode: _mode, onChange: _setMode),
            ],
          ),
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({required this.device, required this.onSettings});

  final ZrDevice device;
  final VoidCallback onSettings;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    final (label, tone) = switch (session.state) {
      ZrConnState.paired => (l.t('paired'), ZTone.good),
      ZrConnState.connecting => (l.t('connecting'), ZTone.working),
      ZrConnState.linking => (l.t('linking'), ZTone.working),
      ZrConnState.reconnecting => (l.t('reconnecting'), ZTone.working),
      _ => (l.t('disconnected'), ZTone.bad),
    };
    final name = session.hostName.isNotEmpty
        ? session.hostName
        : (device.name.isNotEmpty ? device.name : l.t('unknown_computer'));

    return Padding(
      padding: const EdgeInsets.fromLTRB(Z.s1, Z.s1, Z.s1, Z.s2),
      child: Row(
        children: [
          IconButton(
            onPressed: () => Navigator.of(context).maybePop(),
            tooltip: l.t('back_to_devices'),
            icon: const Icon(Icons.arrow_back_rounded, color: Z.inkSoft),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(name,
                    style: Z.body.copyWith(fontWeight: FontWeight.w600),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis),
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                  children: [
                    ZStatusDot(tone),
                    const SizedBox(width: 6),
                    Text(label,
                        style: Z.mono.copyWith(fontSize: 12.5, color: Z.inkMuted)),
                    if (session.latency != null) ...[
                      Text(' · ${session.latency!.inMilliseconds} ms',
                          style: Z.mono
                              .copyWith(fontSize: 12.5, color: Z.inkFaint)),
                    ],
                    const SizedBox(width: 8),
                    const Icon(Icons.lock_rounded, size: 12, color: Z.oliveSoft),
                    const SizedBox(width: 3),
                    Text(l.t('e2ee'),
                        style: Z.mono.copyWith(fontSize: 11.5, color: Z.oliveSoft)),
                  ],
                ),
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: onSettings,
            tooltip: l.t('settings'),
            icon: const Icon(Icons.tune_rounded, color: Z.inkSoft),
          ),
        ],
      ),
    );
  }
}

class _Panes extends StatelessWidget {
  const _Panes({
    required this.mode,
    required this.bannerDismissed,
    required this.onDismissBanner,
    required this.backgroundRestricted,
    required this.onDismissBackground,
  });

  final ZrMode mode;
  final bool bannerDismissed;
  final VoidCallback onDismissBanner;
  final bool backgroundRestricted;
  final VoidCallback onDismissBackground;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    final showBanner =
        session.isPaired && session.caps.agentLooksOld && !bannerDismissed;

    return Column(
      children: [
        AnimatedSize(
          duration: Z.normal,
          curve: Z.ease,
          child: backgroundRestricted
              ? Padding(
                  padding: const EdgeInsets.fromLTRB(Z.s4, 0, Z.s4, Z.s2),
                  child: ZCard(
                    padding: const EdgeInsets.all(Z.s3),
                    child: Row(
                      children: [
                        const Icon(Icons.battery_alert_rounded,
                            size: 18, color: Z.warning),
                        const SizedBox(width: Z.s2),
                        Expanded(
                          child: Text(l.t('background_restricted_banner'),
                              style: Z.bodySoft.copyWith(fontSize: 14.5)),
                        ),
                        const SizedBox(width: Z.s2),
                        ZButton(
                          label: l.t('open_app_settings'),
                          kind: ZButtonKind.ghost,
                          size: ZButtonSize.sm,
                          onPressed: () async {
                            await ZrNative.instance.openAppSettings();
                            onDismissBackground();
                          },
                        ),
                      ],
                    ),
                  ),
                )
              : const SizedBox(width: double.infinity),
        ),
        AnimatedSize(
          duration: Z.normal,
          curve: Z.ease,
          child: showBanner
              ? Padding(
                  padding: const EdgeInsets.fromLTRB(Z.s4, 0, Z.s4, Z.s2),
                  child: ZCard(
                    padding: const EdgeInsets.all(Z.s3),
                    child: Row(
                      children: [
                        const Icon(Icons.upgrade_rounded,
                            size: 18, color: Z.warning),
                        const SizedBox(width: Z.s2),
                        Expanded(
                          child: Text(l.t('agent_old_banner'),
                              style: Z.bodySoft.copyWith(fontSize: 14.5)),
                        ),
                        const SizedBox(width: Z.s2),
                        ZButton(
                          label: l.t('agent_old_dismiss'),
                          kind: ZButtonKind.ghost,
                          size: ZButtonSize.sm,
                          onPressed: onDismissBanner,
                        ),
                      ],
                    ),
                  ),
                )
              : const SizedBox(width: double.infinity),
        ),
        Expanded(
          child: IndexedStack(
            index: mode.index,
            sizing: StackFit.expand,
            children: const [
              TouchpadPane(),
              ScreenPane(),
              KeysPane(),
              MediaPane(),
            ],
          ),
        ),
      ],
    );
  }
}

/// The mode dock. Four targets, each at least 48pt, labels always visible —
/// icon-only docks make people learn a legend.
class _ModeDock extends StatelessWidget {
  const _ModeDock({required this.mode, required this.onChange});

  final ZrMode mode;
  final ValueChanged<ZrMode> onChange;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    const items = [
      (ZrMode.pad, Icons.touch_app_rounded, 'tab_pad'),
      (ZrMode.screen, Icons.desktop_windows_rounded, 'tab_screen'),
      (ZrMode.keys, Icons.keyboard_rounded, 'tab_keys'),
      (ZrMode.media, Icons.play_circle_outline_rounded, 'tab_media'),
    ];
    return Container(
      decoration: const BoxDecoration(
        color: Z.surface1,
        border: Border(top: BorderSide(color: Z.line)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: Z.s2, vertical: Z.s2),
      child: Row(
        children: [
          for (final (value, icon, key) in items)
            Expanded(
              child: _DockButton(
                icon: icon,
                label: l.t(key),
                selected: mode == value,
                onTap: () => onChange(value),
              ),
            ),
        ],
      ),
    );
  }
}

class _DockButton extends StatelessWidget {
  const _DockButton({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          height: 56,
          margin: const EdgeInsets.symmetric(horizontal: 3),
          decoration: BoxDecoration(
            color: selected ? Z.olive : Colors.transparent,
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: selected ? Z.oliveMid : Colors.transparent),
          ),
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon,
                  size: 21, color: selected ? Z.oliveBright : Z.inkMuted),
              const SizedBox(height: 2),
              // one line, always: a dock cell is ~88pt wide and a wrapped
              // label pushes the icon out of a fixed-height button.
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: Z.label.copyWith(
                  fontSize: 12,
                  color: selected ? Z.oliveBright : Z.inkMuted,
                ),
              ),
            ],
          ),
        ),
      );
}

/// Covers the panes while there is nothing to drive. Explains the state and
/// offers the one action that helps — never a bare spinner with no way out.
class _ConnectionOverlay extends StatelessWidget {
  const _ConnectionOverlay();

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final session = context.watch<ZrSession>();
    if (session.isPaired) return const SizedBox.shrink();

    final working = session.state == ZrConnState.connecting ||
        session.state == ZrConnState.linking ||
        session.state == ZrConnState.reconnecting;

    final (title, body) = switch (session.error) {
      ZrConnError.noSuchRoom => (l.t('err_room_title'), l.t('err_room_body')),
      ZrConnError.roomFull => (l.t('err_full_title'), l.t('err_full_body')),
      ZrConnError.lanUnreachable => (l.t('err_lan_title'), l.t('err_lan_body')),
      ZrConnError.unreachable => (
          l.t('err_connect_title'),
          l.t('err_connect_body')
        ),
      null => (
          switch (session.state) {
            ZrConnState.connecting => l.t('connecting'),
            ZrConnState.linking => l.t('linking'),
            ZrConnState.reconnecting => l.t('reconnecting'),
            ZrConnState.closed => l.t('closed_host'),
            _ => l.t('connecting'),
          },
          working ? '' : l.t('closed_host'),
        ),
    };

    // A small phone in landscape has ~200pt of height here once the dock and
    // top bar are taken: the copy has to be able to scroll, and the two actions
    // have to be able to stack, or the overlay clips the way out of the error.
    return Container(
      color: Z.bg.withValues(alpha: .96),
      alignment: Alignment.center,
      padding: const EdgeInsets.all(Z.s5),
      child: SingleChildScrollView(
        child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (working)
            const SizedBox(
              width: 30,
              height: 30,
              child: CircularProgressIndicator(
                strokeWidth: 2.4,
                valueColor: AlwaysStoppedAnimation(Z.oliveSoft),
              ),
            )
          else
            const Icon(Icons.power_off_rounded, size: 34, color: Z.danger),
          const SizedBox(height: Z.s4),
          Text(title, style: Z.title, textAlign: TextAlign.center),
          const SizedBox(height: Z.s2),
          if (body.isNotEmpty)
            Text(body, style: Z.bodySoft, textAlign: TextAlign.center),
          if (!working) ...[
            const SizedBox(height: Z.s5),
            Wrap(
              alignment: WrapAlignment.center,
              spacing: Z.s3,
              runSpacing: Z.s2,
              children: [
                ZButton(
                  label: l.t('back_to_devices'),
                  kind: ZButtonKind.ghost,
                  onPressed: () => Navigator.of(context).maybePop(),
                ),
                ZButton(
                  label: l.t('try_again'),
                  icon: Icons.refresh_rounded,
                  onPressed: session.retry,
                ),
              ],
            ),
          ],
        ],
        ),
      ),
    );
  }
}
