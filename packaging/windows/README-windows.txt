ZlefRemote for Windows — your phone is the trackpad
====================================================

What you just installed
-----------------------
  ZlefRemote.exe          the tray app: pick a mode, hit Start, scan the QR
  zlefremote-agent.exe    the agent it drives (also usable on its own, from a
                          terminal, with -mode lan|remote)

Using it
--------
1. Click the ZlefRemote icon in the notification area (the up-arrow next to the
   clock hides new icons — drag it onto the taskbar to keep it visible).
2. Choose a mode:
     Local network — phone and PC on the same Wi-Fi. Fastest, nothing leaves
                     your network.
     Remote        — anywhere. The pairing goes through remote.zlef.fr, which
                     only ever sees ciphertext: the key lives in the QR's #k=
                     fragment and is never sent to the server.
3. Press Start and scan the QR with your phone's camera. That's it — the phone
   becomes a trackpad, a keyboard, a media remote and a live screen view.

"Remember this computer" (Remote mode) keeps one encryption key on this PC so
saved phones reconnect in one tap instead of scanning a new code every time.

Windows Firewall
----------------
The first time you start a session in Local network mode, Windows asks whether
to let zlefremote-agent.exe accept incoming connections. Allow it on Private
networks — that is the LAN listener your phone connects to. Remote mode makes
only outbound connections and needs no rule.

Right-click menu
----------------
  Start with Windows   puts the icon back in the tray at every sign-in
  Update the agent…    in-place update, checksum-verified, from remote.zlef.fr
  Quit                 stops the session and removes the icon

Where things live
-----------------
  Program        %LOCALAPPDATA%\Programs\ZlefRemote
  Preferences    HKCU\Software\ZlefRemote  (mode, "remember" tick)
  Saved identity %APPDATA%\zlefremote\identity  (only with "Remember this
                 computer"; deleting it rotates this PC's address)

Uninstalling removes the program, the shortcuts and the preferences. It leaves
the saved identity alone so a reinstall keeps your saved phones working.

Not signed yet: SmartScreen may warn on first run ("More info" → "Run anyway").

remote.zlef.fr
