package main

import (
	"context"
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

// watch est la boucle de l'agent. SIGUSR1 remonte un téléphone éjecté, SIGTERM démonte et quitte.
func watch(dir string) {
	ctx := usb.NewContext()
	remount := make(chan os.Signal, 1)
	signal.Notify(remount, syscall.SIGUSR1)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
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
		case <-stop:
			if cur != nil {
				cur.unmount(dir)
			}
			return
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

// notify n'attend pas osascript, qui peut ne jamais rendre la main.
func notify(msg string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exec.CommandContext(ctx, "osascript", "-e", fmt.Sprintf("display notification %q with title \"Android\"", msg)).Run()
	}()
}
