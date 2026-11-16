package mtp

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

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
