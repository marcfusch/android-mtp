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

// Délai de chaque transfert USB, en millisecondes. Court, pour qu'un téléphone verrouillé ou
// figé renvoie une erreur au lieu de bloquer.
const usbTimeout = 3000

var errLocked = errors.New("le téléphone ne répond pas, déverrouille le")

// phone est un téléphone monté : la session MTP et le serveur FUSE qui la sert.
type phone struct {
	dev    *mtp.Device
	server *fuse.Server
	done   chan struct{}
}

// mountPhone ouvre la session MTP et monte le téléphone dans dir. Configure ferme une session
// restée ouverte et réinitialise l'USB si l'ouverture échoue.
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
	fail := func(err error) (*phone, error) {
		dev.Close()
		dev.Done()
		return nil, err
	}

	dev.Timeout = usbTimeout
	if err := dev.Configure(); err != nil {
		return fail(fmt.Errorf("%w (%v)", errLocked, err))
	}
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
	server, err := fusefs.Mount(dir, root, &fusefs.Options{
		MountOptions: fuse.MountOptions{
			SingleThreaded: true,
			FsName:         "android",
			Name:           "mtp",
			Options:        []string{"volname=Android", "local", "noappledouble", "noapplexattr"},
		},
		UID:          uint32(os.Getuid()),
		GID:          uint32(os.Getgid()),
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	})
	if err != nil {
		return fail(err)
	}

	p := &phone{dev: dev, server: server, done: make(chan struct{})}
	go func() {
		server.Wait()
		close(p.done)
	}()
	return p, nil
}

// ejected dit si le montage a disparu sans que l'agent l'ait démonté : éjection Finder ou android off.
func (p *phone) ejected() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// unmount démonte, de force si le montage est occupé, puis ferme la session MTP.
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

// release ferme la session MTP une fois le serveur FUSE arrêté.
func (p *phone) release() {
	p.dev.Close()
	p.dev.Done()
}
