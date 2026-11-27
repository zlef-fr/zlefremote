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
        const SizedBox(height: Z.s2),
        ZSection(
          title: l.t('gesture_sheet'),
          child: const _GestureList(),
        ),
        ZSectionTitle(l.t('section_computer')),
        _AgentUpdate(session: session),
        const SizedBox(height: Z.s2),
        ZSection(
          title: l.t('cap_why'),
          child: _CapabilityReport(session: session),
        ),
      ],
    ),
  );
}

/// The gesture vocabulary, written down once.
///
/// It lives here rather than pinned over the pad: a strip of instructions on
/// the control surface is read once and then in the way forever, but "what can
/// I do with three fingers?" is a real question with one obvious place to look.
class _GestureList extends StatelessWidget {
  const _GestureList();

  static const _lines = [
    'gesture_two_scroll',
    'gesture_two_tap',
    'gesture_two_flick',
    'gesture_pinch',
    'gesture_three_side',
    'gesture_three_up',
    'gesture_three_down',
    'gesture_three_tap',
  ];

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final key in _lines)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Text(l.t(key), style: Z.bodySoft.copyWith(fontSize: 14)),
          ),
        const SizedBox(height: 2),
        Text(
          l.t('gesture_screen_three'),
          style: Z.bodySoft.copyWith(fontSize: 13, color: Z.inkFaint),
        ),
      ],
    );
  }
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
    ZrCap.gestures,
    ZrCap.agentUpdate,
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


/// Updating the computer's agent, from the computer's remote.
///
/// The phone already knows the agent is old — it is the thing comparing
/// capabilities and printing "update the agent" — so making the user walk over
/// and type a command was the wrong shape. An agent too old to accept the verb
/// still gets said out loud, with the one command that fixes it forever.
class _AgentUpdate extends StatelessWidget {
  const _AgentUpdate({required this.session});
  final ZrSession session;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    return ListenableBuilder(
      listenable: session,
      builder: (context, _) {
        final state = session.caps[ZrCap.agentUpdate];
        if (state != ZrCapState.ready) {
          return ZCapabilityNotice(cap: ZrCap.agentUpdate, state: state);
        }
        final status = switch (session.agentUpdate) {
          ZrAgentUpdate.running => l.t('agent_update_running'),
          ZrAgentUpdate.done => l.f('agent_update_done',
              {'v': session.agentUpdateVersion ?? ''}),
          ZrAgentUpdate.alreadyCurrent => l.t('agent_update_current'),
          ZrAgentUpdate.failed => l.f('agent_update_failed',
              {'reason': session.agentUpdateError ?? ''}),
          ZrAgentUpdate.idle => '',
        };
        return ZCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ZButton(
                label: l.t('agent_update'),
                icon: Icons.system_update_alt_rounded,
                kind: ZButtonKind.ghost,
                expand: true,
                loading: session.agentUpdate == ZrAgentUpdate.running,
                onPressed: session.agentUpdate == ZrAgentUpdate.running
                    ? null
                    : session.updateAgent,
              ),
              if (status.isNotEmpty) ...[
                const SizedBox(height: Z.s2),
                Text(
                  status,
                  style: Z.bodySoft.copyWith(
                    fontSize: 14.5,
                    color: session.agentUpdate == ZrAgentUpdate.failed
                        ? Z.danger
                        : Z.inkMuted,
                  ),
                ),
              ],
            ],
          ),
        );
      },
    );
  }
}
