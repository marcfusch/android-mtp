// Copyright 2012 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type androidNode struct {
	mtpNodeImpl

	// If set, the backing file was changed.
	write     bool
	start     time.Time
	byteCount int64

	// devSize is the size of the object on the device during an edit. Size runs ahead of it
	// while writes wait in an androidFile buffer.
	devSize int64
}

func (n *androidNode) startEdit() bool {
	if n.write {
		return true
	}

	n.start = time.Now()
	n.byteCount = 0
	n.devSize = n.Size
	if err := n.fs.dev.AndroidBeginEditObject(n.Handle()); err != nil {
		log.Println("AndroidBeginEditObject failed:", err)
		return false
	}
	n.write = true
	return true
}

func (n *androidNode) endEdit() bool {
	if !n.write {
		return true
	}

	dt := time.Now().Sub(n.start)
	log.Printf("%d bytes in %v: %d mb/s",
		n.byteCount, dt, (1e3*n.byteCount)/(dt.Nanoseconds()))

	if err := n.fs.dev.AndroidEndEditObject(n.Handle()); err != nil {
		log.Println("AndroidEndEditObject failed:", err)
		return false
	}
	n.write = false
	return true
}

var _ = (fs.NodeOpener)((*androidNode)(nil))

func (n *androidNode) Open(ctx context.Context, flags uint32) (file fs.FileHandle, fuseFlags uint32, code syscall.Errno) {
	return &androidFile{
		node: n,
	}, 0, 0
}

var _ = (fs.NodeSetattrer)((*androidNode)(nil))

func (n *androidNode) Setattr(ctx context.Context, file fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) (code syscall.Errno) {
	if size, ok := in.GetSize(); ok {
		if f, ok := file.(*androidFile); ok {
			if errno := f.flushPending(); errno != 0 {
				return errno
			}
		}
		w := n.write
		if !n.startEdit() {
			return syscall.EIO
		}
		if err := n.fs.dev.AndroidTruncate(n.Handle(), int64(size)); err != nil {
			log.Println("AndroidTruncate failed:", err)
			return syscall.EIO
		}
		n.Size = int64(size)
		n.devSize = int64(size)

		if !w {
			if !n.endEdit() {
				return syscall.EIO
			}
		}
	}
	return n.Getattr(ctx, file, out)
}

var _ = mtpNode((*androidNode)(nil))

// writeBufSize is the largest partial object sent in one transaction. Every transaction has a
// fixed cost, so consecutive small writes are gathered up to this size before being sent.
const writeBufSize = 1 << 20

type androidFile struct {
	fs.FileHandle
	node *androidNode

	// pending holds consecutive writes starting at pendingOff, not yet sent to the device.
	pending    []byte
	pendingOff int64
}

func (f *androidFile) flushPending() syscall.Errno {
	if len(f.pending) == 0 {
		return 0
	}
	n := f.node
	if !n.startEdit() {
		return syscall.EIO
	}
	// Android refuses a partial write that starts past the end of the object, and does so
	// without reading its data, which leaves the transfer hanging until the device is
	// unplugged. macOS can deliver the end of a large file first, so grow the object to the
	// write offset beforehand; the earlier writes fill the gap when they arrive.
	if f.pendingOff > n.devSize {
		if err := n.fs.dev.AndroidTruncate(n.Handle(), f.pendingOff); err != nil {
			log.Println("AndroidTruncate failed:", err)
			f.pending = f.pending[:0]
			return syscall.EIO
		}
		n.devSize = f.pendingOff
	}
	err := n.fs.dev.AndroidSendPartialObject(n.Handle(), f.pendingOff, uint32(len(f.pending)), bytes.NewBuffer(f.pending))
	if end := f.pendingOff + int64(len(f.pending)); err == nil && end > n.devSize {
		n.devSize = end
	}
	f.pending = f.pending[:0]
	if err != nil {
		log.Println("AndroidSendPartialObject failed:", err)
		return syscall.EIO
	}
	return 0
}

var _ = (fs.FileReader)((*androidFile)(nil))

func (f *androidFile) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if errno := f.flushPending(); errno != 0 {
		return nil, errno
	}
	if off > f.node.Size {
		// ENXIO = no such address.
		return nil, syscall.Errno(int(syscall.ENXIO))
	}

	if off+int64(len(dest)) > f.node.Size {
		dest = dest[:f.node.Size-off]
	}
	b := bytes.NewBuffer(dest[:0])
	err := f.node.fs.dev.AndroidGetPartialObject64(f.node.Handle(), b, off, uint32(len(dest)))
	if err != nil {
		log.Println("AndroidGetPartialObject64 failed:", err)
		return nil, syscall.EIO
	}

	return fuse.ReadResultData(dest[:b.Len()]), 0
}

func (f *androidFile) String() string {
	return fmt.Sprintf("androidFile h=0x%x", f.node.Handle())
}

var _ = (fs.FileWriter)((*androidFile)(nil))

func (f *androidFile) Write(ctx context.Context, dest []byte, off int64) (written uint32, status syscall.Errno) {
	if !f.node.startEdit() {
		return 0, syscall.EIO
	}
	contiguous := off == f.pendingOff+int64(len(f.pending))
	if len(f.pending) > 0 && (!contiguous || len(f.pending)+len(dest) > writeBufSize) {
		if errno := f.flushPending(); errno != 0 {
			return 0, errno
		}
	}
	if len(f.pending) == 0 {
		f.pendingOff = off
	}
	f.pending = append(f.pending, dest...)
	f.node.byteCount += int64(len(dest))
	if end := off + int64(len(dest)); end > f.node.Size {
		f.node.Size = end
	}
	if len(f.pending) >= writeBufSize {
		if errno := f.flushPending(); errno != 0 {
			return 0, errno
		}
	}
	return uint32(len(dest)), 0
}

var _ = (fs.FileFlusher)((*androidFile)(nil))

func (f *androidFile) Flush(ctx context.Context) syscall.Errno {
	if errno := f.flushPending(); errno != 0 {
		f.node.endEdit()
		return errno
	}
	if !f.node.endEdit() {
		return syscall.EIO
	}
	return 0
}

var _ = (fs.FileFsyncer)((*androidFile)(nil))

func (f *androidFile) Fsync(ctx context.Context, flags uint32) syscall.Errno {
	return f.flushPending()
}

var _ = (fs.FileReleaser)((*androidFile)(nil))

// Release sends writes that macOS delivered after the last Flush, which would otherwise be lost.
func (f *androidFile) Release(ctx context.Context) syscall.Errno {
	if len(f.pending) == 0 {
		return 0
	}
	errno := f.flushPending()
	if !f.node.endEdit() && errno == 0 {
		errno = syscall.EIO
	}
	return errno
}
