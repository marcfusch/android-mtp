package main

import (
	"errors"
	"fmt"
	"testing"
)

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
