// EN + FR for the desktop remote. Same resolution rule as the rest of the site:
// zl-lang cookie → navigator.language → English (this is a general-audience
// tool, not a French-data project). Two locales ⇒ no picker.
const ZRDeskI18n = (() => {
  const EN = {
    connecting: 'Connecting…',
    linked: 'Linked — waiting for the computer…',
    waiting: 'Waiting for the first frame…',
    reconnecting: 'Connection lost — reconnecting…',
    closed: 'Session closed',
    nokey: 'This link carries no key. Copy the whole link the agent printed — everything after # is the key.',
    noroom: 'This room no longer exists — the agent is probably not running.',
    hostgone: 'The other computer closed the session.',
    noscreen: 'This computer cannot share its screen (agent built without capture). Keyboard and mouse still work.',

    take: 'Take control',
    take_hint: 'Click, or tap Right Ctrl. Right Ctrl again gives control back.',
    taken: 'controlling',
    released: 'control released',
    quality: 'Quality',
    monitor: 'Screen',
    clip: 'Send clipboard',
    full: 'Fullscreen',
    unfull: 'Exit fullscreen',
    quit: 'Disconnect',

    help_title: 'Shortcuts',
    help_grab: 'take / give back control',
    help_full: 'fullscreen',
    help_quality: 'image quality',
    help_monitor: 'next screen',
    help_raw: 'raw keyboard (games)',
    help_clip: 'send my clipboard',
    help_cad: 'send Ctrl+Alt+Del',
    help_alttab: 'send Alt+Tab',
    help_super: 'send the Windows/Super key',
    help_esc: 'send Escape',
    help_quit: 'disconnect',
    help_capture_full: 'Full key capture is on: this tab receives every shortcut, including Ctrl+W and Alt+Tab.',
    help_capture_part: 'Your browser keeps some shortcuts for itself (Ctrl+W, Ctrl+T, Alt+Tab, Escape…). Go fullscreen in Chrome or Edge to capture them too — or use the chords above to send them.',

    raw_on: 'raw keyboard on — characters follow the other computer’s layout',
    raw_off: 'layout-safe typing on',
    clip_in: 'clipboard received',
    clip_out: 'clipboard sent',
    clip_manual: 'clipboard: press Right Ctrl + C to send yours',
    clip_denied: 'clipboard permission refused — Right Ctrl + C still sends yours',
    q_fast: 'fast', q_balanced: 'balanced', q_sharp: 'sharp',
    ok: 'OK',
  };

  const FR = {
    connecting: 'Connexion…',
    linked: 'Liaison établie — en attente de l’ordinateur…',
    waiting: 'En attente de la première image…',
    reconnecting: 'Connexion perdue — reconnexion…',
    closed: 'Session terminée',
    nokey: 'Ce lien ne contient pas de clé. Copiez le lien entier affiché par l’agent — tout ce qui suit le # est la clé.',
    noroom: 'Ce salon n’existe plus — l’agent n’est sans doute pas lancé.',
    hostgone: 'L’autre ordinateur a fermé la session.',
    noscreen: 'Cet ordinateur ne peut pas partager son écran (agent compilé sans capture). Le clavier et la souris fonctionnent quand même.',

    take: 'Prendre la main',
    take_hint: 'Cliquez, ou appuyez sur Ctrl droite. Ctrl droite à nouveau rend la main.',
    taken: 'aux commandes',
    released: 'main rendue',
    quality: 'Qualité',
    monitor: 'Écran',
    clip: 'Envoyer le presse-papiers',
    full: 'Plein écran',
    unfull: 'Quitter le plein écran',
    quit: 'Se déconnecter',

    help_title: 'Raccourcis',
    help_grab: 'prendre / rendre la main',
    help_full: 'plein écran',
    help_quality: 'qualité d’image',
    help_monitor: 'écran suivant',
    help_raw: 'clavier brut (jeux)',
    help_clip: 'envoyer mon presse-papiers',
    help_cad: 'envoyer Ctrl+Alt+Suppr',
    help_alttab: 'envoyer Alt+Tab',
    help_super: 'envoyer la touche Windows/Super',
    help_esc: 'envoyer Échap',
    help_quit: 'se déconnecter',
    help_capture_full: 'Capture clavier totale : cet onglet reçoit tous les raccourcis, y compris Ctrl+W et Alt+Tab.',
    help_capture_part: 'Votre navigateur garde certains raccourcis (Ctrl+W, Ctrl+T, Alt+Tab, Échap…). Passez en plein écran sur Chrome ou Edge pour les capturer aussi — ou utilisez les combinaisons ci-dessus pour les envoyer.',

    raw_on: 'clavier brut — les caractères suivent la disposition de l’autre ordinateur',
    raw_off: 'frappe sans souci de disposition',
    clip_in: 'presse-papiers reçu',
    clip_out: 'presse-papiers envoyé',
    clip_manual: 'presse-papiers : Ctrl droite + C pour envoyer le vôtre',
    clip_denied: 'permission presse-papiers refusée — Ctrl droite + C envoie quand même le vôtre',
    q_fast: 'rapide', q_balanced: 'équilibré', q_sharp: 'net',
    ok: 'OK',
  };

  function pick() {
    const m = document.cookie.match(/(?:^|;\s*)zl-lang=([a-z-]+)/i);
    const raw = (m ? m[1] : (navigator.language || 'en')).toLowerCase();
    return raw.startsWith('fr') ? 'fr' : 'en';
  }

  const lang = pick();
  const dict = lang === 'fr' ? FR : EN;
  document.documentElement.lang = lang;
  return { lang, t: (k) => (k in dict ? dict[k] : (EN[k] || k)) };
})();
