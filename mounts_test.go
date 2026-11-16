package main

import "testing"

func TestMountedIn(t *testing.T) {
	table := `/dev/disk3s1s1 on / (apfs, sealed, local, read-only, journaled)
android on /Users/marc/Android (macfuse, local, nodev, nosuid, synchronous, mounted by marc)
`
	if !mountedIn(table, "/Users/marc/Android") {
		t.Error("montage présent non détecté")
	}
	if mountedIn(table, "/Users/marc/Andro") || mountedIn(table, "/Users/marc") {
		t.Error("un préfixe du chemin ne doit pas compter comme monté")
	}
}
