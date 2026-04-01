package sysboxutil

import (
	"context"
	"net"
	"os"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/envbox/xunix"
)

const (
	ManagerSocketPath = "/run/sysbox/sysmgr.sock"
	FSSocketPath      = "/run/sysbox/sysfs.sock"
)

// WaitForManager waits for the sysbox-mgr to startup.
func WaitForManager(ctx context.Context) error {
	fs := xunix.GetFS(ctx)

	_, err := fs.Stat(ManagerSocketPath)
	if err == nil {
		return nil
	}

	const (
		period = time.Second
	)

	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			_, err := fs.Stat(ManagerSocketPath)
			if err != nil {
				if !xerrors.Is(err, os.ErrNotExist) {
					return xerrors.Errorf("unexpected stat err %s: %w", ManagerSocketPath, err)
				}
				continue
			}
			return nil
		}
	}
}

// WaitForFS waits for the sysbox-fs socket to be available and accepting
// connections.
func WaitForFS(ctx context.Context) error {
	fs := xunix.GetFS(ctx)

	const period = time.Second

	t := time.NewTicker(period)
	defer t.Stop()

	for {
		stat, err := fs.Stat(FSSocketPath)
		if err == nil {
			if stat.Mode()&os.ModeSocket == 0 {
				return xerrors.Errorf("%s exists but is not a socket", FSSocketPath)
			}

			// Ensure the socket is accepting connections before starting
			// sysbox-mgr, which depends on sysbox-fs being ready.
			conn, err := (&net.Dialer{Timeout: period}).DialContext(ctx, "unix", FSSocketPath)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		} else if !xerrors.Is(err, os.ErrNotExist) {
			return xerrors.Errorf("unexpected stat err %s: %w", FSSocketPath, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}
