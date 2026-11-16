package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/hanwen/go-mtpfs/fs"
	"github.com/hanwen/go-mtpfs/mtp"
	"github.com/hanwen/usb"
)

// Délais USB en millisecondes : court à la connexion pour qu'un téléphone verrouillé échoue vite,
// long ensuite car un transfert qui expire fige le téléphone jusqu'au débranchement.
const (
	connectTimeout = 3000
	ioTimeout      = 30000
)

const maxWrite = 1 << 20

var errLocked = errors.New("le téléphone ne répond pas, déverrouille le")

type phone struct {
	dev    *mtp.Device
	server *fuse.Server
	done   chan struct{}
}

func mountPhone(ctx *usb.Context, dir string) (*phone, error) {
	devs, err := mtp.FindDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, errors.New("aucun appareil MTP")
	}
	for _, d := range devs[1:] {
		d.Done()
	}
	dev := devs[0]
	debug := os.Getenv("ANDROID_MTP_DEBUG") != ""
	fail := func(err error) (*phone, error) {
		dev.Close()
		dev.Done()
		return nil, err
	}

	dev.Timeout = connectTimeout
	dev.MTPDebug = debug
	if err := dev.Configure(); err != nil {
		return fail(fmt.Errorf("%w (%v)", errLocked, err))
	}
	dev.Timeout = ioTimeout
	sids, err := fs.SelectStorages(dev, "")
	if err != nil {
		return fail(err)
	}
	if len(sids) == 0 {
		return fail(errLocked)
	}
	root, err := fs.NewDeviceFSRoot(dev, sids, fs.DeviceFsOptions{RemovableVFat: true, Android: true})
	if err != nil {
		return fail(err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	sec := time.Second
	opts := &fusefs.Options{
		MountOptions: fuse.MountOptions{
			SingleThreaded: true,
			MaxWrite:       maxWrite,
			Debug:          debug,
			FsName:         "android",
			Name:           "mtp",
			Options:        []string{"volname=Android", "local", "noappledouble"},
		},
		UID:          uint32(os.Getuid()),
		GID:          uint32(os.Getgid()),
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}
	server, err := fuse.NewServer(macRawFS{fusefs.NewNodeFS(root, opts)}, dir, &opts.MountOptions)
	if err != nil {
		return fail(err)
	}
	go server.Serve()
	if err := server.WaitMount(); err != nil {
		return fail(err)
	}

	p := &phone{dev: dev, server: server, done: make(chan struct{})}
	go func() {
		server.Wait()
		close(p.done)
	}()
	return p, nil
}

func (p *phone) ejected() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

func (p *phone) unmount(dir string) {
	if p.server.Unmount() != nil {
		forceUnmount(dir)
	}
	select {
	case <-p.done:
	case <-time.After(5 * time.Second):
	}
	p.release()
}

func (p *phone) release() {
	p.dev.Close()
	p.dev.Done()
}
