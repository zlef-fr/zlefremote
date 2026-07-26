package main

import "strings"

// Two locales (en default + fr) → resolved silently from the OS UI language,
// no language picker (zlef DA rule for 2-locale projects). ZLEFREMOTE_LANG or
// -lang overrides for testing.

type strs map[string]string

var enStrings = strs{
	"title":         "ZlefRemote",
	"tagline":       "Your phone is the trackpad",
	"mode_lan":      "Local network",
	"mode_lan_d":    "Same Wi-Fi · fastest, fully local",
	"mode_remote":   "Remote",
	"mode_remote_d": "From anywhere · end-to-end encrypted",
	"start":         "Start",
	"stop":          "Stop",
	"st_idle":       "Stopped",
	"st_starting":   "Starting…",
	"st_waiting":    "Scan the code with your phone",
	"st_paired":     "Phone connected",
	"remember":      "Remember this computer",
	"remember_d":    "Saved phones reconnect in one tap",
	"remember_old":  "Update the ZlefRemote agent (1.1.0+) to use this",
	"saved_hint":    "Saved — your phone can reconnect to this computer anytime",
	"clients_1":     "1 phone connected",
	"clients_n":     "%d phones connected",
	"copy":          "Copy link",
	"copied":        "Copied!",
	"open_phone":    "Or open this link on your phone:",
	"no_agent":      "Agent not found. Install the ZlefRemote agent.",
	"unknown_ip":    "unknown",

	// tray / menu (Windows-only surface)
	"menu_show":      "Open ZlefRemote",
	"menu_start_lan": "Start on the local network",
	"menu_start_rem": "Start remotely",
	"menu_stop":      "Stop the session",
	"menu_copy":      "Copy the pairing link",
	"menu_autostart": "Start with Windows",
	"menu_update":    "Update the agent…",
	"menu_website":   "remote.zlef.fr",
	"menu_quit":      "Quit",
	"tip_idle":       "ZlefRemote — stopped",
	"tip_starting":   "ZlefRemote — starting…",
	"tip_waiting":    "ZlefRemote — waiting for a phone",
	"tip_paired":     "ZlefRemote — phone connected",
	"balloon_paired": "Your phone is now this computer's trackpad and keyboard.",
	"balloon_bye":    "The phone disconnected.",
	"update_done":    "The agent is up to date.",
	"update_fail":    "Could not update the agent.",
	"hint_tray":      "Right-click the tray icon for more.",
}

var frStrings = strs{
	"title":         "ZlefRemote",
	"tagline":       "Votre téléphone devient le trackpad",
	"mode_lan":      "Réseau local",
	"mode_lan_d":    "Même Wi-Fi · le plus rapide, 100% local",
	"mode_remote":   "À distance",
	"mode_remote_d": "Depuis partout · chiffré de bout en bout",
	"start":         "Démarrer",
	"stop":          "Arrêter",
	"st_idle":       "Arrêté",
	"st_starting":   "Démarrage…",
	"st_waiting":    "Scannez le code avec votre téléphone",
	"st_paired":     "Téléphone connecté",
	"remember":      "Mémoriser cet ordinateur",
	"remember_d":    "Les téléphones enregistrés se reconnectent en un geste",
	"remember_old":  "Mettez à jour l'agent ZlefRemote (1.1.0+) pour l'utiliser",
	"saved_hint":    "Enregistré — votre téléphone peut se reconnecter à cet ordinateur à tout moment",
	"clients_1":     "1 téléphone connecté",
	"clients_n":     "%d téléphones connectés",
	"copy":          "Copier le lien",
	"copied":        "Copié !",
	"open_phone":    "Ou ouvrez ce lien sur votre téléphone :",
	"no_agent":      "Agent introuvable. Installez l'agent ZlefRemote.",
	"unknown_ip":    "inconnu",

	"menu_show":      "Ouvrir ZlefRemote",
	"menu_start_lan": "Démarrer sur le réseau local",
	"menu_start_rem": "Démarrer à distance",
	"menu_stop":      "Arrêter la session",
	"menu_copy":      "Copier le lien d'appairage",
	"menu_autostart": "Démarrer avec Windows",
	"menu_update":    "Mettre à jour l'agent…",
	"menu_website":   "remote.zlef.fr",
	"menu_quit":      "Quitter",
	"tip_idle":       "ZlefRemote — arrêté",
	"tip_starting":   "ZlefRemote — démarrage…",
	"tip_waiting":    "ZlefRemote — en attente d'un téléphone",
	"tip_paired":     "ZlefRemote — téléphone connecté",
	"balloon_paired": "Votre téléphone est maintenant le trackpad et le clavier de cet ordinateur.",
	"balloon_bye":    "Le téléphone s'est déconnecté.",
	"update_done":    "L'agent est à jour.",
	"update_fail":    "Impossible de mettre à jour l'agent.",
	"hint_tray":      "Clic droit sur l'icône pour plus d'options.",
}

var active = enStrings

// setLang picks the dictionary from a BCP-47-ish tag ("fr", "fr-FR", "en-US").
func setLang(tag string) {
	if strings.HasPrefix(strings.ToLower(tag), "fr") {
		active = frStrings
		return
	}
	active = enStrings
}

// t looks a string up; unknown keys return the key (loud but harmless).
func t(key string) string {
	if s, ok := active[key]; ok {
		return s
	}
	return key
}
