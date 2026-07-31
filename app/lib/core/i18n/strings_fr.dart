/// Français. Toute clé absente retombe sur l'anglais (strings_en.dart).
const Map<String, String> stringsFr = {
  'app_name': 'ZlefRemote',
  'tagline': 'Votre téléphone devient le trackpad.',

  // ── appareils ──────────────────────────────────────────────────────────────
  'devices_title': 'Vos ordinateurs',
  'devices_sub': 'Touchez-en un pour le contrôler.',
  'devices_empty_title': 'Aucun ordinateur',
  'devices_empty_sub':
      'Lancez l’agent ZlefRemote sur un ordinateur, puis scannez le QR code affiché.',
  'devices_empty_cta': 'Ajouter un ordinateur',
  'add_device': 'Ajouter un ordinateur',
  'add_title': 'Ajouter un ordinateur',
  'add_scan_title': 'Scannez le QR code de l’agent',
  'add_scan_hint':
      'Lancez ZlefRemote sur l’ordinateur et visez le code qu’il affiche.',
  'add_camera_denied': 'Accès caméra refusé',
  'add_camera_denied_sub':
      'ZlefRemote a besoin de la caméra pour lire un code d’appairage. Autorisez-la dans les réglages Android, ou collez le lien.',
  'add_torch': 'Lampe',
  'add_paste_title': 'Ou collez le lien d’appairage',
  'add_paste_ph': 'https://remote.zlef.fr/r/…#k=…',
  'add_paste_from_clipboard': 'Coller depuis le presse-papiers',
  'add_connect': 'Connecter',
  'add_bad_link':
      'Ce n’est pas un lien d’appairage ZlefRemote. Copiez celui qu’affiche l’agent et réessayez.',
  'add_clipboard_empty': 'Presse-papiers vide.',
  'unknown_computer': 'Ordinateur',
  'rename': 'Renommer',
  'rename_title': 'Renommer l’ordinateur',
  'rename_hint': 'Nom visible sur ce téléphone uniquement',
  'remove': 'Retirer',
  'remove_title': 'Retirer cet ordinateur ?',
  'remove_body':
      'Sa clé est supprimée de ce téléphone. Il faudra rescanner un QR code pour le contrôler à nouveau.',
  'cancel': 'Annuler',
  'save': 'Enregistrer',
  'last_used': 'Dernier usage',
  'just_now': 'à l’instant',
  'never_used': 'jamais utilisé',
  'unit_min': 'min',
  'unit_hour': 'h',
  'unit_day': 'j',
  'lan_device': 'Wi-Fi seulement',
  'one_shot_device': 'Appairage unique',
  'one_shot_note':
      'Cet ordinateur n’a pas été lancé en mode mémoire : impossible de s’y reconnecter plus tard. Lancez l’agent avec --remember pour un accès en un geste.',

  // ── connexion ──────────────────────────────────────────────────────────────
  'connecting': 'Connexion…',
  'linking': 'Appairage sécurisé…',
  'paired': 'Connecté',
  'reconnecting': 'Reconnexion…',
  'disconnected': 'Déconnecté',
  'closed_host': 'L’ordinateur a mis fin à la session.',
  'err_room_title': 'Ordinateur hors ligne',
  'err_room_body':
      'Il n’est pas joignable pour le moment. Lancez ZlefRemote dessus, puis réessayez.',
  'err_full_title': 'Cet ordinateur est occupé',
  'err_full_body': 'Quatre téléphones y sont déjà connectés.',
  'err_connect_title': 'Relais injoignable',
  'err_connect_body': 'Vérifiez la connexion de ce téléphone et réessayez.',
  'err_lan_title': 'Pas sur le même Wi-Fi',
  'err_lan_body':
      'Cet ordinateur a été appairé sur votre réseau local. Rejoignez ce Wi-Fi, ou réappairez-le en mode distant pour l’atteindre de partout.',
  'try_again': 'Réessayer',
  'back_to_devices': 'Ordinateurs',
  'e2ee': 'Chiffré de bout en bout',
  'e2ee_long':
      'Tout ce que vous envoyez est scellé sur ce téléphone et ouvert par cet ordinateur seul. Le relais ne transporte que du chiffré, illisible pour lui.',
  'latency': 'ping',

  // ── surfaces de contrôle ───────────────────────────────────────────────────
  'tab_pad': 'Trackpad',
  'tab_screen': 'Écran',
  'tab_keys': 'Clavier',
  'tab_media': 'Média',
  'pad_hint': 'Glissez · tapez · 2 doigts défilent · 3 doigts changent d’app',
  'btn_left': 'Gauche',
  'btn_right': 'Droit',
  'btn_mid': 'Milieu',
  'drag_lock': 'Verrou glisser',
  'drag_lock_on': 'Verrou actif — le bouton gauche reste enfoncé',
  'scroll_rail': 'Défiler',

  'screen_waiting': 'En attente de l’écran…',
  'screen_hint':
      'Tapez pour cliquer · double-tap = double-clic · tap à 2 doigts = clic droit · pincez pour zoomer',
  'screen_failed': 'L’ordinateur n’a pas pu capturer son écran.',
  'screen_start': 'Afficher l’écran',
  'screen_stop': 'Arrêter',
  'quality': 'Qualité',
  'q_low': 'Basse',
  'q_balanced': 'Équilibrée',
  'q_sharp': 'Nette',
  'display': 'Écran',
  'zoom_reset': 'Ajuster',

  'keys_type_ph': 'Tapez — chaque caractère part directement vers l’ordinateur',
  'keys_echo_hint': 'Ce que vous tapez s’affiche ici, puis sur l’ordinateur.',
  'key_esc': 'Échap',
  'key_tab': 'Tab',
  'key_enter': 'Entrée',
  'key_backspace': 'Retour',
  'key_delete': 'Suppr',
  'key_space': 'Espace',
  'key_home': 'Début',
  'key_end': 'Fin',
  'key_pgup': 'Page ↑',
  'key_pgdn': 'Page ↓',
  'mod_ctrl': 'Ctrl',
  'mod_alt': 'Alt',
  'mod_shift': 'Maj',
  'mod_meta': 'Win/⌘',
  'mods_hint':
      'Touchez un modificateur, puis une touche — il se désactive après un appui.',
  'fkeys': 'Touches F',
  'shortcuts': 'Raccourcis',
  'sc_copy': 'Copier',
  'sc_paste': 'Coller',
  'sc_cut': 'Couper',
  'sc_undo': 'Annuler',
  'sc_selectall': 'Tout sélectionner',
  'sc_switch': 'Changer d’app',
  'sc_close': 'Fermer la fenêtre',
  'sc_lock': 'Verrouiller l’écran',

  'media_transport': 'Lecture',
  'vol_down': 'Volume −',
  'vol_up': 'Volume +',
  'mute': 'Muet',
  'play_pause': 'Lecture / Pause',
  'previous': 'Précédent',
  'next': 'Suivant',
  'brightness': 'Luminosité',
  'bright_all': 'Tous les écrans',
  'bright_screen': 'Écran',
  'bright_method': 'Méthode',
  'bright_software_note':
      'Atténuation logicielle : elle atteint les écrans externes, mais délave les couleurs.',
  'bright_floor_note':
      'Jamais sous 5 % — il faut encore pouvoir voir l’écran.',

  'clipboard': 'Presse-papiers partagé',
  'clip_from_host': 'Copié sur l’ordinateur',
  'clip_copy_here': 'Copier sur ce téléphone',
  'clip_send': 'Envoyer le presse-papiers du téléphone',
  'clip_sent': 'Envoyé à l’ordinateur',
  'clip_copied': 'Copié',
  'clip_empty': 'Rien n’a encore été copié sur l’ordinateur.',

  // ── réglages ───────────────────────────────────────────────────────────────
  'settings': 'Réglages',
  'section_pointer': 'Pointeur',
  'pointer_speed': 'Vitesse du pointeur',
  'scroll_speed': 'Vitesse de défilement',
  'natural_scroll': 'Défilement naturel',
  'natural_scroll_note': 'Le contenu suit le doigt, comme sur un téléphone.',
  'section_phone': 'Ce téléphone',
  'haptics': 'Retour haptique',
  'haptics_note': 'Une brève vibration aux clics et aux touches.',
  'keep_awake': 'Garder l’écran allumé',
  'keep_awake_note': 'Tant qu’un ordinateur est connecté.',
  'lock_unlock_for_more': 'Déverrouillez pour le trackpad et le clavier',
  'lock_controls': 'Contrôles sur l’écran verrouillé',
  'lock_controls_note':
      'Affiche lecture et volume par-dessus l’écran verrouillé, comme une app de navigation — sans déverrouiller, et sans emprunter la carte du lecteur de musique. Le trackpad, le clavier et vos ordinateurs enregistrés restent derrière le verrou. Désactivez si vous préférez que le téléphone n’affiche rien tant qu’il est verrouillé.',
  'lock_full': 'Contrôle complet sur l’écran verrouillé',
  'lock_full_note':
      'Place toute la télécommande — trackpad, clavier, écran — par-dessus l’écran verrouillé, au lieu de la seule lecture et du volume. Pratique, et à comprendre : quiconque prend ce téléphone peut alors piloter votre ordinateur sans connaître votre code.',
  'lock_full_confirm_title': 'Autoriser le contrôle complet verrouillé ?',
  'lock_full_confirm_body':
      'Toute personne tenant ce téléphone pourra déplacer la souris, taper et voir l’écran de votre ordinateur sans le déverrouiller. Vos ordinateurs enregistrés et les réglages de l’app restent protégés.',
  'lock_full_confirm_ok': 'Autoriser',
  'lock_full_warning':
      'Le contrôle complet est accessible sans déverrouiller ce téléphone.',
  'volume_keys': 'Le volume du téléphone pilote l’ordinateur',
  'volume_keys_note':
      'Pendant une session, les boutons de volume changent le volume de l’ordinateur au lieu de celui du téléphone.',
  'section_language': 'Langue',
  'language_auto': 'Suivre le téléphone',
  'section_about': 'À propos',
  'version': 'Version',
  'privacy': 'Confidentialité',
  'open_source': 'Code source',
  'agent_download': 'Télécharger l’agent pour ordinateur',

  // ── capacités indisponibles ────────────────────────────────────────────────
  'cap_unavailable': 'Indisponible',
  'cap_why': 'Pourquoi ?',
  'cap_agent_update_cmd':
      'Sur l’ordinateur, lancez :  zlefremote-agent -update',

  'cap_screen_title': 'Partage d’écran',
  'cap_screen_hostLacks':
      'L’agent de cet ordinateur a été compilé sans capture d’écran : il n’a aucune image à envoyer. Réinstallez l’agent depuis remote.zlef.fr pour obtenir une version capable de le faire.',
  'cap_screen_agentOld':
      'Le partage d’écran demande l’agent 1.2 ou plus récent ; celui-ci est plus ancien.',

  'cap_brightness_title': 'Luminosité',
  'cap_brightness_hostLacks':
      'Cet ordinateur n’expose aucun réglage de luminosité. Les portables en ont normalement un ; une tour reliée à un écran externe, non. Sous Linux, installer brightnessctl le fait généralement apparaître.',

  'cap_brightnessPerScreen_title': 'Luminosité par écran',
  'cap_brightnessPerScreen_hostLacks':
      'Un seul écran réglable a été trouvé : le curseur le contrôle déjà.',

  'cap_brightnessMethod_title': 'Méthode de luminosité',
  'cap_brightnessMethod_hostLacks':
      'Cet ordinateur n’a qu’une seule façon de changer la luminosité : rien à choisir.',

  'cap_clipboard_title': 'Presse-papiers partagé',
  'cap_clipboard_hostLacks':
      'L’agent n’a trouvé aucun outil de presse-papiers sur cet ordinateur. Sous Linux, installez xclip, xsel ou wl-clipboard puis relancez-le.',
  'cap_clipboard_agentOld':
      'Le presse-papiers partagé demande l’agent 1.7 ou plus récent ; celui-ci est plus ancien.',

  'cap_keyHold_title': 'Touches maintenues',
  'cap_keyHold_hostLacks':
      'Cet ordinateur peut taper les touches mais pas les maintenir : Maj+glisser et la répétition ne fonctionneront pas.',
  'cap_keyHold_agentOld':
      'Maintenir une touche demande l’agent 1.7 ou plus récent. Les appuis simples marchent ; Maj+glisser et la répétition non.',

  'cap_multiMonitor_title': 'Plusieurs écrans',
  'cap_multiMonitor_hostLacks': 'Cet ordinateur ne déclare qu’un seul écran.',

  'cap_lockScreenControls_title': 'Contrôles sur l’écran verrouillé',
  'cap_lockScreenControls_needsPermission':
      'Android doit autoriser les notifications avant que la carte média puisse apparaître.',
  'cap_lockScreenControls_phoneLacks':
      'Cette version d’Android n’affichera pas la carte média.',
  'grant_permission': 'Autoriser les notifications',

  'cap_agentUpdate_title': 'Mettre à jour l’agent de l’ordinateur',
  'cap_agentUpdate_agentOld':
      'Cet agent est trop ancien pour être mis à jour d’ici. Sur l’ordinateur, lancez zlefremote-agent -update une fois ; ensuite ce bouton s’en charge.',
  'agent_update': 'Mettre à jour l’agent',
  'agent_update_running': 'Mise à jour de l’agent…',
  'agent_update_done': 'Mis à jour en {v} — relancez l’agent sur l’ordinateur pour en profiter.',
  'agent_update_current': 'L’agent est déjà à jour.',
  'agent_update_failed': 'Échec de la mise à jour : {reason}',
  'section_computer': 'Cet ordinateur',
  'cap_backgroundSession_title': 'Session écran éteint',
  'cap_backgroundSession_needsPermission':
      'Android restreint cette app en arrière-plan : la connexion à votre ordinateur est coupée dès que l’écran s’éteint — la session paraît vivante et ne répond plus. Autorisez l’activité en arrière-plan (Réglages › Batterie › sans restriction) pour la conserver.',
  'open_app_settings': 'Ouvrir les réglages de l’app',
  'background_restricted_banner':
      'L’extinction de l’écran coupera cette session : Android restreint cette app en arrière-plan.',

  'cap_volumeKeys_title': 'Boutons de volume',
  'cap_volumeKeys_phoneLacks':
      'Ce téléphone ne laisse pas l’application lire ses boutons de volume.',

  'agent_old_banner':
      'Cet ordinateur utilise un agent antérieur à 1.7. Certaines fonctions ci-dessous restent désactivées tant qu’il n’est pas mis à jour.',
  'agent_old_dismiss': 'Compris',

  // ── mises à jour ───────────────────────────────────────────────────────────
  'update_title': 'Mise à jour',
  'update_check': 'Rechercher une mise à jour',
  'update_checking': 'Recherche…',
  'update_current': 'Vous avez la dernière version.',
  'update_available': 'La version {v} est disponible',
  'update_download': 'Télécharger',
  'update_downloading': 'Téléchargement…',
  'update_install': 'Installer',
  'update_failed': 'Échec de la mise à jour. Réessayez plus tard.',
  'update_needs_permission':
      'Android doit autoriser cette app à installer des paquets. Accordez-le, puis touchez à nouveau Installer.',
};
