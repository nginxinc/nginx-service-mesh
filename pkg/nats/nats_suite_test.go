package nats_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	testserver "github.com/nats-io/nats-server/v2/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	testFilename string
	testDataDir  string
	natsSession  *NATSSession
)

var natsTLSConf = `
pid_file: "%[1]s"
listen: localhost:%[2]d
tls: {
  ca_file: "%[3]s/ca_pem"
  cert_file: "%[3]s/server-cert_pem"
  key_file: "%[3]s/server-key_pem"
  verify: true
}
`

func TestNATSSuite(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)

	var err error
	testDataDir, err = os.MkdirTemp(".", "testdata-*")
	Expect(err).ToNot(HaveOccurred())

	RunSpecs(t, "NATS Test Suite")
}

var _ = BeforeSuite(func() {
	// create test file
	testFile, err := os.CreateTemp("", "test.pem")
	Expect(err).ToNot(HaveOccurred())
	testFilename = testFile.Name()

	natsSession = NewSecureNATSSession(testDataDir)
	natsSession.Start()
})

var _ = AfterSuite(func() {
	// remove test file
	err := os.Remove(testFilename)
	Expect(err).ToNot(HaveOccurred())

	// kill local nats server and cleanup artifacts
	natsSession.Cleanup()
	Expect(os.RemoveAll(testDataDir)).To(Succeed())
})

// NATSSession is a wrapper around a nats-server
// that can be used in tests.
type NATSSession struct {
	pidFile    *os.File
	configFile *os.File
	server     *server.Server
	certDir    string
	Port       int
}

// NewSecureNATSSession creates a NATs session.
// certDir is the absolute path to the where the certificates are stored.
func NewSecureNATSSession(certDir string) *NATSSession {
	path, err := filepath.Abs(certDir)
	Expect(err).ToNot(HaveOccurred())

	session := &NATSSession{
		certDir: path,
	}

	session.bootstrap()
	session.writeCerts()
	session.writeTLSConfigFile()

	return session
}

// bootstrap creates pid and conf files and assigns random open port.
func (ns *NATSSession) bootstrap() {
	ns.pidFile = createTempFile("nats.pid")
	ns.configFile = createTempFile("nats.conf")
	ns.Port = pickPort()
}

// Create and write certs needed for NATS client and server.
func (ns *NATSSession) writeCerts() {
	caPrivKey, caCert, ca, err := newTestCA() //nolint:varnamelen // ca is a good name
	Expect(err).ToNot(HaveOccurred())

	serverPrivKey, serverCert, err := newTestCertFromCA(caPrivKey, &ca)
	Expect(err).ToNot(HaveOccurred())

	clientPrivKey, clientCert, err := newTestCertFromCA(caPrivKey, &ca)
	Expect(err).ToNot(HaveOccurred())

	buf, err := encodeCert(caCert)
	Expect(err).ToNot(HaveOccurred())
	Expect(pemToFile(buf, ns.certDir, "ca_pem")).To(Succeed())

	buf, err = encodePrivateKey(serverPrivKey)
	Expect(err).ToNot(HaveOccurred())
	Expect(pemToFile(buf, ns.certDir, "server-key_pem")).To(Succeed())

	buf, err = encodeCert(serverCert)
	Expect(err).ToNot(HaveOccurred())
	Expect(pemToFile(buf, ns.certDir, "server-cert_pem")).To(Succeed())

	buf, err = encodePrivateKey(clientPrivKey)
	Expect(err).ToNot(HaveOccurred())
	Expect(pemToFile(buf, ns.certDir, "client-key_pem")).To(Succeed())

	buf, err = encodeCert(clientCert)
	Expect(err).ToNot(HaveOccurred())
	Expect(pemToFile(buf, ns.certDir, "client-cert_pem")).To(Succeed())
}

func (ns *NATSSession) writeTLSConfigFile() {
	fmtConf := fmt.Sprintf(natsTLSConf, ns.pidFile.Name(), ns.Port, ns.certDir)
	_, err := ns.configFile.Write([]byte(fmtConf))
	Expect(err).ToNot(HaveOccurred())
}

// Start starts the nats-server process.
func (ns *NATSSession) Start() {
	opts, err := server.ProcessConfigFile(ns.configFile.Name())
	Expect(err).ToNot(HaveOccurred())
	ns.server = testserver.RunServer(opts)
}

// Cleanup kills the server and cleans up the temp files.
func (ns *NATSSession) Cleanup() {
	ns.server.Shutdown()
	if ns.pidFile != nil {
		Expect(os.Remove(ns.pidFile.Name())).To(Succeed())
	}
	if ns.configFile != nil {
		Expect(os.Remove(ns.configFile.Name())).To(Succeed())
	}
}

func createTempFile(pattern string) *os.File {
	f, err := os.CreateTemp("", pattern)
	Expect(err).ToNot(HaveOccurred())

	return f
}

// picks a random port to use for the nats-server.
func pickPort() int {
	min := int64(49152)
	num, err := rand.Int(rand.Reader, big.NewInt(65535-min))
	Expect(err).ToNot(HaveOccurred())
	return int(num.Int64() + min)
}

var testCert = x509.Certificate{
	SerialNumber: big.NewInt(2019),
	Subject: pkix.Name{
		Organization: []string{"NGINX, Inc."},
		Country:      []string{"US"},
	},
	DNSNames:    []string{"localhost"},
	NotBefore:   time.Now(),
	NotAfter:    time.Now().AddDate(10, 0, 0),
	ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	KeyUsage:    x509.KeyUsageDigitalSignature,
}

// newTestCA creates a CA for testing.
func newTestCA() (*rsa.PrivateKey, []byte, x509.Certificate, error) {
	ca := testCert //nolint:varnamelen // ca is a perfectly clear name here
	ca.IsCA = true
	ca.KeyUsage |= x509.KeyUsageCertSign
	ca.BasicConstraintsValid = true

	//nolint:gosec // 1024 cert generation is faster
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, x509.Certificate{}, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, &ca, &ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, x509.Certificate{}, err
	}

	return caPrivKey, caBytes, ca, err
}

// newTestCertFromCA creates a cert for testing.
func newTestCertFromCA(
	caPrivKey *rsa.PrivateKey,
	ca *x509.Certificate, //nolint:varnamelen // ca is a perfectly clear name here
) (*rsa.PrivateKey, []byte, error) {
	cert := testCert

	//nolint:gosec // 1024 cert generation is faster
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	return certPrivKey, certBytes, nil
}

// encodePrivateKey encodes a private key for testing.
func encodePrivateKey(privKey *rsa.PrivateKey) (*bytes.Buffer, error) {
	caPrivKeyPem := new(bytes.Buffer)
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	if err = pem.Encode(caPrivKeyPem, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}); err != nil {
		return nil, err
	}

	return caPrivKeyPem, nil
}

// encodeCert encodes a cert for testing.
func encodeCert(caBytes []byte) (*bytes.Buffer, error) {
	caPem := new(bytes.Buffer)
	if err := pem.Encode(caPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return nil, err
	}

	return caPem, nil
}

// pemToFile writes PEM to file for testing.
func pemToFile(buf *bytes.Buffer, path, filename string) error {
	return os.WriteFile(filepath.Join(path, filename), buf.Bytes(), 0o600)
}
