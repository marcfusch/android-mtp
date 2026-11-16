package main

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ptpcamerad (Transfert d'images) prend le téléphone au branchement et ne le rend jamais.
// launchd le relance dès qu'il meurt, même désactivé, mais une copie gelée par SIGSTOP reste en
// place sans rien prendre.

// freezePtpcamerad gèle ptpcamerad. S'il tient déjà le téléphone, il faut le tuer, ce qui peut
// figer le téléphone jusqu'au prochain branchement : elle renvoie alors true.
func freezePtpcamerad(s usbStatus) (killed bool) {
	if s.owner == "ptpcamerad" {
		syscall.Kill(s.ownerPID, syscall.SIGKILL)
		killed = true
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			if pids := ptpcameradPIDs(); len(pids) > 0 && pids[0] != s.ownerPID {
				break
			}
		}
	}
	signalPtpcamerad(syscall.SIGSTOP)
	return killed
}

func signalPtpcamerad(sig syscall.Signal) {
	for _, pid := range ptpcameradPIDs() {
		syscall.Kill(pid, sig)
	}
}

func ptpcameradPIDs() []int {
	out, _ := exec.Command("pgrep", "-x", "ptpcamerad").Output()
	return parsePIDs(string(out))
}

func parsePIDs(out string) []int {
	var pids []int
	for _, f := range strings.Fields(out) {
		if pid, err := strconv.Atoi(f); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}
