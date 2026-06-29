// Client (remote) strings. EN default, FR. 2 locales → no selector; silent
// resolution: zl-lang cookie → navigator.language → en.
const ZRI18n = (() => {
  const S = {
    en: {
      connecting: 'Connecting…', linking: 'Pairing securely…', reconnecting: 'Reconnecting…',
      paired: 'Connected', waiting: 'Waiting for your computer…',
      nokey: 'This link is missing its encryption key. Scan the QR code from the agent again.',
      closed: 'Disconnected', closed_host: 'The computer ended the session.',
      err_connect: 'Could not reach the relay. Check your connection.',
      err_room: 'That room no longer exists — restart the agent and rescan.',
      err_full: 'This room is full.',
      to: 'controlling', secure: 'end-to-end encrypted',
      tab_pad: 'Trackpad', tab_keys: 'Keyboard', tab_media: 'Media',
      hint_pad: 'Drag to move · tap to click · two-finger tap = right-click · two fingers to scroll',
      btn_left: 'Left click', btn_right: 'Right click', btn_mid: 'Middle', btn_drag: 'Drag lock',
      type_ph: 'Type here — text is sent as you type',
      send: 'Send', enter: 'Enter', back: '⌫', tab: 'Tab', esc: 'Esc', space: 'Space',
      win: 'Win/⌘', ctrl: 'Ctrl', alt: 'Alt', shift: 'Shift',
      arrows: 'Arrows', fkeys: 'Function keys',
      vol_down: 'Vol −', vol_up: 'Vol +', mute: 'Mute', play: 'Play/Pause', prev: 'Prev', next: 'Next',
      reconnect: 'Reconnect', settings: 'Settings', sensitivity: 'Pointer speed',
      scroll_speed: 'Scroll speed', natural_scroll: 'Natural scrolling', close: 'Close',
      // home / saved devices
      home_title: 'Your devices', home_sub: 'Tap a computer to take control.',
      saved: 'Saved devices', add_device: 'Add a device', connect: 'Connect',
      empty_title: 'No saved computers yet', empty_sub: 'Add one to control it from your phone in a single tap.',
      add_title: 'Add a device', add_scan: 'Scan QR code', add_scan_hint: 'Point your phone at the QR code shown by the ZlefRemote agent.',
      add_paste: 'Or paste a pairing link', add_paste_ph: 'https://remote.zlef.fr/r/…#k=…', add_go: 'Connect',
      add_bad: 'That doesn’t look like a ZlefRemote link. Copy it from the agent and try again.',
      scan_denied: 'Camera access was blocked. Paste the link instead.', scan_unsupported: 'QR scanning isn’t available on this browser — paste the link below.',
      last_used: 'Last used', just_now: 'just now', now_connecting: 'Connecting…',
      rename: 'Rename', remove: 'Remove', remove_q: 'Remove this device?', rename_ph: 'Device name',
      saved_toast: 'Saved — reconnect anytime from your devices.',
      ephemeral_note: 'This computer wasn’t started in remember mode, so it can’t be saved. Run the agent with --remember to reconnect in one tap.',
      offline_title: 'Computer is offline', offline_sub: 'It isn’t reachable right now. Start ZlefRemote on it (with --remember), then try again.',
      try_again: 'Try again', back_home: 'Devices', save_device: 'Save this device',
      new_device: 'New computer', unknown_device: 'Computer',
      install_app: 'Install app',
    },
    fr: {
      connecting: 'Connexion…', linking: 'Appairage sécurisé…', reconnecting: 'Reconnexion…',
      paired: 'Connecté', waiting: 'En attente de votre ordinateur…',
      nokey: 'Ce lien n’a pas sa clé de chiffrement. Rescannez le QR code de l’agent.',
      closed: 'Déconnecté', closed_host: 'L’ordinateur a mis fin à la session.',
      err_connect: 'Relais injoignable. Vérifiez votre connexion.',
      err_room: 'Ce salon n’existe plus — relancez l’agent et rescannez.',
      err_full: 'Ce salon est plein.',
      to: 'contrôle de', secure: 'chiffré de bout en bout',
      tab_pad: 'Trackpad', tab_keys: 'Clavier', tab_media: 'Média',
      hint_pad: 'Glissez pour bouger · tap pour cliquer · tap à 2 doigts = clic droit · 2 doigts pour défiler',
      btn_left: 'Clic gauche', btn_right: 'Clic droit', btn_mid: 'Milieu', btn_drag: 'Verrou glisser',
      type_ph: 'Tapez ici — le texte est envoyé à la frappe',
      send: 'Envoyer', enter: 'Entrée', back: '⌫', tab: 'Tab', esc: 'Échap', space: 'Espace',
      win: 'Win/⌘', ctrl: 'Ctrl', alt: 'Alt', shift: 'Maj',
      arrows: 'Flèches', fkeys: 'Touches F',
      vol_down: 'Vol −', vol_up: 'Vol +', mute: 'Muet', play: 'Lecture/Pause', prev: 'Préc.', next: 'Suiv.',
      reconnect: 'Reconnecter', settings: 'Réglages', sensitivity: 'Vitesse du pointeur',
      scroll_speed: 'Vitesse de défilement', natural_scroll: 'Défilement naturel', close: 'Fermer',
      // accueil / appareils enregistrés
      home_title: 'Vos appareils', home_sub: 'Touchez un ordinateur pour le contrôler.',
      saved: 'Appareils enregistrés', add_device: 'Ajouter un appareil', connect: 'Connecter',
      empty_title: 'Aucun ordinateur enregistré', empty_sub: 'Ajoutez-en un pour le contrôler depuis votre téléphone en un seul geste.',
      add_title: 'Ajouter un appareil', add_scan: 'Scanner le QR code', add_scan_hint: 'Visez le QR code affiché par l’agent ZlefRemote.',
      add_paste: 'Ou collez un lien d’appairage', add_paste_ph: 'https://remote.zlef.fr/r/…#k=…', add_go: 'Connecter',
      add_bad: 'Ce lien ne ressemble pas à un lien ZlefRemote. Copiez-le depuis l’agent et réessayez.',
      scan_denied: 'Accès à la caméra refusé. Collez plutôt le lien.', scan_unsupported: 'Le scan QR n’est pas disponible sur ce navigateur — collez le lien ci-dessous.',
      last_used: 'Dernier usage', just_now: 'à l’instant', now_connecting: 'Connexion…',
      rename: 'Renommer', remove: 'Retirer', remove_q: 'Retirer cet appareil ?', rename_ph: 'Nom de l’appareil',
      saved_toast: 'Enregistré — reconnectez-vous quand vous voulez depuis vos appareils.',
      ephemeral_note: 'Cet ordinateur n’a pas été lancé en mode mémoire ; il ne peut pas être enregistré. Lancez l’agent avec --remember pour vous reconnecter en un geste.',
      offline_title: 'Ordinateur hors ligne', offline_sub: 'Il n’est pas joignable pour le moment. Lancez ZlefRemote dessus (avec --remember), puis réessayez.',
      try_again: 'Réessayer', back_home: 'Appareils', save_device: 'Enregistrer cet appareil',
      new_device: 'Nouvel ordinateur', unknown_device: 'Ordinateur',
      install_app: 'Installer l’app',
    },
  };
  function lang() {
    const m = document.cookie.match(/(?:^|;\s*)zl-lang=(\w{2})/);
    if (m && S[m[1]]) return m[1];
    return (navigator.language || 'en').toLowerCase().startsWith('fr') ? 'fr' : 'en';
  }
  const L = lang();
  const t = (k) => (S[L] && S[L][k]) || S.en[k] || k;
  return { t, lang: L };
})();
