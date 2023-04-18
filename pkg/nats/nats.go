// Package nats contains the secure message bus implementation for the nats-server.
package nats

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	// ErrNoConnection indicates there is no nats connection.
	ErrNoConnection = errors.New("no nats connection found")
	// ErrInvalidConfig indicates an invalid secure NATs configuration.
	ErrInvalidConfig = errors.New("invalid secure NATs configuration")
	// ErrNoServerName indicates that no server name was provided.
	ErrNoServerName = errors.New("no server name provided")
	// ErrNoCertificate indicates that no certificate was presented.
	ErrNoCertificate = errors.New("server did not present a certificate")
)

// RootCertParseError indicates that the root certificate could not be parsed.
type RootCertParseError struct {
	file string
}

func (e RootCertParseError) Error() string {
	return fmt.Sprintf("could not parse root certificate from %s", e.file)
}

// SecureMessageBus securely connects to NATs
// and provides methods to publish and subscribe to subjects.
type SecureMessageBus struct {
	Conn *nats.Conn
	opts nats.Options
}

// SecurableConfig implements everything that
// NewSecureMessageBus needs to kick off a
// secure connection to a NATS server.
type SecurableConfig interface {
	Validate() error
	CreateTLSConfig() (*tls.Config, error)
	CAFile() string
	ServerName() string
}

// NewSecureMessageBus returns a new instance of a Secure Message Bus.
// Must provide a valid secure config in order to enable mTLS with the NATs server.
func NewSecureMessageBus(secureConfig SecurableConfig) (*SecureMessageBus, error) {
	messageBus := &SecureMessageBus{
		opts: nats.GetDefaultOptions(),
	}
	if err := secureConfig.Validate(); err != nil {
		return nil, fmt.Errorf("%v: %w", ErrInvalidConfig, err) //nolint:errorlint // only one %w allowed
	}
	tc, err := secureConfig.CreateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}
	if err = nats.Secure(tc)(&messageBus.opts); err != nil {
		return nil, fmt.Errorf("failed to set Secure option on NATs options: %w", err)
	}

	return messageBus, nil
}

// Close closes the NATs connection if it exists.
// Returns an error if no connection is found.
func (m *SecureMessageBus) Close() error {
	if m.Conn == nil {
		return ErrNoConnection
	}
	m.Conn.Close()

	return nil
}

// Connect connects to NATs with the specified opts.
func (m *SecureMessageBus) Connect(url string, opts ...nats.Option) error {
	m.opts.Url = url
	for _, o := range opts {
		if err := o(&m.opts); err != nil {
			return fmt.Errorf("failed to apply NATs option: %w", err)
		}
	}
	var err error
	m.Conn, err = m.opts.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to NATs server: %w", err)
	}

	return nil
}

// Publish publishes message to the subject.
// If publish is called before connection to NATs an error will be returned.
func (m *SecureMessageBus) Publish(subj string, msg []byte) error {
	if !m.IsConnected() {
		return ErrNoConnection
	}

	return m.Conn.Publish(subj, msg)
}

// Subscribe subscribes to the provided subject.
// If subscribe is called before connection to NATs an error will be returned.
// Data from messages sent to the subject will be placed on the msgCh channel.
func (m *SecureMessageBus) Subscribe(subj string, msgCh chan []byte) error {
	if !m.IsConnected() {
		return ErrNoConnection
	}
	_, err := m.Conn.Subscribe(subj, func(m *nats.Msg) {
		msgCh <- m.Data
	})

	return err
}

// IsConnected returns whether or not the secure message bus is connected to NATs.
func (m *SecureMessageBus) IsConnected() bool {
	return m.Conn != nil && m.Conn.IsConnected()
}

// InMemorySecureConfig implements the securableconfig
// interface from the nats package in a way that allows
// you to provide an in memory certificate getter to the
// NATS client.
type InMemorySecureConfig struct {
	certGetter       func(*tls.CertificateRequestInfo) (*tls.Certificate, error)
	serverName       string
	caBundleFilepath string
}

// NewInMemorySecureConfig creates a new InMemorySecureConfig.
func NewInMemorySecureConfig(
	server, caBundle string,
	cert func(*tls.CertificateRequestInfo) (*tls.Certificate, error),
) InMemorySecureConfig {
	return InMemorySecureConfig{
		certGetter:       cert,
		serverName:       server,
		caBundleFilepath: caBundle,
	}
}

// CAFile getter for CA Filepath.
// implements SecurableConfig from NATS package.
func (i InMemorySecureConfig) CAFile() string {
	return i.caBundleFilepath
}

