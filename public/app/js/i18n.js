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
