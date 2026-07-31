import 'package:flutter/material.dart';

import '../core/caps.dart';
import '../core/i18n/i18n.dart';
import '../core/session.dart';
import 'settings_controls.dart';
import 'theme.dart';
import 'widgets.dart';

/// Quick settings, without leaving the computer you are driving.
///
/// It ends with a full capability report for this computer: everything it can
/// do and everything it can't, with the reason. That report is the answer to
/// "why is this button missing?" — asked once, here, instead of never.
Future<void> showSessionSettings(BuildContext context, ZrSession session) {
  final l = L10n.of(context);
  return showZSheet<void>(
    context,
    title: l.t('settings'),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const SettingsControls(inSession: true),
        ZSectionTitle(l.t('cap_why')),
        _CapabilityReport(session: session),
      ],
    ),
  );
}

class _CapabilityReport extends StatelessWidget {
  const _CapabilityReport({required this.session});
  final ZrSession session;

  static const _hostCaps = [
    ZrCap.screen,
    ZrCap.multiMonitor,
    ZrCap.brightness,
    ZrCap.brightnessPerScreen,
    ZrCap.brightnessMethod,
    ZrCap.clipboard,
    ZrCap.keyHold,
  ];

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final cap in _hostCaps) ...[
          _CapRow(cap: cap, state: session.caps[cap], l: l),
          const SizedBox(height: Z.s2),
        ],
      ],
    );
  }
}

class _CapRow extends StatelessWidget {
  const _CapRow({required this.cap, required this.state, required this.l});

  final ZrCap cap;
  final ZrCapState state;
  final L10n l;

  @override
  Widget build(BuildContext context) {
    final ready = state == ZrCapState.ready;
    final reason = l.capReason(cap, state);
    return ZCard(
      dimmed: !ready,
      padding: const EdgeInsets.all(Z.s3),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                ready ? Icons.check_circle_outline_rounded : Icons.block_rounded,
                size: 18,
                color: ready ? Z.oliveSoft : Z.inkMuted,
              ),
              const SizedBox(width: Z.s2),
              Expanded(
                child: Text(l.capTitle(cap),
                    style: Z.body.copyWith(fontWeight: FontWeight.w600)),
              ),
            ],
          ),
          if (!ready && reason != null) ...[
            const SizedBox(height: Z.s2),
            Text(reason, style: Z.bodySoft.copyWith(fontSize: 14.5)),
          ],
        ],
      ),
    );
  }
}
