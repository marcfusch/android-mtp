package main

import (
	"errors"
	"fmt"
	"testing"
)

// step est un tour de watch : ce que voit l'agent, ce qu'il doit faire et le résultat du montage.
type step struct {
	iface       string
	userEjected bool
	want        action
	mountOK     bool
}

func run(t *testing.T, tr *tracker, steps []step) {
	t.Helper()
	for i, s := range steps {
		got := tr.next(s.iface, s.userEjected)
		if got != s.want {
			t.Fatalf("tour %d (%+v) : action %d, attendu %d", i, s, got, s.want)
		}
		if got == mount {
			if s.mountOK {
				tr.mountSucceeded(s.iface)
			} else {
				tr.mountFailed(s.iface)
			}
		}
	}
}

func TestTrackerPlugAndUnplug(t *testing.T) {
	run(t, &tracker{}, []step{
		{iface: "", want: none},
		{iface: "a", want: mount, mountOK: true},
		{iface: "a", want: none},
		{iface: "", want: unmount},
		{iface: "", want: none},
		{iface: "b", want: mount, mountOK: true},
	})
}

func TestTrackerRetriesUntilUnlocked(t *testing.T) {
	run(t, &tracker{}, []step{
		{iface: "a", want: mount},
		{iface: "a", want: mount},
		{iface: "a", want: mount, mountOK: true},
		{iface: "a", want: none},
	})
}

func TestTrackerEjectWaitsForReplug(t *testing.T) {
	run(t, &tracker{}, []step{
		{iface: "a", want: mount, mountOK: true},
		{iface: "a", userEjected: true, want: release},
		{iface: "a", want: none},
		{iface: "a", want: none},
		{iface: "", want: none},
		{iface: "b", want: mount, mountOK: true},
	})
}

func TestTrackerRemountAfterEject(t *testing.T) {
	tr := &tracker{}
	run(t, tr, []step{
		{iface: "a", want: mount, mountOK: true},
		{iface: "a", userEjected: true, want: release},
		{iface: "a", want: none},
	})
	tr.remount()
	run(t, tr, []step{{iface: "a", want: mount, mountOK: true}})
}

func TestTrackerReenumeratedWhileMounted(t *testing.T) {
	run(t, &tracker{}, []step{
		{iface: "a", want: mount, mountOK: true},
		{iface: "b", want: unmount},
		{iface: "b", want: mount, mountOK: true},
	})
}

func TestTrackerWarnsOncePerPlug(t *testing.T) {
	tr := &tracker{}
	if !tr.mountFailed("a") {
		t.Error("le premier échec doit prévenir")
	}
	if tr.mountFailed("a") {
		t.Error("un second échec sur la même interface ne doit pas prévenir")
	}
	tr.next("", false)
	if !tr.mountFailed("a") {
		t.Error("après un débranchement, un échec doit de nouveau prévenir")
	}
	if !tr.mountFailed("b") {
		t.Error("une nouvelle interface doit prévenir")
	}
}

func TestFailureMessage(t *testing.T) {
	locked := fmt.Errorf("%w (timeout)", errLocked)
	other := errors.New("aucun appareil MTP")
	tests := []struct {
		err    error
		killed bool
		want   string
	}{
		{locked, true, "Débranche et rebranche le téléphone"},
		{locked, false, "Déverrouille le téléphone et choisis Transfert de fichiers"},
		{other, false, "aucun appareil MTP"},
	}
	for _, tt := range tests {
		if got := failureMessage(tt.err, tt.killed); got != tt.want {
			t.Errorf("failureMessage(%v, %v) = %q, want %q", tt.err, tt.killed, got, tt.want)
		}
	}
}
