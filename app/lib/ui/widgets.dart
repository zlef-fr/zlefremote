import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../core/caps.dart';
import '../core/i18n/i18n.dart';
import 'theme.dart';

enum ZButtonKind { primary, secondary, ghost, destructive }

enum ZButtonSize { sm, md, lg }

/// Flat, crisp, 1px border, one-shade hover/press shift. Never a gradient,
/// never a glow, never a scale.
class ZButton extends StatefulWidget {
  const ZButton({
    super.key,
    required this.label,
    this.onPressed,
    this.kind = ZButtonKind.primary,
    this.size = ZButtonSize.md,
    this.icon,
    this.loading = false,
    this.expand = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final ZButtonKind kind;
  final ZButtonSize size;
  final IconData? icon;
  final bool loading;
  final bool expand;

  @override
  State<ZButton> createState() => _ZButtonState();
}

class _ZButtonState extends State<ZButton> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    final enabled = widget.onPressed != null && !widget.loading;
    final (fill, ink, border) = switch (widget.kind) {
      ZButtonKind.primary => (
          _down ? Z.oliveMid : Z.olive,
          Z.oliveBright,
          Z.oliveMid,
        ),
      ZButtonKind.secondary => (
          _down ? Z.grapeMid : Z.grape,
          Z.ink,
          Z.grapeMid,
        ),
      ZButtonKind.ghost => (
          _down ? Z.surface3 : Colors.transparent,
          Z.ink,
          Z.lineStrong,
        ),
      ZButtonKind.destructive => (
          _down ? const Color(0xFF9E4456) : Z.danger,
          Z.bg,
          Z.danger,
        ),
    };
    final (height, pad, style) = switch (widget.size) {
      ZButtonSize.sm => (40.0, Z.s3, Z.label),
      ZButtonSize.md => (Z.tap, Z.s4, Z.body),
      ZButtonSize.lg => (56.0, Z.s5, Z.body.copyWith(fontSize: 18)),
    };

    return Opacity(
      opacity: enabled ? 1 : .45,
      child: GestureDetector(
        onTapDown: enabled ? (_) => setState(() => _down = true) : null,
        onTapUp: enabled ? (_) => setState(() => _down = false) : null,
        onTapCancel: enabled ? () => setState(() => _down = false) : null,
        onTap: enabled
            ? () {
                HapticFeedback.selectionClick();
                widget.onPressed!();
              }
            : null,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          height: height,
          width: widget.expand ? double.infinity : null,
          padding: EdgeInsets.symmetric(horizontal: pad),
          decoration: BoxDecoration(
            color: fill,
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: border, width: 1),
          ),
          child: Row(
            mainAxisSize: widget.expand ? MainAxisSize.max : MainAxisSize.min,
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (widget.loading)
                Padding(
                  padding: const EdgeInsets.only(right: Z.s2),
                  child: SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation(ink),
                    ),
                  ),
                )
              else if (widget.icon != null)
                Padding(
                  padding: const EdgeInsets.only(right: Z.s2),
                  child: Icon(widget.icon, size: 19, color: ink),
                ),
              Flexible(
                child: Text(
                  widget.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: style.copyWith(color: ink, fontWeight: FontWeight.w600),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 8px radius, hairline border, solid surface. The one container in the app.
class ZCard extends StatelessWidget {
  const ZCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(Z.s4),
    this.accent = false,
    this.onTap,
    this.dimmed = false,
  });

  final Widget child;
  final EdgeInsets padding;
  final bool accent;
  final VoidCallback? onTap;

  /// A card whose feature is unavailable: readable, obviously inert.
  final bool dimmed;

  @override
  Widget build(BuildContext context) {
    final card = AnimatedContainer(
      duration: Z.normal,
      curve: Z.ease,
      padding: padding,
      decoration: BoxDecoration(
        color: dimmed ? Z.surface1.withValues(alpha: .6) : Z.surface1,
        borderRadius: BorderRadius.circular(Z.rLg),
        border: Border.all(color: accent ? Z.oliveMid : Z.line),
      ),
      child: child,
    );
    if (onTap == null) return card;
    return GestureDetector(onTap: onTap, behavior: HitTestBehavior.opaque, child: card);
  }
}

/// Selectable chip — monitor picker, quality picker, brightness target.
class ZChip extends StatelessWidget {
  const ZChip({
    super.key,
    required this.label,
    required this.selected,
    this.onTap,
    this.icon,
    this.sub,
  });

  final String label;
  final bool selected;
  final VoidCallback? onTap;
  final IconData? icon;
  final String? sub;

  @override
  Widget build(BuildContext context) {
    final enabled = onTap != null;
    return Opacity(
      opacity: enabled ? 1 : .45,
      child: GestureDetector(
        onTap: enabled
            ? () {
                HapticFeedback.selectionClick();
                onTap!();
              }
            : null,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          constraints: const BoxConstraints(minHeight: 40),
          padding: const EdgeInsets.symmetric(horizontal: Z.s3, vertical: Z.s2),
          decoration: BoxDecoration(
            color: selected ? Z.olive : Z.surface2,
            borderRadius: BorderRadius.circular(Z.rSm),
            border: Border.all(color: selected ? Z.oliveMid : Z.line),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                Icon(icon,
                    size: 17, color: selected ? Z.oliveBright : Z.inkSoft),
                const SizedBox(width: Z.s2),
              ],
              // Flexible + ellipsis: a chip sits in a Wrap whose line width is
              // whatever the parent leaves it, so a long label ("Select all",
              // "Sélectionner tout") must degrade instead of overflowing.
              Flexible(
                child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Z.label.copyWith(
                      color: selected ? Z.oliveBright : Z.inkSoft,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  if (sub != null)
                    Text(sub!,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: Z.mono.copyWith(
                          fontSize: 11,
                          color: selected ? Z.oliveSoft : Z.inkFaint,
                        )),
                ],
              ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Lightly-rounded rectangle, never a pill.
class ZToggle extends StatelessWidget {
  const ZToggle({super.key, required this.value, this.onChanged});

  final bool value;
  final ValueChanged<bool>? onChanged;

  @override
  Widget build(BuildContext context) {
    final enabled = onChanged != null;
    return Opacity(
      opacity: enabled ? 1 : .45,
      child: GestureDetector(
        onTap: enabled
            ? () {
                HapticFeedback.selectionClick();
                onChanged!(!value);
              }
            : null,
        child: AnimatedContainer(
          duration: Z.fast,
          curve: Z.ease,
          width: 52,
          height: 30,
          padding: const EdgeInsets.all(3),
          decoration: BoxDecoration(
            color: value ? Z.olive : Z.surface3,
            borderRadius: BorderRadius.circular(Z.rMd),
            border: Border.all(color: value ? Z.oliveMid : Z.lineStrong),
          ),
          child: AnimatedAlign(
            duration: Z.fast,
            curve: Z.ease,
            alignment: value ? Alignment.centerRight : Alignment.centerLeft,
            child: Container(
              width: 22,
              height: 22,
              decoration: BoxDecoration(
                color: value ? Z.oliveBright : Z.inkMuted,
                borderRadius: BorderRadius.circular(Z.rSm),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Label + explanation + control, the settings row used everywhere.
/// A settings row. The explanation is real but it is not on screen by default:
/// a row of prose under every toggle turns a settings page into a document.
/// The ⓘ reveals it for the one setting the user is actually wondering about.
class ZRow extends StatefulWidget {
  const ZRow({
    super.key,
    required this.title,
    this.note,
    this.trailing,
    this.onTap,
    this.dimmed = false,
  });

  final String title;
  final String? note;
  final Widget? trailing;
  final VoidCallback? onTap;
  final bool dimmed;

  @override
  State<ZRow> createState() => _ZRowState();
}

class _ZRowState extends State<ZRow> {
  bool _open = false;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: widget.onTap,
      behavior: HitTestBehavior.opaque,
      child: Opacity(
        opacity: widget.dimmed ? .55 : 1,
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: Z.s3),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(child: Text(widget.title, style: Z.body)),
                  if (widget.note != null)
                    GestureDetector(
                      onTap: () => setState(() => _open = !_open),
                      behavior: HitTestBehavior.opaque,
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: Z.s2),
                        child: Icon(
                          _open
                              ? Icons.info_rounded
                              : Icons.info_outline_rounded,
                          size: 18,
                          color: _open ? Z.oliveSoft : Z.inkFaint,
                        ),
                      ),
                    ),
                  if (widget.trailing != null) ...[
                    const SizedBox(width: Z.s2),
                    widget.trailing!,
                  ],
                ],
              ),
              AnimatedSize(
                duration: Z.fast,
                curve: Z.ease,
                alignment: Alignment.topLeft,
                child: _open && widget.note != null
                    ? Padding(
                        padding: const EdgeInsets.only(top: Z.s2, right: Z.s6),
                        child: Text(widget.note!,
                            style: Z.bodySoft
                                .copyWith(fontSize: 14.5, color: Z.inkMuted)),
                      )
                    : const SizedBox(width: double.infinity),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class ZSectionTitle extends StatelessWidget {
  const ZSectionTitle(this.text, {super.key, this.trailing});
  final String text;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: Z.s3, top: Z.s5),
        child: Row(
          children: [
            Expanded(child: Text(text.toUpperCase(), style: Z.eyebrow)),
            ?trailing,
          ],
        ),
      );
}

/// A slider that stays on palette: flat track, square-ish thumb.
class ZSlider extends StatelessWidget {
  const ZSlider({
    super.key,
    required this.value,
    required this.min,
    required this.max,
    required this.onChanged,
    this.onChangeEnd,
    this.enabled = true,
  });

  final double value;
  final double min;
  final double max;
  final ValueChanged<double> onChanged;
  final ValueChanged<double>? onChangeEnd;
  final bool enabled;

  @override
  Widget build(BuildContext context) => SliderTheme(
        data: SliderThemeData(
          trackHeight: 6,
          activeTrackColor: Z.oliveMid,
          inactiveTrackColor: Z.surface3,
          thumbColor: Z.oliveBright,
          overlayColor: Colors.transparent,
          trackShape: const RectangularSliderTrackShape(),
          thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 11),
          disabledActiveTrackColor: Z.surface3,
          disabledInactiveTrackColor: Z.surface3,
          disabledThumbColor: Z.inkFaint,
        ),
        child: Slider(
          value: value.clamp(min, max),
          min: min,
          max: max,
          onChanged: enabled ? onChanged : null,
          onChangeEnd: enabled ? onChangeEnd : null,
        ),
      );
}

/// Connection dot: olive live, amber working, rose dead.
class ZStatusDot extends StatelessWidget {
  const ZStatusDot(this.tone, {super.key});
  final ZTone tone;

  @override
  Widget build(BuildContext context) => AnimatedContainer(
        duration: Z.normal,
        curve: Z.ease,
        width: 9,
        height: 9,
        decoration: BoxDecoration(
          color: switch (tone) {
            ZTone.good => Z.oliveSoft,
            ZTone.working => Z.warning,
            ZTone.bad => Z.danger,
          },
          shape: BoxShape.circle,
        ),
      );
}

enum ZTone { good, working, bad }

/// The heart of the "say it, don't hide it" rule.
///
/// Renders an unavailable feature as a visible, inert card carrying its name,
/// the reason it cannot run, and — when there is one — the fix. Used for host
/// capabilities and for phone-side permissions alike.
class ZCapabilityNotice extends StatefulWidget {
  const ZCapabilityNotice({
    super.key,
    required this.cap,
    required this.state,
    this.action,
    this.actionLabel,
    this.compact = false,
  });

  final ZrCap cap;
  final ZrCapState state;
  final VoidCallback? action;
  final String? actionLabel;
  final bool compact;

  @override
  State<ZCapabilityNotice> createState() => _ZCapabilityNoticeState();
}

class _ZCapabilityNoticeState extends State<ZCapabilityNotice> {
  bool _open = false;

  ZrCap get cap => widget.cap;
  ZrCapState get state => widget.state;
  VoidCallback? get action => widget.action;
  String? get actionLabel => widget.actionLabel;
  bool get compact => widget.compact;

  @override
  Widget build(BuildContext context) {
    final l = L10n.of(context);
    final reason = l.capReason(cap, state);
    if (reason == null) return const SizedBox.shrink();

    final tone = state == ZrCapState.agentOld ? Z.warning : Z.inkMuted;
    return ZCard(
      dimmed: true,
      padding: EdgeInsets.all(compact ? Z.s3 : Z.s4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          GestureDetector(
            onTap: () => setState(() => _open = !_open),
            behavior: HitTestBehavior.opaque,
            child: Row(
              children: [
                Icon(
                  state == ZrCapState.agentOld
                      ? Icons.upgrade_rounded
                      : Icons.info_outline_rounded,
                  size: 18,
                  color: tone,
                ),
                const SizedBox(width: Z.s2),
                Expanded(
                  child: Text(
                    l.capTitle(cap),
                    style: Z.body.copyWith(fontWeight: FontWeight.w600),
                  ),
                ),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                  decoration: BoxDecoration(
                    color: Z.surface3,
                    borderRadius: BorderRadius.circular(Z.rSm),
                    border: Border.all(color: Z.line),
                  ),
                  child: Text(l.t('cap_unavailable'),
                      style: Z.mono.copyWith(fontSize: 11.5, color: tone)),
                ),
                const SizedBox(width: Z.s2),
                Icon(
                  _open ? Icons.expand_less_rounded : Icons.expand_more_rounded,
                  size: 20,
                  color: Z.inkFaint,
                ),
              ],
            ),
          ),
          // The reason is one tap away, not on screen by default: the name of
          // the feature plus "unavailable" already answers "is it me?", and
          // the paragraph only matters to whoever wants to fix it.
          AnimatedSize(
            duration: Z.fast,
            curve: Z.ease,
            alignment: Alignment.topLeft,
            child: !_open
                ? const SizedBox(width: double.infinity)
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const SizedBox(height: Z.s2),
                      Text(reason, style: Z.bodySoft.copyWith(fontSize: 15.5)),
                      if (state == ZrCapState.agentOld) ...[
                        const SizedBox(height: Z.s2),
                        Container(
                          width: double.infinity,
                          padding: const EdgeInsets.symmetric(
                              horizontal: Z.s3, vertical: Z.s2),
                          decoration: BoxDecoration(
                            color: Z.surface2,
                            borderRadius: BorderRadius.circular(Z.rSm),
                            border: Border.all(color: Z.line),
                          ),
                          child: Text(l.t('cap_agent_update_cmd'),
                              style: Z.mono),
                        ),
                      ],
                      if (action != null && actionLabel != null) ...[
                        const SizedBox(height: Z.s3),
                        ZButton(
                          label: actionLabel!,
                          kind: ZButtonKind.ghost,
                          size: ZButtonSize.sm,
                          onPressed: action,
                        ),
                      ],
                    ],
                  ),
          ),
        ],
      ),
    );
  }
}

/// A titled block that starts closed. Everything the keyboard offers is
/// reachable, but a phone screen shows what you asked for rather than the whole
/// catalogue at once.
class ZSection extends StatefulWidget {
  const ZSection({
    super.key,
    required this.title,
    required this.child,
    this.initiallyOpen = false,
    this.trailing,
  });

  final String title;
  final Widget child;
  final bool initiallyOpen;
  final Widget? trailing;

  @override
  State<ZSection> createState() => _ZSectionState();
}

class _ZSectionState extends State<ZSection> {
  late bool _open = widget.initiallyOpen;

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          GestureDetector(
            onTap: () {
              HapticFeedback.selectionClick();
              setState(() => _open = !_open);
            },
            behavior: HitTestBehavior.opaque,
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: Z.s2),
              child: Row(
                children: [
                  Expanded(
                      child: Text(widget.title.toUpperCase(), style: Z.eyebrow)),
                  ?widget.trailing,
                  Icon(
                    _open
                        ? Icons.expand_less_rounded
                        : Icons.expand_more_rounded,
                    size: 20,
                    color: Z.inkFaint,
                  ),
                ],
              ),
            ),
          ),
          AnimatedSize(
            duration: Z.fast,
            curve: Z.ease,
            alignment: Alignment.topCenter,
            child: _open
                ? Padding(
                    padding: const EdgeInsets.only(bottom: Z.s2),
                    child: widget.child,
                  )
                : const SizedBox(width: double.infinity),
          ),
        ],
      );
}

/// Entrance: a short rise + fade, staggered by [index]. Honours the platform's
/// reduce-motion setting.
class ZReveal extends StatefulWidget {
  const ZReveal({super.key, required this.child, this.index = 0});
  final Widget child;
  final int index;

  @override
  State<ZReveal> createState() => _ZRevealState();
}

class _ZRevealState extends State<ZReveal> with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: Z.slow,
  );
  Timer? _stagger;

