/// English — the fallback locale. ZlefRemote is a general-audience tool, not a
/// French-data one, so `en` is the default when the phone asks for anything we
/// don't ship.
const Map<String, String> stringsEn = {
  'app_name': 'ZlefRemote',
  'tagline': 'Your phone is the trackpad.',

  // ── devices ────────────────────────────────────────────────────────────────
  'devices_title': 'Your computers',
  'devices_sub': 'Tap one to take control.',
  'devices_empty_title': 'No computers yet',
  'devices_empty_sub':
      'Run the ZlefRemote agent on a computer, then scan the QR code it shows.',
  'devices_empty_cta': 'Add a computer',
  'add_device': 'Add a computer',
  'add_title': 'Add a computer',
  'add_scan_title': 'Scan the agent’s QR code',
  'add_scan_hint':
      'Start ZlefRemote on the computer and point the camera at the code it shows.',
  'add_camera_denied': 'Camera access is off',
  'add_camera_denied_sub':
      'ZlefRemote needs the camera to read a pairing code. Allow it in Android settings, or paste the link instead.',
  'add_torch': 'Torch',
  'add_paste_title': 'Or paste the pairing link',
  'add_paste_ph': 'https://remote.zlef.fr/r/…#k=…',
  'add_paste_from_clipboard': 'Paste from clipboard',
  'add_connect': 'Connect',
  'add_bad_link':
      'That isn’t a ZlefRemote pairing link. Copy the one the agent prints and try again.',
  'add_clipboard_empty': 'Nothing on the clipboard.',
  'unknown_computer': 'Computer',
  'rename': 'Rename',
  'rename_title': 'Rename computer',
  'rename_hint': 'Name on this phone only',
  'remove': 'Remove',
  'remove_title': 'Remove this computer?',
  'remove_body':
      'Its key is deleted from this phone. You will need a new QR scan to control it again.',
  'cancel': 'Cancel',
  'save': 'Save',
  'last_used': 'Last used',
  'just_now': 'just now',
  'never_used': 'never used',
  'unit_min': 'min',
  'unit_hour': 'h',
  'unit_day': 'd',
  'lan_device': 'Wi-Fi only',
  'one_shot_device': 'One-time pairing',
  'one_shot_note':
      'This computer wasn’t started in remember mode, so it can’t be reconnected later. Run the agent with --remember for one-tap access.',

  // ── connection ─────────────────────────────────────────────────────────────
  'connecting': 'Connecting…',
  'linking': 'Pairing securely…',
  'paired': 'Connected',
  'reconnecting': 'Reconnecting…',
  'disconnected': 'Disconnected',
  'closed_host': 'The computer ended the session.',
  'err_room_title': 'Computer is offline',
  'err_room_body':
      'It isn’t reachable right now. Start ZlefRemote on it, then try again.',
  'err_full_title': 'That computer is busy',
  'err_full_body': 'Four phones are already connected to it.',
  'err_connect_title': 'Can’t reach the relay',
  'err_connect_body': 'Check this phone’s connection and try again.',
  'err_lan_title': 'Not on the same Wi-Fi',
  'err_lan_body':
      'This computer was paired over your local network. Join that Wi-Fi, or pair it again in remote mode to reach it from anywhere.',
  'try_again': 'Try again',
  'back_to_devices': 'Computers',
  'e2ee': 'End-to-end encrypted',
  'e2ee_long':
      'Everything you send is sealed on this phone and opened only by that computer. The relay moves ciphertext it cannot read.',
  'latency': 'ping',

  // ── control surfaces ───────────────────────────────────────────────────────
  'tab_pad': 'Trackpad',
  'tab_screen': 'Screen',
  'tab_keys': 'Keyboard',
  'tab_media': 'Media',
  'pad_hint': 'Drag · tap · 2 fingers scroll · 3 fingers switch app',
  'btn_left': 'Left',
  'btn_right': 'Right',
  'btn_mid': 'Middle',
  'drag_lock': 'Drag lock',
  'drag_lock_on': 'Drag lock on — the left button is held down',
  'scroll_rail': 'Scroll',

  // screen
  'screen_waiting': 'Waiting for the screen…',
  'screen_hint':
      'Tap to click · double-tap to double-click · two-finger tap right-clicks · pinch to zoom',
  'screen_failed': 'The computer couldn’t capture its screen.',
  'screen_start': 'Show the screen',
  'screen_stop': 'Stop',
  'quality': 'Quality',
  'q_low': 'Low',
  'q_balanced': 'Balanced',
  'q_sharp': 'Sharp',
  'display': 'Display',
  'zoom_reset': 'Fit',

  // keyboard
  'keys_type_ph': 'Type — every character goes straight to the computer',
  'keys_echo_hint': 'What you type appears here, then on the computer.',
  'key_esc': 'Esc',
  'key_tab': 'Tab',
  'key_enter': 'Enter',
  'key_backspace': 'Backspace',
  'key_delete': 'Delete',
  'key_space': 'Space',
  'key_home': 'Home',
  'key_end': 'End',
  'key_pgup': 'PgUp',
  'key_pgdn': 'PgDn',
  'mod_ctrl': 'Ctrl',
  'mod_alt': 'Alt',
  'mod_shift': 'Shift',
  'mod_meta': 'Win/⌘',
  'mods_hint': 'Tap a modifier, then a key — it clears itself after one press.',
  'fkeys': 'Function keys',
  'shortcuts': 'Shortcuts',
  'sc_copy': 'Copy',
  'sc_paste': 'Paste',
  'sc_cut': 'Cut',
  'sc_undo': 'Undo',
  'sc_selectall': 'Select all',
  'sc_switch': 'Switch app',
  'sc_close': 'Close window',
  'sc_lock': 'Lock screen',

  // media
  'media_transport': 'Playback',
  'vol_down': 'Volume down',
  'vol_up': 'Volume up',
  'mute': 'Mute',
  'play_pause': 'Play / Pause',
  'previous': 'Previous',
  'next': 'Next',
  'brightness': 'Brightness',
  'bright_all': 'All screens',
  'bright_screen': 'Screen',
  'bright_method': 'Method',
  'bright_software_note':
      'Software dimming: it reaches external monitors, but colours wash out.',
  'bright_floor_note': 'Never goes below 5% — you still need to see the screen.',

  // clipboard
  'clipboard': 'Shared clipboard',
  'clip_from_host': 'Copied on the computer',
  'clip_copy_here': 'Copy to this phone',
  'clip_send': 'Send this phone’s clipboard',
  'clip_sent': 'Sent to the computer',
  'clip_copied': 'Copied',
  'clip_empty': 'Nothing copied on the computer yet.',

  // ── settings ───────────────────────────────────────────────────────────────
  'settings': 'Settings',
  'section_pointer': 'Pointer',
  'pointer_speed': 'Pointer speed',
  'scroll_speed': 'Scroll speed',
  'natural_scroll': 'Natural scrolling',
  'natural_scroll_note': 'Content follows your finger, like a phone.',
  'section_phone': 'This phone',
  'haptics': 'Haptic feedback',
  'haptics_note': 'A short buzz on clicks and key presses.',
  'keep_awake': 'Keep the screen on',
  'keep_awake_note': 'While a computer is connected.',
  'lock_unlock_for_more': 'Unlock the phone for the trackpad and keyboard',
  'lock_controls': 'Lock-screen controls',
  'lock_controls_note':
      'Shows playback and volume over the lock screen, the way a navigation app does — no unlocking, and no borrowing the music player’s card. The trackpad, keyboard and your saved computers stay behind the lock. Turn this off if you would rather the phone show nothing until it is unlocked.',
  'volume_keys': 'Volume rocker controls the computer',
  'volume_keys_note':
      'While a session is open, the phone’s volume buttons change the computer’s volume instead of its own.',
  'section_language': 'Language',
  'language_auto': 'Follow the phone',
  'section_about': 'About',
  'version': 'Version',
  'privacy': 'Privacy',
  'open_source': 'Source code',
  'agent_download': 'Get the computer agent',

  // ── capability disclosure ──────────────────────────────────────────────────
  // Shown wherever a feature exists but cannot run. Never hide the feature.
  'cap_unavailable': 'Not available',
  'cap_why': 'Why?',
  'cap_agent_update_cmd': 'On the computer, run:  zlefremote-agent -update',

  'cap_screen_title': 'Screen sharing',
  'cap_screen_hostLacks':
      'This computer’s agent was built without screen capture, so it has no picture to send. Reinstall the agent from remote.zlef.fr to get a build that can.',
  'cap_screen_agentOld':
      'Screen sharing needs agent 1.2 or newer, and this computer is running something older.',

  'cap_brightness_title': 'Brightness',
  'cap_brightness_hostLacks':
      'This computer exposes no brightness control. Laptops normally do; a desktop driving an external monitor normally doesn’t. On Linux, installing brightnessctl usually makes it appear.',

  'cap_brightnessPerScreen_title': 'Per-monitor brightness',
  'cap_brightnessPerScreen_hostLacks':
      'Only one adjustable display was found, so the slider already controls it.',

  'cap_brightnessMethod_title': 'Brightness method',
  'cap_brightnessMethod_hostLacks':
      'This computer has a single way to change brightness, so there is nothing to pick between.',

  'cap_clipboard_title': 'Shared clipboard',
  'cap_clipboard_hostLacks':
      'The agent found no clipboard tool on this computer. On Linux install xclip, xsel or wl-clipboard and restart it.',
  'cap_clipboard_agentOld':
      'A shared clipboard needs agent 1.7 or newer, and this computer is running something older.',

  'cap_keyHold_title': 'Held keys',
  'cap_keyHold_hostLacks':
      'This computer can tap keys but not hold them, so shift-drag and key repeat won’t work.',
  'cap_keyHold_agentOld':
      'Holding keys needs agent 1.7 or newer. Taps work fine meanwhile; shift-drag and key repeat don’t.',

  'cap_multiMonitor_title': 'Multiple monitors',
  'cap_multiMonitor_hostLacks': 'This computer reports a single display.',

  'cap_lockScreenControls_title': 'Lock-screen controls',
  'cap_lockScreenControls_needsPermission':
      'Android needs permission to post notifications before the media card can appear.',
  'cap_lockScreenControls_phoneLacks':
      'This Android version won’t show the media card.',
  'grant_permission': 'Allow notifications',

  'cap_agentUpdate_title': 'Update the computer’s agent',
  'cap_agentUpdate_agentOld':
      'This agent is too old to be updated from here. On the computer, run zlefremote-agent -update once; after that this button does it for you.',
  'agent_update': 'Update the agent',
  'agent_update_running': 'Updating the agent…',
  'agent_update_done': 'Updated to {v} — restart the agent on the computer to use it.',
  'agent_update_current': 'The agent is already up to date.',
  'agent_update_failed': 'Update failed: {reason}',
  'section_computer': 'This computer',
  'cap_backgroundSession_title': 'Session with the screen off',
  'cap_backgroundSession_needsPermission':
      'Android is restricting this app in the background, so the connection to your computer is cut the moment the screen goes off — the session looks alive and answers nothing. Allow background activity (Settings › Battery › unrestricted) to keep it.',
  'open_app_settings': 'Open app settings',
  'background_restricted_banner':
      'The screen going off will cut this session: Android has this app background-restricted.',

  'cap_volumeKeys_title': 'Volume rocker',
  'cap_volumeKeys_phoneLacks':
      'This phone doesn’t let the app read its volume buttons.',

  'agent_old_banner':
      'This computer runs an agent older than 1.7. Some features below are switched off until it is updated.',
  'agent_old_dismiss': 'Got it',

  // ── updates ────────────────────────────────────────────────────────────────
  'update_title': 'Update',
  'update_check': 'Check for updates',
  'update_checking': 'Checking…',
  'update_current': 'You’re on the latest version.',
  'update_available': 'Version {v} is available',
  'update_download': 'Download',
  'update_downloading': 'Downloading…',
  'update_install': 'Install',
  'update_failed': 'Update failed. Try again later.',
  'update_needs_permission':
      'Android must allow this app to install packages. Grant it, then tap Install again.',
};
