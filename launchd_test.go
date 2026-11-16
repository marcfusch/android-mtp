package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plistStrings renvoie les valeurs <string> du plist, dans l'ordre.
func plistStrings(t *testing.T, data []byte) []string {
	t.Helper()
	var doc struct {
		Dict struct {
			Strings []string `xml:"string"`
			Array   struct {
				Strings []string `xml:"string"`
			} `xml:"array"`
		} `xml:"dict"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("plist invalide : %v\n%s", err, data)
	}
	return append(doc.Dict.Array.Strings, doc.Dict.Strings...)
}

func TestAgentPlist(t *testing.T) {
	a := newAgent("/Users/a & <b>")
	got := plistStrings(t, a.plist())
	want := []string{
		"/Users/a & <b>/.local/bin/android", "watch",
		label, "Interactive", a.Log, a.Log,
	}
	if len(got) != len(want) {
		t.Fatalf("plist = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("plist[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewAgentKeepsOutOfDocuments(t *testing.T) {
	a := newAgent("/Users/x")
	for _, p := range []string{a.Bin, a.Log, a.Plist} {
		if rel, _ := filepath.Rel("/Users/x/Documents", p); !strings.HasPrefix(rel, "..") {
			t.Errorf("%s est dans ~/Documents, illisible pour launchd", p)
		}
	}
}

func TestCopyExecutableReplaces(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "bin", "android")
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := copyExecutable(src, dst); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	info, _ := os.Stat(dst)
	if string(data) != "v2" || info.Mode().Perm() != 0o755 {
		t.Errorf("dst = %q %v, want \"v2\" -rwxr-xr-x", data, info.Mode().Perm())
	}
	if _, err := os.Stat(dst + ".new"); !os.IsNotExist(err) {
		t.Error("le fichier temporaire est resté")
	}
}