  @override
  void initState() {
    super.initState();
    // keep the handle: an un-cancelled delay outlives the widget and shows up
    // as a pending timer in tests (and as a needless wake-up in the app).
    _stagger = Timer(Duration(milliseconds: 40 * widget.index), () {
      if (mounted) _c.forward();
    });
  }

  @override
  void dispose() {
    _stagger?.cancel();
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (MediaQuery.maybeDisableAnimationsOf(context) ?? false) {
      return widget.child;
    }
    final curved = CurvedAnimation(parent: _c, curve: Z.ease);
    return FadeTransition(
      opacity: curved,
      child: SlideTransition(
        position:
            Tween(begin: const Offset(0, .04), end: Offset.zero).animate(curved),
        child: widget.child,
      ),
    );
  }
}

/// Bottom sheet on palette. Content scrolls; the sheet never fills the screen.
Future<T?> showZSheet<T>(
  BuildContext context, {
  required String title,
  required Widget child,
}) {
  return showModalBottomSheet<T>(
    context: context,
    backgroundColor: Colors.transparent,
    barrierColor: Z.scrim,
    isScrollControlled: true,
    builder: (context) => Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Container(
        decoration: const BoxDecoration(
          color: Z.surface1,
          borderRadius: BorderRadius.vertical(top: Radius.circular(Z.rXl)),
          border: Border(top: BorderSide(color: Z.lineStrong)),
        ),
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * .88,
        ),
        child: SafeArea(
          top: false,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(Z.s4, Z.s3, Z.s2, Z.s2),
                child: Row(
                  children: [
                    Expanded(child: Text(title, style: Z.title)),
                    IconButton(
                      icon: const Icon(Icons.close_rounded, color: Z.inkSoft),
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ],
                ),
              ),
              const Divider(height: 1, color: Z.line),
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(Z.s4, Z.s3, Z.s4, Z.s5),
                  child: child,
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

void zToast(BuildContext context, String message) {
  ScaffoldMessenger.of(context)
    ..clearSnackBars()
    ..showSnackBar(
      SnackBar(
        content: Text(message, style: Z.body),
        backgroundColor: Z.surface3,
        duration: const Duration(seconds: 3),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(Z.rMd),
          side: const BorderSide(color: Z.lineStrong),
        ),
      ),
    );
}
