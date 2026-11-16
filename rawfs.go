package main

import (
	"github.com/hanwen/go-fuse/v2/fuse"
	"golang.org/x/text/unicode/norm"
)

// macRawFS convertit les noms en NFC, car macOS les passe décomposés et le téléphone les garde
// précomposés. Il accepte aussi les attributs étendus sans les garder : MTP n'en a pas, et le
// Finder abandonne une copie s'il ne peut pas les écrire.
type macRawFS struct {
	fuse.RawFileSystem
}

func nfc(name string) string {
	return norm.NFC.String(name)
}

func (fs macRawFS) Lookup(cancel <-chan struct{}, header *fuse.InHeader, name string, out *fuse.EntryOut) fuse.Status {
	return fs.RawFileSystem.Lookup(cancel, header, nfc(name), out)
}

func (fs macRawFS) Mknod(cancel <-chan struct{}, input *fuse.MknodIn, name string, out *fuse.EntryOut) fuse.Status {
	return fs.RawFileSystem.Mknod(cancel, input, nfc(name), out)
}

func (fs macRawFS) Mkdir(cancel <-chan struct{}, input *fuse.MkdirIn, name string, out *fuse.EntryOut) fuse.Status {
	return fs.RawFileSystem.Mkdir(cancel, input, nfc(name), out)
}

func (fs macRawFS) Create(cancel <-chan struct{}, input *fuse.CreateIn, name string, out *fuse.CreateOut) fuse.Status {
	return fs.RawFileSystem.Create(cancel, input, nfc(name), out)
}

func (fs macRawFS) Unlink(cancel <-chan struct{}, header *fuse.InHeader, name string) fuse.Status {
	return fs.RawFileSystem.Unlink(cancel, header, nfc(name))
}

func (fs macRawFS) Rmdir(cancel <-chan struct{}, header *fuse.InHeader, name string) fuse.Status {
	return fs.RawFileSystem.Rmdir(cancel, header, nfc(name))
}

func (fs macRawFS) Rename(cancel <-chan struct{}, input *fuse.RenameIn, oldName, newName string) fuse.Status {
	return fs.RawFileSystem.Rename(cancel, input, nfc(oldName), nfc(newName))
}

func (fs macRawFS) Link(cancel <-chan struct{}, input *fuse.LinkIn, name string, out *fuse.EntryOut) fuse.Status {
	return fs.RawFileSystem.Link(cancel, input, nfc(name), out)
}

func (fs macRawFS) Symlink(cancel <-chan struct{}, header *fuse.InHeader, target, name string, out *fuse.EntryOut) fuse.Status {
	return fs.RawFileSystem.Symlink(cancel, header, target, nfc(name), out)
}

func (macRawFS) GetXAttr(cancel <-chan struct{}, header *fuse.InHeader, attr string, dest []byte) (uint32, fuse.Status) {
	return 0, fuse.ENOATTR
}

func (macRawFS) ListXAttr(cancel <-chan struct{}, header *fuse.InHeader, dest []byte) (uint32, fuse.Status) {
	return 0, fuse.OK
}

func (macRawFS) SetXAttr(cancel <-chan struct{}, input *fuse.SetXAttrIn, attr string, data []byte) fuse.Status {
	return fuse.OK
}

func (macRawFS) RemoveXAttr(cancel <-chan struct{}, header *fuse.InHeader, attr string) fuse.Status {
	return fuse.ENOATTR
}
