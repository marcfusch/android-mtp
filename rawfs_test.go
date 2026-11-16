package main

import (
	"testing"

	"github.com/hanwen/go-fuse/v2/fuse"
)

// recordingFS note les noms que reçoit go-mtpfs.
type recordingFS struct {
	fuse.RawFileSystem
	names []string
}

func (r *recordingFS) Lookup(_ <-chan struct{}, _ *fuse.InHeader, name string, _ *fuse.EntryOut) fuse.Status {
	r.names = append(r.names, name)
	return fuse.OK
}

func (r *recordingFS) Mkdir(_ <-chan struct{}, _ *fuse.MkdirIn, name string, _ *fuse.EntryOut) fuse.Status {
	r.names = append(r.names, name)
	return fuse.OK
}

func (r *recordingFS) Create(_ <-chan struct{}, _ *fuse.CreateIn, name string, _ *fuse.CreateOut) fuse.Status {
	r.names = append(r.names, name)
	return fuse.OK
}

func (r *recordingFS) Rename(_ <-chan struct{}, _ *fuse.RenameIn, oldName, newName string) fuse.Status {
	r.names = append(r.names, oldName, newName)
	return fuse.OK
}

func TestMacRawFSPassesNamesInNFC(t *testing.T) {
	const nfd = "partagé"
	const nfc = "partagé"
	rec := &recordingFS{RawFileSystem: fuse.NewDefaultRawFileSystem()}
	fs := macRawFS{rec}

	fs.Lookup(nil, &fuse.InHeader{}, nfd, &fuse.EntryOut{})
	fs.Mkdir(nil, &fuse.MkdirIn{}, nfd, &fuse.EntryOut{})
	fs.Create(nil, &fuse.CreateIn{}, nfd, &fuse.CreateOut{})
	fs.Rename(nil, &fuse.RenameIn{}, nfd, "toronto")

	want := []string{nfc, nfc, nfc, nfc, "toronto"}
	if len(rec.names) != len(want) {
		t.Fatalf("noms reçus = %q, want %q", rec.names, want)
	}
	for i := range want {
		if rec.names[i] != want[i] {
			t.Errorf("nom %d = %q, want %q", i, rec.names[i], want[i])
		}
	}
}

func TestMacRawFSAcceptsXattrsWithoutKeepingThem(t *testing.T) {
	fs := macRawFS{fuse.NewDefaultRawFileSystem()}
	if st := fs.SetXAttr(nil, &fuse.SetXAttrIn{}, "com.apple.FinderInfo", []byte{1}); st != fuse.OK {
		t.Errorf("SetXAttr = %v, want OK", st)
	}
	if _, st := fs.GetXAttr(nil, &fuse.InHeader{}, "com.apple.FinderInfo", nil); st != fuse.ENOATTR {
		t.Errorf("GetXAttr = %v, want ENOATTR", st)
	}
	if n, st := fs.ListXAttr(nil, &fuse.InHeader{}, nil); n != 0 || st != fuse.OK {
		t.Errorf("ListXAttr = %d %v, want 0 OK", n, st)
	}
}
