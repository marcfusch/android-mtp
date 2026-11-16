package main

type action int

const (
	none action = iota
	mount
	unmount
	release
)

// tracker décide de chaque tour de watch, sans effet de bord.
type tracker struct {
	mounted string
	ejected string // éjectée par l'utilisateur : pas remontée avant le prochain branchement
	warned  string // déjà prévenu d'un échec sur cette interface
}

func (t *tracker) next(iface string, userEjected bool) action {
	if t.mounted != "" {
		switch {
		case userEjected:
			t.ejected, t.mounted = t.mounted, ""
			return release
		case iface != t.mounted:
			t.mounted = ""
			return unmount
		}
		return none
	}
	switch iface {
	case "":
		t.ejected, t.warned = "", ""
		return none
	case t.ejected:
		return none
	}
	return mount
}

func (t *tracker) mountSucceeded(iface string) {
	t.mounted, t.warned = iface, ""
}

// mountFailed dit s'il faut prévenir : seulement au premier échec sur une interface.
func (t *tracker) mountFailed(iface string) bool {
	if t.warned == iface {
		return false
	}
	t.warned = iface
	return true
}

func (t *tracker) remount() {
	t.ejected = ""
}
