package main

import (
	"os/exec"
	"strings"
)

// isMounted lit la table des montages sans toucher au système de fichiers, qui peut ne plus répondre.
func isMounted(dir string) bool {
	out, _ := exec.Command("mount").Output()
	return mountedIn(string(out), dir)
}

func mountedIn(table, dir string) bool {
	return strings.Contains(table, " on "+dir+" (")
}

func forceUnmount(dir string) {
	if exec.Command("umount", dir).Run() != nil {
		exec.Command("diskutil", "unmount", "force", dir).Run()
	}
}
