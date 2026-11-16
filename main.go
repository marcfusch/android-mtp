// android monte un téléphone Android dans ~/Android via MTP et macFUSE.
//
//	android            ouvre ~/Android, en remontant le téléphone s'il a été éjecté
//	android off        éjecte, sans remonter avant le prochain branchement
//	android watch      monte au branchement et démonte au débranchement (agent launchd)
//	android install    installe la commande et l'agent
//	android uninstall  retire la commande et l'agent
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Join(home, "Android")

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "":
		err = open(dir)
	case "off":
		if isMounted(dir) {
			forceUnmount(dir)
		}
	case "watch":
		watch(dir)
	case "install":
		err = install(home)
	case "uninstall":
		err = uninstall(home, dir)
	default:
		err = errors.New("usage: android [off|watch|install|uninstall]")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "android:", err)
		os.Exit(1)
	}
}

// open demande à l'agent de remonter le téléphone s'il le faut, puis ouvre le Finder.
func open(dir string) error {
	if !isMounted(dir) {
		if readUSB().iface == "" {
			return errors.New("aucun téléphone en Transfert de fichiers")
		}
		if err := signalAgent(); err != nil {
			return fmt.Errorf("l'agent ne tourne pas, lance android install (%w)", err)
		}
		if !waitMounted(dir, 15*time.Second) {
			return errLocked
		}
	}
	return exec.Command("open", dir).Run()
}

func waitMounted(dir string, timeout time.Duration) bool {
	for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		if isMounted(dir) {
			return true
		}
	}
	return false
}
