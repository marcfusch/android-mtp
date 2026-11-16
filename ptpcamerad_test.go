package main

import "testing"

func TestParsePIDs(t *testing.T) {
	got := parsePIDs("39052\n15546\nnope\n")
	if len(got) != 2 || got[0] != 39052 || got[1] != 15546 {
		t.Errorf("parsePIDs() = %v", got)
	}
	if got := parsePIDs(""); got != nil {
		t.Errorf("parsePIDs(\"\") = %v, want nil", got)
	}
}
