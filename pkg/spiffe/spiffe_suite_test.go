package spiffe_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSpiffeCert(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpiffeCert Test Suite")
}

var (
	testCACert                   []*x509.Certificate
	testCert                     []*x509.Certificate
	testKey                      crypto.Signer
	rootPEM, certPEM, privateKey string
	errDecodeCert                = errors.New("failed to decode certificate")
	errDecodeKey                 = errors.New("failed to decode key")
	errFakeWatchFail             = errors.New("fake watch error")
	errFakeWriteFail             = errors.New("fake write error")
	errFakeFetchFail             = errors.New("fake fetch error")
	host                         = "test.host.com"
	validFrom                    = time.Now()
	validFor                     = 365 * 24 * time.Hour
)

var _ = BeforeSuite(func() {
	var err error
	c, k, r := getTestCACertKey()
	rootPEM = string(r)
	certPEM = string(c)
	privateKey = string(k)
	testCACert, err = parseCert(rootPEM)
	Expect(err).ToNot(HaveOccurred())
	testCert, err = parseCert(certPEM)
	Expect(err).ToNot(HaveOccurred())
	testKey, err = parseKey(privateKey)
	Expect(err).ToNot(HaveOccurred())
})

func makeSVIDResponse(cert, key, caBytes []byte) *workloadapi.X509Context {
	defaultSVID, err := x509svid.Parse(cert, key)
	Expect(err).ToNot(HaveOccurred())

	trust, err := spiffeid.TrustDomainFromString(host)
	Expect(err).ToNot(HaveOccurred())
	bundle, err := x509bundle.Parse(trust, caBytes)
	Expect(err).ToNot(HaveOccurred())
	bundleSet := x509bundle.NewSet(bundle)

	return &workloadapi.X509Context{
		SVIDs:   []*x509svid.SVID{defaultSVID},
		Bundles: bundleSet,
	}
}

func stubContextUpdateCall(ctx context.Context, watcher workloadapi.X509ContextWatcher) error {
	// call watcher one time then block on ctx.Done()
	watcher.OnX509ContextUpdate(&workloadapi.X509Context{
		SVIDs: []*x509svid.SVID{
			{
				ID: spiffeid.ID{},
			},
		},
		Bundles: nil,
	})
	<-ctx.Done()

	return status.Error(codes.Canceled, "context canceled")
}

func getTestCACertKey() ([]byte, []byte, []byte) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	rootTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   "Root CA",
		},
		NotBefore:             validFrom,
		NotAfter:              validFrom.Add(validFor),
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey)
	Expect(err).ToNot(HaveOccurred())
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())
	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	Expect(err).ToNot(HaveOccurred())
	uri, err := url.Parse("spiffe://" + host)
	Expect(err).ToNot(HaveOccurred())
	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   "test_cert_1",
		},
		URIs:                  []*url.URL{uri},
		DNSNames:              []string{host},
		NotBefore:             validFrom,
		NotAfter:              validFrom.Add(validFor),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &rootTemplate, &key.PublicKey, rootKey)
	Expect(err).ToNot(HaveOccurred())

	cert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	keyB, err := x509.MarshalPKCS8PrivateKey(key)
	Expect(err).ToNot(HaveOccurred())
	privKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyB,
	})

	caCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	return cert, privKey, caCert
}

func parseCert(cert string) ([]*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		return nil, errDecodeCert
	}
	certs, err := x509.ParseCertificates(block.Bytes)
	if err != nil {
		return nil, err
	}

	return certs, nil
}

func parseKey(key string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, errDecodeKey
	}

	pkey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	pk := pkey.(*rsa.PrivateKey) //nolint:forcetypeassert // we know this is an RSA key cause we made it

	return pk, err
}
