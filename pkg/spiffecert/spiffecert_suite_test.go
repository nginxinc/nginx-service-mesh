package spiffecert_test

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

	. "github.com/onsi/ginkgo"
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
	testCACert       []*x509.Certificate
	testCert         []*x509.Certificate
	testKey          crypto.Signer
	errDecodeCert    = errors.New("failed to decode certificate")
	errDecodeKey     = errors.New("failed to decode key")
	errFakeWatchFail = errors.New("fake watch error")
	errFakeWriteFail = errors.New("fake write error")
	errFakeFetchFail = errors.New("fake fetch error")
	host             = "test.host.com"
	validFrom        = time.Now()
	validFor         = 365 * 24 * time.Hour
)

var _ = BeforeSuite(func() {
	var err error
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

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

const rootPEM = `
-----BEGIN CERTIFICATE-----
MIIDQDCCAigCCQCpN7/BKC8RATANBgkqhkiG9w0BAQUFADBiMQswCQYDVQQGEwJV
UzETMBEGA1UECAwKV2FzaGluZ3RvbjEQMA4GA1UEBwwHU2VhdHRsZTEOMAwGA1UE
CgwFVGVzdCAxDTALBgNVBAsMBFRlc3QxDTALBgNVBAMMBFRlc3QwHhcNMjEwNDA0
MDAzMTA0WhcNMjEwNDA1MDAzMTA0WjBiMQswCQYDVQQGEwJVUzETMBEGA1UECAwK
V2FzaGluZ3RvbjEQMA4GA1UEBwwHU2VhdHRsZTEOMAwGA1UECgwFVGVzdCAxDTAL
BgNVBAsMBFRlc3QxDTALBgNVBAMMBFRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQCyxhkREjRAyT3z7yg+Q7gZXlvrJtguk87a2BhyQ3Ld60Ep/gvj
kCIz0X+m8Dvq2u6wTSDBGLTDY3tbLYMnzWTLJmL9PxzmED+gg+VSKloxSNL0CMJs
/C2pw1C7YpeQ2oJGqLofgnPyr/O1TJ+jjZ4hjh8FfEuyp92v60dcpZbwvJLvI/Q0
OoleyIyDkq91pyiofSgJqN4NkJsd8xHjBxMcskQ08Ep06nniQDwwfwIQwtioTHJb
0rSC1wotYXJwG+JnrreZZt6PDotKuWBxcCjrzUYCkeEY9bPYJU6JkY85ndQf9BFL
d9eiy8iWu98KTOah0XSCE6gA0FPMrxSBPz6pAgMBAAEwDQYJKoZIhvcNAQEFBQAD
ggEBAFQ3rJ671aExC++43/8ytiKIvGoQTTsSZKm8U1xaYdsVlZRmi9bdW1nfOld7
xxRREhhvGMSm/6FxgfCLwzvJH2buelFxTZXrziiNN/YOQxteIHSSr9nE3YNdPKXB
5iewUDw6J+egBHkabmPdZ7+Q7M7TkJ4K0O6adiF7yq8QlRBAsuZE10riSq4SnQzY
9gAntE85Py2AFO+Blfa5H1ojQ5UicGZyXoQXIf0KBppfNFyfJPECETU9Lp2Zsnpm
2UZl9n9i9vJRkz5Ww1Wgon1P9TRMx8Lay1D8AkqsHdw9Ese5kTEZscSm3MhiVS+H
kvBFHAh5B52Bx1mJMBabSU2lW/k=
-----END CERTIFICATE-----
`

const certPEM = `
-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIJAIQDKzcb/uzpMA0GCSqGSIb3DQEBCwUAMGIxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApXYXNoaW5ndG9uMRAwDgYDVQQHDAdTZWF0dGxlMQ4w
DAYDVQQKDAVUZXN0IDENMAsGA1UECwwEVGVzdDENMAsGA1UEAwwEVGVzdDAeFw0y
MTA0MDQwMDMxMDRaFw0yMTA0MDUwMDMxMDRaMGcxCzAJBgNVBAYTAlVTMRMwEQYD
VQQIDApXYXNoaW5ndG9uMRAwDgYDVQQHDAdTZWF0dGxlMRMwEQYDVQQKDApUZXN0
LCBJbmMuMQ0wCwYDVQQLDARUZXN0MQ0wCwYDVQQDDARUZXN0MIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwylGy82Zh55vre4VhvmzwZB2wdtD3mf77eRx
0hhIT6rviC3OG273FIu70byKVj3G0q1b1APvTS7sDILWBxBAL0+v97IR/vUoYL0+
BpS5jVTvg+HiCCL+QweXhCC46yBdeGvqldF/lEnwDoOiQArnHCp8EudKEASVCYha
iPRRjA6nnkM952zk2cc1wfo/lg8K/qKYZpcZ36TowLZCeG37RIp5upzEGS4vfAxy
+ZL+WQu5GlDszGgNeLF2VcWN0kqWFzM2dlQ08xO782I5DLhqpxLN6m6EzYaU6qHj
u/LgReXFEWaoZWC7pjk0/5O89vVOnkTXLil7U0MAJZytlRlf4wIDAQABox8wHTAb
BgNVHREEFDASghB0ZXN0LmV4YW1wbGUuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQBN
LZu2xPEoSgEaG+3OupYXcjCNA59b3pOIGTv4bSlWUbudaJmwB4Fe5SE8xe76XtLU
HxVcIsFrbuzXltIWIci2RW6YD239+OL3N25LoGhffhmw191w9fqLLJsADByD+GZW
IfsAp18DQo+mJyfkZMcJPB8Kpe5vWqrKOHxkTOFNlQxzqkDcz07NNJFYSaAzHp/c
58aCHqmoiReRjfZtZmRlBWCC0Y34+NxXuMXLSCWflConv+ToKOBTaiQPvgtceQs5
hcoDtyhZUNqClOlLu9jjEYfnuXKD5UlSzZ5zYmeDSmaHyguwGqnT1+77DAuHEpM6
gxIl1RZlXU7Inwktj4Ug
-----END CERTIFICATE-----
`

const privateKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAwylGy82Zh55vre4VhvmzwZB2wdtD3mf77eRx0hhIT6rviC3O
G273FIu70byKVj3G0q1b1APvTS7sDILWBxBAL0+v97IR/vUoYL0+BpS5jVTvg+Hi
CCL+QweXhCC46yBdeGvqldF/lEnwDoOiQArnHCp8EudKEASVCYhaiPRRjA6nnkM9
52zk2cc1wfo/lg8K/qKYZpcZ36TowLZCeG37RIp5upzEGS4vfAxy+ZL+WQu5GlDs
zGgNeLF2VcWN0kqWFzM2dlQ08xO782I5DLhqpxLN6m6EzYaU6qHju/LgReXFEWao
ZWC7pjk0/5O89vVOnkTXLil7U0MAJZytlRlf4wIDAQABAoIBACBMTnkgF46IO/dO
9aUW4hbgr6a5gOvnzZu7ONMKTb1Rjr68xeVoYd+2mGjHiSVop+Tp586YsBvX7hzL
8lvM5rJtv7OAdtX5AKux5ff02Rh4vALZeSzbjbTOJtcpCzFuc8mgInyU5UZHTkE4
q7tpkoHozgva1zj7aSbguAr+VBiXcgXNgnVo4Tp+SexPAo5TDWflHBQ35UHAS1to
G+cUVynhUdpWyDPi1NiXf5+CmR++EjW09QPu5hoQZCukdq0fgbtPsH5c5o7e8TZs
UKMF3pfIyVLv9HljMXzlZzgjYn5a/W6SFz6YMYNBKjStMX6fMD37wSlDlq2616U6
tU0o/TECgYEA8/C0Z4mooQCAEpaQPr5gBVCXSMjse+Vy5k8UtN9kYRQAsV9L+7H7
R1yBIiHN7ZXrrL7SeylDVRtMAGgm0SV3XV/0WheSkGrNhQ7t5oyU7zO0IZHdyoiw
eY/Ec7GGUjoawFF6XIAiJnZy1ANAWTg/934/LmLEhTNuNdMwzNLnmO0CgYEAzM86
JJzZllVAzUmCe6RVGbhexb30ymgmRrYXxiuPbTJeK5etb6OxZwcJNcUSI3KoSXKY
9PG+eGy6DIQBJ5ty5oKfx7FJk0KcCJkTtovhi0Xgp0ES15l8NI+e4vOoBRtJ9iBI
TlWDsLiQGrL7NEJ+CpNxEvACQRqmWqFLe1zJ0g8CgYEAii2n0xpcBc8lvOHClXf7
JiePenAt3MSNAD59aTM9RewxtEdZ4BniT3rrvuzNHC6XEAQLcC5gcJ4EwBo/GquR
YLgQztOZduq4vg1F3xl058Yu2/EnZClnZYR1cF93ya4WJyhAGpOORKFFzCiHU8KU
IVpG6bySuyz12dFmTC+PdsUCgYEAwDJTsNoUgrQ8XKl4LolXZwySu2R4XJ2CFed5
xflI3kNfBe+PzW1C5JlAtlnanLNTY6GMEojtolr9+RLDdqS1HcZFJQOlNPFUNelZ
C3yXSrhniu1RPkwFt9lzVC0tZqVmMfe3gvNS4rtAWB3QCQnA+DHG8euTYf8dT31/
tSOtLVcCgYEAqY9sKJCWKKYoBB+nO+EFd0TFTUZ5wvmP0sU166phaskuK4ScAnuk
m6giumtbP/p8rjKdOh2vt4Zu7W2+SmPtDd58nmSF1yMIS/wewwvPIbE6ikR1ghU9
Fe08lUhkgBbvxa7G6RYD83mC19D4tIJ1htFYQgAouclbz5L8i0+MVOk=
-----END RSA PRIVATE KEY-----
`
