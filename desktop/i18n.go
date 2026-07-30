package main

import (
	"os"
	"strings"
)

// EN + FR, resolved from the OS locale (or --lang / the saved preference).
// English is the fallback: ZlefRemote is a general-audience tool, not a
// French-data project.
type Lang struct {
	code string
	d    map[string]string
}

func (l *Lang) T(key string) string {
	if v, ok := l.d[key]; ok {
		return v
	}
	if v, ok := dictEN[key]; ok {
		return v
	}
	return key
}

func (l *Lang) Code() string { return l.code }

func pickLang(pref string) *Lang {
	code := strings.ToLower(pref)
	if code == "" {
		for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
			if v := os.Getenv(env); v != "" {
				code = strings.ToLower(v)
				break
			}
		}
	}
	if strings.HasPrefix(code, "fr") {
		return &Lang{code: "fr", d: dictFR}
	}
	return &Lang{code: "en", d: dictEN}
}

var dictEN = map[string]string{
	"app.title":       "ZlefRemote",
	"app.tagline":     "control another computer, end-to-end encrypted",
	"connect.paste":   "Paste the pairing link the agent printed",
	"connect.hint":    "Enter to connect · Ctrl+V to paste · Tab to pick a saved machine",
	"connect.saved":   "Saved machines",
	"connect.none":    "No saved machine yet. Start the agent on the other computer with --remember, then paste its link here.",
	"connect.forget":  "Del to forget",
	"connect.connect": "Connect",
	"connect.help":    "The agent prints a link like https://remote.zlef.fr/r/AB12CD#k=… — copy it whole, the part after # is the key and never reaches our servers.",

	"state.connecting":   "Connecting…",
	"state.linked":       "Linked — waiting for the host…",
	"state.paired":       "Connected",
	"state.reconnecting": "Connection lost — reconnecting…",
	"state.closed":       "Session closed",
	"state.nostream":     "Waiting for the first frame…",
	"state.noscreen":     "This host cannot share its screen (agent built without screen capture). Keyboard and mouse still work.",

	"hud.take":       "Right Ctrl — take control",
	"hud.release":    "Right Ctrl — release",
	"hud.help":       "Right Ctrl + / — shortcuts",
	"hud.grabbed":    "controlling",
	"hud.released":   "released",
	"hud.raw":        "raw keys",
	"hud.text":       "layout-safe typing",
	"hud.monitor":    "screen",
	"hud.latency":    "latency",
	"hud.clipon":     "clipboard shared",
	"hud.clipoff":    "clipboard off",
	"hud.click":      "click to take control",
	"help.title":     "Shortcuts",
	"help.grab":      "Right Ctrl — take / release control",
	"help.fullscr":   "Right Ctrl + F — fullscreen",
	"help.preset":    "Right Ctrl + P — image quality",
	"help.monitor":   "Right Ctrl + M — next monitor",
	"help.raw":       "Right Ctrl + K — raw keyboard (games)",
	"help.hud":       "Right Ctrl + H — hide this bar",
	"help.clip":      "Right Ctrl + C — push my clipboard",
	"help.cad":       "Right Ctrl + Del — send Ctrl+Alt+Del",
	"help.alttab":    "Right Ctrl + Tab — send Alt+Tab",
	"help.super":     "Right Ctrl + S — send the Windows/Super key",
	"help.esc":       "Right Ctrl + Esc — send Escape",
	"help.quit":      "Right Ctrl + Q — disconnect",
	"help.close":     "any key to close",
	"toast.clipin":   "clipboard received",
	"toast.clipout":  "clipboard sent",
	"toast.preset":   "quality: ",
	"toast.monitor":  "screen ",
	"toast.raw.on":   "raw keyboard on — characters follow the host layout",
	"toast.raw.off":  "layout-safe typing on",
	"toast.saved":    "machine saved",
	"error.link":     "That link doesn't look right: ",
	"error.conn":     "Could not connect: ",
	"error.room":     "This room no longer exists — the agent is probably not running.",
	"error.hostgone": "The other computer closed the session.",
}

var dictFR = map[string]string{
	"app.title":       "ZlefRemote",
	"app.tagline":     "contrôlez un autre ordinateur, chiffré de bout en bout",
	"connect.paste":   "Collez le lien d'appairage affiché par l'agent",
	"connect.hint":    "Entrée pour se connecter · Ctrl+V pour coller · Tab pour choisir une machine",
	"connect.saved":   "Machines enregistrées",
	"connect.none":    "Aucune machine enregistrée. Lancez l'agent sur l'autre ordinateur avec --remember, puis collez son lien ici.",
	"connect.forget":  "Suppr pour oublier",
	"connect.connect": "Se connecter",
	"connect.help":    "L'agent affiche un lien du type https://remote.zlef.fr/r/AB12CD#k=… — copiez-le en entier : ce qui suit le # est la clé, elle n'atteint jamais nos serveurs.",

	"state.connecting":   "Connexion…",
	"state.linked":       "Liaison établie — en attente de l'hôte…",
	"state.paired":       "Connecté",
	"state.reconnecting": "Connexion perdue — reconnexion…",
	"state.closed":       "Session terminée",
	"state.nostream":     "En attente de la première image…",
	"state.noscreen":     "Cet hôte ne peut pas partager son écran (agent compilé sans capture). Le clavier et la souris fonctionnent quand même.",

	"hud.take":       "Ctrl droite — prendre la main",
	"hud.release":    "Ctrl droite — rendre la main",
	"hud.help":       "Ctrl droite + / — raccourcis",
	"hud.grabbed":    "aux commandes",
	"hud.released":   "main rendue",
	"hud.raw":        "touches brutes",
	"hud.text":       "frappe sans souci de disposition",
	"hud.monitor":    "écran",
	"hud.latency":    "latence",
	"hud.clipon":     "presse-papiers partagé",
	"hud.clipoff":    "presse-papiers désactivé",
	"hud.click":      "cliquez pour prendre la main",
	"help.title":     "Raccourcis",
	"help.grab":      "Ctrl droite — prendre / rendre la main",
	"help.fullscr":   "Ctrl droite + F — plein écran",
	"help.preset":    "Ctrl droite + P — qualité d'image",
	"help.monitor":   "Ctrl droite + M — écran suivant",
	"help.raw":       "Ctrl droite + K — clavier brut (jeux)",
	"help.hud":       "Ctrl droite + H — masquer cette barre",
	"help.clip":      "Ctrl droite + C — envoyer mon presse-papiers",
	"help.cad":       "Ctrl droite + Suppr — envoyer Ctrl+Alt+Suppr",
	"help.alttab":    "Ctrl droite + Tab — envoyer Alt+Tab",
	"help.super":     "Ctrl droite + S — envoyer la touche Windows/Super",
	"help.esc":       "Ctrl droite + Échap — envoyer Échap",
	"help.quit":      "Ctrl droite + Q — se déconnecter",
	"help.close":     "une touche pour fermer",
	"toast.clipin":   "presse-papiers reçu",
	"toast.clipout":  "presse-papiers envoyé",
	"toast.preset":   "qualité : ",
	"toast.monitor":  "écran ",
	"toast.raw.on":   "clavier brut — les caractères suivent la disposition de l'hôte",
	"toast.raw.off":  "frappe sans souci de disposition",
	"toast.saved":    "machine enregistrée",
	"error.link":     "Ce lien ne va pas : ",
	"error.conn":     "Connexion impossible : ",
	"error.room":     "Ce salon n'existe plus — l'agent n'est sans doute pas lancé.",
	"error.hostgone": "L'autre ordinateur a fermé la session.",
}
