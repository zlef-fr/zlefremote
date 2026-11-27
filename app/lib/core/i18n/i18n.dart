import 'package:flutter/widgets.dart';

import '../caps.dart';
import 'strings_en.dart';
import 'strings_fr.dart';

/// Two locales, so no language picker: the app follows the phone and lets
/// Settings override it. English is the fallback — ZlefRemote is a general
/// tool, not a French-data one.
class L10n {
  const L10n(this.locale);

  final Locale locale;

  static const supported = [Locale('en'), Locale('fr')];

  static L10n of(BuildContext context) =>
      Localizations.of<L10n>(context, L10n) ?? const L10n(Locale('en'));

  Map<String, String> get _table =>
      locale.languageCode == 'fr' ? stringsFr : stringsEn;

  String get languageCode => locale.languageCode;

  String t(String key) => _table[key] ?? stringsEn[key] ?? key;

  /// `{name}` placeholders.
  String f(String key, Map<String, String> args) {
    var out = t(key);
    args.forEach((k, v) => out = out.replaceAll('{$k}', v));
    return out;
  }

  /// Title of a capability, e.g. "Shared clipboard".
  String capTitle(ZrCap cap) => t('cap_${cap.name}_title');

  /// Plain-language reason a capability is off, or null when it is usable.
  /// Falls back to a generic line rather than showing nothing — silence is
  /// exactly what this app refuses to do.
  String? capReason(ZrCap cap, ZrCapState state) {
    if (state == ZrCapState.ready) return null;
    final specific = _table['cap_${cap.name}_${state.name}'] ??
        stringsEn['cap_${cap.name}_${state.name}'];
    if (specific != null) return specific;
    return switch (state) {
      ZrCapState.agentOld => t('cap_screen_agentOld'),
      _ => t('cap_unavailable'),
    };
  }

  String relativeTime(DateTime? when) {
    if (when == null) return t('never_used');
    final seconds = DateTime.now().difference(when).inSeconds;
    if (seconds < 45) return t('just_now');
    final minutes = seconds ~/ 60;
    if (minutes < 60) return '$minutes ${t('unit_min')}';
    final hours = minutes ~/ 60;
    if (hours < 24) return '$hours ${t('unit_hour')}';
    return '${hours ~/ 24} ${t('unit_day')}';
  }
}

class L10nDelegate extends LocalizationsDelegate<L10n> {
  const L10nDelegate();

  @override
  bool isSupported(Locale locale) =>
      L10n.supported.any((l) => l.languageCode == locale.languageCode);

  @override
  Future<L10n> load(Locale locale) async => L10n(locale);

  @override
  bool shouldReload(L10nDelegate old) => false;
}