// ServerName getter for ServerName.
// implements SecurableConfig from NATS package.
func (i InMemorySecureConfig) ServerName() string {
	return i.serverName
}

// Validate validates potential TLS Config.
// implements SecurableConfig from NATS package.
func (i *InMemorySecureConfig) Validate() error {
	_, err := i.certGetter(nil)

	return err
}

// CreateTLSConfig creates a TLS Config for NATS connection.
// implements SecurableConfig from NATS package.
func (i *InMemorySecureConfig) CreateTLSConfig() (*tls.Config, error) {
	return &tls.Config{
		ServerName:           i.serverName,
		MinVersion:           tls.VersionTLS13,
		GetClientCertificate: i.certGetter,
		//nolint:gosec // manually verify this cert
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return VerifyServerCertificate(i, rawCerts)
		},
	}, nil
}

// OnDiskSecureConfig contains the configuration needed to connect to NATs securely.
// All fields must be provided and all file paths must exist.
type OnDiskSecureConfig struct {
	// CertFile is the path to the client's Certificate file.
	certFile,
	// KeyFile is the path to the client's Private Key file.
	keyFile,
	// CAFile is the path to the root CA Certificate file.
	caFile,
	// ServerName is the server name of the nats-server.
	// Must match one of the DNS names on the server certificate.
	serverName string
}

// NewOnDiskSecureConfig make an OnDiskSecureConfig.
func NewOnDiskSecureConfig(
	serverName, certFile, keyFile, caFile string,
) OnDiskSecureConfig {
	return OnDiskSecureConfig{
		serverName: serverName,
		certFile:   certFile,
		keyFile:    keyFile,
		caFile:     caFile,
	}
}

// Validate validates that all files in the config exist and that the server name is non-empty.
func (sc OnDiskSecureConfig) Validate() error {
	if err := fileExists(sc.certFile); err != nil {
		return err
	}
	if err := fileExists(sc.keyFile); err != nil {
		return err
	}
	if err := fileExists(sc.caFile); err != nil {
		return err
	}
	if sc.serverName == "" {
		return ErrNoServerName
	}

	return nil
}

// CreateTLSConfig creates a TLS config using the client cert and key
// in the secure config.
func (sc OnDiskSecureConfig) CreateTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		ServerName: sc.serverName,
		//nolint:gosec // Skip server certificate verification so that we can manually verify
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(certificates [][]byte, _ [][]*x509.Certificate) error {
			return VerifyServerCertificate(sc, certificates)
		},
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(sc.certFile, sc.keyFile)

			return &cert, err
		},
		MinVersion: tls.VersionTLS13,
	}

	return tlsConfig, nil
}

// ServerName implements getter for SecurableConfig interface.
func (sc OnDiskSecureConfig) ServerName() string {
	return sc.serverName
}

// CAFile implements getter for SecurableConfig interface.
func (sc OnDiskSecureConfig) CAFile() string {
	return sc.caFile
}

// VerifyServerCertificate uses a custom CA File from a securable config
// to validate a NATS server during connection.
func VerifyServerCertificate(conf SecurableConfig, certificates [][]byte) error {
	if len(certificates) == 0 {
		return ErrNoCertificate
	}

	certs := make([]*x509.Certificate, len(certificates))
	for i, asn1Data := range certificates {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return fmt.Errorf("tls: failed to parse certificate from server: %w", err)
		}
		certs[i] = cert
	}

	rootCAs, err := initRootCAs(conf.CAFile())
	if err != nil {
		return fmt.Errorf("failed to initialize root ca: %w", err)
	}

	opts := x509.VerifyOptions{
		Roots:         rootCAs,
		CurrentTime:   time.Now(),
		DNSName:       conf.ServerName(),
		Intermediates: x509.NewCertPool(),
	}

	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	if _, err = certs[0].Verify(opts); err != nil {
		return err
	}

	return nil
}

func initRootCAs(file ...string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, fileIter := range file {
		// (tlsDir passed in via CLI flag)
		rootPEM, err := os.ReadFile(fileIter) //nolint:gosec // file here is a combination of types.Svid*Name and tlsDir
		if err != nil || rootPEM == nil {
			return nil, fmt.Errorf("could not load or parse rootCA file: %w", err)
		}
		ok := pool.AppendCertsFromPEM(rootPEM)
		if !ok {
			return nil, RootCertParseError{file: fileIter}
		}
	}

	return pool, nil
}

func fileExists(file string) error {
	if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
		return err
	}

	return nil
}
