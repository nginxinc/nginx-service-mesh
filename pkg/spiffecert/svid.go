/*
Package spiffecert ...

	When moving away from on-disk certificate configurations
	and towards in memory certificate configurations some logic
	in the disksvidwriter package was desired with much change
	to its interface. Since that package only intends to provide
	a means of writing certs to disk this package was made to use
	in its place.
*/
package spiffecert

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nginxinc/nginx-service-mesh/pkg/taskqueue"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const (
	CABundleFileMode = os.FileMode(0o644) //nolint
)

// ErrNoCertificates occurs when a workloadapi.X509Context contains
// no certificates, but is expected to by a parsing function.
var ErrNoCertificates = errors.New("no certificates in svid response")

/* CABundleManager manages SPIRE events and CA Bundles. */
type CABundleManager struct {
	TaskQueue          *taskqueue.TaskQueue
	CABundleFilepath   string
	latestCABundleHash []byte
	currentCert        []byte
	currentKey         []byte
	certLock           sync.RWMutex
}

/*
Write Implements svidWriter interface.

	Writes CA Bundle to disk if none have
	been written yet. Otherwise enqueues
	a spire event in the taskqueue.
*/
func (h *CABundleManager) Write(svidResponse *workloadapi.X509Context) error {
	// handle initial bundle download
	if h.latestCABundleHash == nil {
		caBytes, _, err := h.CABundleBytesFromSVIDResponse(svidResponse)
		if err != nil {
			return fmt.Errorf("couldnt marshal CA bundle: %w", err)
		}

		if err = os.WriteFile(
			h.CABundleFilepath,
			caBytes,
			CABundleFileMode,
		); err != nil {
			return fmt.Errorf("couldnt write CA bundle: %w", err)
		}

		// we need copies of cert and key for NATS' sake
		_, _, err = h.CertKeyBytesFromSVIDResponse(svidResponse)
		if err != nil {
			return err
		}
	}
	// queue this anyways because certs and keys
	// still need to go into the correct KV stores
	h.TaskQueue.Enqueue("SPIRE", svidResponse)

	return nil
}

/*
TestAndUpdateCABundle Takes CA Bundle bytes

	and tests if they are equal to the previous
	CA Bundle. Updates internal hash if so.
	Returns if the bundle has changed.
*/
func (h *CABundleManager) TestAndUpdateCABundle(caBundle []byte) bool {
	caSha := sha256.New()
	caSha.Write(caBundle)
	caBundleHash := caSha.Sum(nil)
	isNew := !bytes.Equal(caBundleHash, h.latestCABundleHash)
	if isNew {
		h.latestCABundleHash = caBundleHash
	}

	return isNew
}

// WaitForCABundle Waits given seconds for CABundle to be written.
func (h *CABundleManager) WaitForCABundle(maxSeconds int) error {
	for i := 1; i < maxSeconds; i++ {
		time.Sleep(time.Second)
		if _, err := os.Stat(h.CABundleFilepath); err == nil {
			return nil
		}
	}

	return ErrCertTimeout
}

/*
CABundleBytesFromSVIDResponse Extracts

	CA Bundle bytes from svidResponse. Also
	tests hash value and updates internal
	hash if it has been updates. Returns CA
	Bytes,whether or not the CA Bundle has
	changed, and possibly a marshal error.
*/
func (h *CABundleManager) CABundleBytesFromSVIDResponse(svidResponse *workloadapi.X509Context) ([]byte, bool, error) {
	trustDomain := svidResponse.DefaultSVID().ID.TrustDomain()
	bundle, err := svidResponse.Bundles.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing CA bundle from svid response: %w", err)
	}
	b, e := bundle.Marshal()

	return b, h.TestAndUpdateCABundle(b), e
}

/*
SerialNumberFromSVIDResponse Extracts

	the default SVID certificate's serial
	number from a given SVID Response
*/
func (h *CABundleManager) SerialNumberFromSVIDResponse(svidResponse *workloadapi.X509Context) ([]byte, error) {
	svid := svidResponse.DefaultSVID()
	if len(svid.Certificates) < 1 {
		return nil, ErrNoCertificates
	}
	serial := svid.Certificates[0].SerialNumber

	return []byte(serial.String()), nil
}

/*
CertKeyBytesFromSVIDResponse Extracts

	cert and key from svidResponse. Also
	updates internal copy of cert and key.
*/
func (h *CABundleManager) CertKeyBytesFromSVIDResponse(svidResponse *workloadapi.X509Context) ([]byte, []byte, error) {
	cert, key, err := svidResponse.DefaultSVID().Marshal()
	if err != nil {
		return nil, nil, err
	}

	h.certLock.Lock()
	defer h.certLock.Unlock()
	h.currentCert = cert
	h.currentKey = key

	// ADD NO ADDITIONAL CALLS HERE

	return cert, key, nil
}

/*
NewCertificateGetter returns a TLS Config GetCertificate function that

	fetches certificates from the CABundleManager.
*/
func (h *CABundleManager) NewCertificateGetter() func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		h.certLock.RLock()
		defer h.certLock.RUnlock()
		cert, err := tls.X509KeyPair(h.currentCert, h.currentKey)

		// ADD NO ADDITIONAL CALLS HERE

		return &cert, err
	}
}
