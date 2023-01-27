// Package spiffe contains code related to spiffe identity management
package spiffe

import (
	"context"
	"fmt"
	"log"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errNoClient = fmt.Errorf("failed to start cert fetcher: nil client")

//go:generate counterfeiter -generate

// Client wraps the workloadapi.Client
//
//counterfeiter:generate ./. Client
type Client interface {
	WatchX509Context(context.Context, workloadapi.X509ContextWatcher) error
	Close() error
}

// implements workloadapi.x509ContextWatcher.
type watcher struct {
	certFetcher X509CertFetcher
}

func newWatcher(fetcher X509CertFetcher) *watcher {
	return &watcher{certFetcher: fetcher}
}

// OnX509ContextUpdate is called when a new X.509 Context is fetched from the SPIFFE Workload API.
// The X.509 context is placed on the cert fetcher's cert channel.
func (w *watcher) OnX509ContextUpdate(svidResp *workloadapi.X509Context) {
	log.Printf("SVID updated for spiffeID: %q\n", svidResp.DefaultSVID().ID)
	w.certFetcher.CertCh <- svidResp
}

// OnX509WatchError is called when there is an error watching the X.509 Context's from the SPIFFE Workload API.
func (w *watcher) OnX509ContextWatchError(err error) {
	msg := "For more information check the logs of the Spire agents and server."
	switch status.Code(err) { //nolint:exhaustive // uses a default
	case codes.Unavailable:
		log.Printf("X509SVIDClient cannot connect to the Spire agent: %v. %s", err, msg)
	case codes.PermissionDenied:
		log.Printf("X509SVIDClient still waiting for certificates: %v. %s", err, msg)
	case codes.Canceled:
		return
	default:
		log.Printf("X509SVIDClient error: %v. %s", err, msg)
	}
}

// CertFetcher fetches certificates
//
//counterfeiter:generate ./. CertFetcher
type CertFetcher interface {
	// Start starts fetching X.509 certificates.
	// It returns an error if it fails to start.
	// Otherwise, certificates are written to the X509Context channel
	// and if there is an unrecoverable error it is written to the error channel.
	Start(context.Context) (<-chan *workloadapi.X509Context, <-chan error, error)
	// Stop closes the connection with the SPIFFE Workload API Client.
	Stop() error
}

// X509CertFetcher fetches certs from the X509 SPIFFE Workload API.
type X509CertFetcher struct {
	client     Client
	WatchErrCh chan error
	CertCh     chan *workloadapi.X509Context
	spireAddr  string
}

// NewX509CertFetcher creates a new instance of CertFetcher.
func NewX509CertFetcher(spireAddr string, client Client) (*X509CertFetcher, error) {
	if client == nil {
		var err error
		ctx := context.Background()
		client, err = workloadapi.New(ctx, workloadapi.WithAddr("unix://"+spireAddr))
		if err != nil {
			return nil, err
		}
	}

	return &X509CertFetcher{
		WatchErrCh: make(chan error),
		CertCh:     make(chan *workloadapi.X509Context),
		spireAddr:  spireAddr,
		client:     client,
	}, nil
}

// Start creates a SPIFFE Workload API Client. If the client cannot be created an error is returned.
// Otherwise, a goroutine is kicked off that watches for new X.509 Contexts over the Workload API.
// If a fatal error occurs while watching for X.509 Contexts it is written to the WatchErrCh channel.
//
//nolint:gocritic
func (c *X509CertFetcher) Start(ctx context.Context) (<-chan *workloadapi.X509Context, <-chan error, error) {
	if c.client == nil {
		return nil, nil, errNoClient
	}

	watcher := newWatcher(*c)
	go func() {
		defer func() {
			if err := c.client.Close(); err != nil && status.Code(err) != codes.Canceled {
				log.Println("error closing SPIFFE Workload API Client: ", err)
			}
		}()
		err := c.client.WatchX509Context(ctx, watcher)
		if err != nil && status.Code(err) != codes.Canceled {
			c.WatchErrCh <- err
		}
	}()

	return c.CertCh, c.WatchErrCh, nil
}

// Stop closes the connection with the SPIFFE Workload API Client.
func (c *X509CertFetcher) Stop() error {
	return c.client.Close()
}
