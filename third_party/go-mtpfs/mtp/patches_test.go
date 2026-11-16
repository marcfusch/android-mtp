package mtp

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestEndsOnFullPacket(t *testing.T) {
	for _, tt := range []struct {
		sent int64
		want bool
	}{
		{0, false},
		{12, false},
		{511, false},
		{512, true},
		{12 + 500, true},
		{12 + 1<<20, false},
		{12 + 1<<20 + 500, true},
	} {
		if got := endsOnFullPacket(tt.sent, 512); got != tt.want {
			t.Errorf("endsOnFullPacket(%d) = %v, want %v", tt.sent, got, tt.want)
		}
	}
}

func decodeTimeString(t *testing.T, s string) time.Time {
	t.Helper()
	enc, err := encodeStr(make([]byte, 2*len(s)+3), s)
	if err != nil {
		t.Fatal(err)
	}
	var got time.Time
	if err := decodeTime(bytes.NewReader(enc), reflect.ValueOf(&got).Elem()); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDecodeTimeIsLocalUnlessUTC(t *testing.T) {
	if got, want := decodeTimeString(t, "20261115T200500"), time.Date(2026, 11, 15, 20, 5, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("local: %v, want %v", got, want)
	}
	if got, want := decodeTimeString(t, "20261115T200500Z"), time.Date(2026, 11, 15, 20, 5, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("UTC: %v, want %v", got, want)
	}
}
