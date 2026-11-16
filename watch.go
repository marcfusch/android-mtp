package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanwen/usb"
)

type action int

const (
	none action = iota
	mount
	unmount
	release
)

// tracker décide quoi faire à chaque tour de watch. Il ne touche à rien, pour être testé seul.
type tracker struct {
	mounted string // interface montée
	ejected string // interface éjectée par l'utilisateur, remontée au prochain branchement seulement
	warned  string // interface pour laquelle l'utilisateur a déjà été prévenu d'un échec
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

// mountFailed renvoie true au premier échec sur une interface, pour ne prévenir qu'une fois.
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

// watch monte le téléphone à chaque branchement et le démonte au débranchement. SIGUSR1 remonte
// un téléphone éjecté.
func watch(dir string) {
	ctx := usb.NewContext()
	remount := make(chan os.Signal, 1)
	signal.Notify(remount, syscall.SIGUSR1)
	tick := time.NewTicker(2 * time.Second)

	if isMounted(dir) {
		forceUnmount(dir)
	}

	var t tracker
	var cur *phone
	for {
		s := readUSB()
		killed := freezePtpcamerad(s)
		if killed {
			log.Print("ptpcamerad tenait le téléphone")
		}

		switch t.next(s.iface, cur != nil && cur.ejected()) {
		case release:
			log.Print("éjecté")
			cur.release()
			cur = nil
		case unmount:
			log.Print("débranché")
			cur.unmount(dir)
			cur = nil
		case mount:
			p, err := mountPhone(ctx, dir)
			if err != nil {
				log.Print(err)
				if t.mountFailed(s.iface) {
					notify(failureMessage(err, killed))
				}
				break
			}
			log.Print("monté")
			notify("Téléphone monté dans ~/Android")
			cur = p
			t.mountSucceeded(s.iface)
		}

		select {
		case <-tick.C:
		case <-remount:
			t.remount()
		}
	}
}

func failureMessage(err error, killed bool) string {
	switch {
	case killed:
		return "Débranche et rebranche le téléphone"
	case errors.Is(err, errLocked):
		return "Déverrouille le téléphone et choisis Transfert de fichiers"
	}
	return err.Error()
}

func notify(msg string) {
	exec.Command("osascript", "-e", fmt.Sprintf("display notification %q with title \"Android\"", msg)).Run()
}
