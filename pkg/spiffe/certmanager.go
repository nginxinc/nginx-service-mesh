// Package spiffe contains code related to spiffe identity management
package spiffe

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrCertTimeout occurs when CertManager does not receive
// the initial trust bundle before the configured timeout.
var ErrCertTimeout = errors.New("timed out waiting for trust bundle")

//go:generate counterfeiter -generate

// Reloader knows how to reload a process
//
//counterfeiter:generate ./. Reloader
type Reloader interface {
	// Reload reloads a process
	Reload() error
}

// CertManager writes SVID certificates and keys to disk.
type CertManager struct {
	svidWriter  SVIDWriter
	certFetcher CertFetcher
	reloader    Reloader
	ErrCh       chan error
	timeout     time.Duration
}

// NewCertManager returns a new instance of the CertManager.
func NewCertManager(svidWriter SVIDWriter, fetcher CertFetcher, timeout time.Duration) *CertManager {
	return &CertManager{
		ErrCh:       make(chan error),
		svidWriter:  svidWriter,
		certFetcher: fetcher,
		timeout:     timeout,
	}
}

// NewCertManagerWithReloader returns a new instance of the CertManager.
func NewCertManagerWithReloader(
	reloader Reloader,
	svidWriter SVIDWriter,
	fetcher CertFetcher,
	timeout time.Duration,
) *CertManager {
	rel := NewCertManager(svidWriter, fetcher, timeout)
	rel.reloader = reloader

	return rel
}

// reloads IFF reloader not nil.
func (c *CertManager) reload() {
	if c.reloader != nil {
		err := c.reloader.Reload()
		if err != nil {
			c.ErrCh <- err
		}
	}
}

// Run is the run loop for the certmanager.
// Starts the certFetcher and waits for certs or an unrecoverable error.
func (c *CertManager) Run(ctx context.Context) error {
	certStream, errStream, err := c.certFetcher.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting cert fetcher: %w", err)
	}

	// Wait for initial trust bundle
	select {
	case certs := <-certStream:
		err = c.svidWriter.Write(certs)
		if err != nil {
			return fmt.Errorf("error writing certificates: %w", err)
		}
		c.reload()
	case err = <-errStream:
		return fmt.Errorf("error waiting for initial trust bundle: %w", err)
	case <-time.After(c.timeout):
		return ErrCertTimeout
	case <-ctx.Done():
		return ctx.Err()
	}

	// Now start goroutine to wait for updates
	go func() {
		for {
			select {
			case err = <-errStream:
				c.ErrCh <- err

				return
			case certs := <-certStream:
				err := c.svidWriter.Write(certs)
				if err != nil {
					c.ErrCh <- err

					return
				}
				c.reload()
			case <-ctx.Done():
				c.ErrCh <- ctx.Err()
			}
		}
	}()

	return nil
}

// Stop stops the internal certFetcher.
func (c *CertManager) Stop() error {
	return c.certFetcher.Stop()
}
