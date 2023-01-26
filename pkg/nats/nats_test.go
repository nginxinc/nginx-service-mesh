package nats_test

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gonats "github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/nats"
)

const natsSubject = "test.subject"

var _ = Describe("Secure Config", func() {
	Context("On Disk Secure Config", func() {
		Context("validate", func() {
			It("returns an error if config is empty", func() {
				// an empty secure config is not valid
				conf := nats.OnDiskSecureConfig{}
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
			})
			It("returns an error if files do not exist", func() {
				// create config with cert file that doesn't exist
				conf := nats.NewOnDiskSecureConfig("server-name", "", testFilename, testFilename)
				err := conf.Validate()
				Expect(err).To(HaveOccurred())

				// create config with key file that doesn't exist
				conf = nats.NewOnDiskSecureConfig("server-name", testFilename, "", testFilename)
				err = conf.Validate()
				Expect(err).To(HaveOccurred())

				// create config with ca file that doesn't exist
				conf = nats.NewOnDiskSecureConfig("server-name", testFilename, testFilename, "")
				err = conf.Validate()
				Expect(err).To(HaveOccurred())
			})
			It("returns an error if server name is empty", func() {
				// create config with files that exist but no server name
				conf := nats.NewOnDiskSecureConfig("", testFilename, testFilename, testFilename)
				err := conf.Validate()
				Expect(err).To(HaveOccurred())
			})
			It("succeeds if all files exist and server name is non-empty", func() {
				conf := nats.NewOnDiskSecureConfig("server-name", testFilename, testFilename, testFilename)
				err := conf.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("CreateTLSConfig", func() {
			It("can create a tls config", func() {
				conf := nats.NewOnDiskSecureConfig(
					"nats-server",
					filepath.Join(testDataDir, "client-cert_pem"),
					filepath.Join(testDataDir, "client-key_pem"),
					"",
				)
				tlsConfig, err := conf.CreateTLSConfig()
				Expect(err).ToNot(HaveOccurred())
				Expect(tlsConfig).ToNot(BeNil())
				Expect(tlsConfig.ServerName).To(Equal("nats-server"))
				Expect(tlsConfig.MinVersion).To(Equal(uint16(tls.VersionTLS13)))
			})
		})
	})

	Context("In Memory Secure Config", func() {
		Context("validate", func() {
			It("returns an error when bad data is received", func() {
				certGetter := func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
					// sample error case: bad data retrieved from source
					c, e := tls.X509KeyPair([]byte("bad data"), []byte("bad data"))

					return &c, e
				}
				cont := nats.NewInMemorySecureConfig(
					"server-name",
					"who cares",
					certGetter,
				)
				Expect(cont.Validate()).ToNot(Succeed())
			})
			It("passes validation when proper data in received", func() {
				certGetter := func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
					certBytes, err := os.ReadFile(filepath.Join(testDataDir, "client-cert_pem"))
					Expect(err).ToNot(HaveOccurred())
					keyBytes, err := os.ReadFile(filepath.Join(testDataDir, "client-key_pem"))
					Expect(err).ToNot(HaveOccurred())
					cert, err := tls.X509KeyPair(certBytes, keyBytes)
					Expect(err).ToNot(HaveOccurred())

					return &cert, nil
				}
				cont := nats.NewInMemorySecureConfig(
					"server-name",
					"who cares",
					certGetter,
				)
				Expect(cont.Validate()).To(Succeed())
			})
		})
	})
})

var _ = Describe("NATs secure message bus", func() {
	It("can create a new secure message bus", func() {
		conf := nats.NewOnDiskSecureConfig(
			"nats-server",
			filepath.Join(testDataDir, "client-cert_pem"),
			filepath.Join(testDataDir, "client-key_pem"),
			filepath.Join(testDataDir, "ca_pem"),
		)
		bus, err := nats.NewSecureMessageBus(conf)
		Expect(err).ToNot(HaveOccurred())
		Expect(bus).ToNot(BeNil())
	})
	It("fails to create a new secure message bus if secure config is invalid", func() {
		conf := nats.NewOnDiskSecureConfig(
			"", "", "", "",
		)
		bus, err := nats.NewSecureMessageBus(conf)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid secure NATs configuration"))
		Expect(bus).To(BeNil())
	})
	It("returns an error if closed without nats connection", func() {
		// create empty message bus
		bus := nats.SecureMessageBus{}
		// try and close
		err := bus.Close()
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(nats.ErrNoConnection))
	})
	It("IsConnected returns false when there is no connection", func() {
		// create empty message bus
		bus := nats.SecureMessageBus{}
		Expect(bus.IsConnected()).To(BeFalse())
	})
	It("fails to connect when certs are invalid", func() {
		conf := nats.NewOnDiskSecureConfig(
			"localhost",
			testFilename,
			filepath.Join(testDataDir, "client-key_pem"),
			filepath.Join(testDataDir, "ca_pem"),
		)
		bus, err := nats.NewSecureMessageBus(conf)
		Expect(err).ToNot(HaveOccurred())

		err = bus.Connect(fmt.Sprintf("localhost:%d", natsSession.Port))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("tls: failed to find any PEM data in certificate input"))
		Expect(bus.IsConnected()).To(BeFalse())
		err = bus.Close()
		Expect(err).To(HaveOccurred())
	})
	It("fails to connect when ca cert is invalid", func() {
		// testFilename points to an empty file
		conf := nats.NewOnDiskSecureConfig(
			"localhost",
			filepath.Join(testDataDir, "client-cert_pem"),
			filepath.Join(testDataDir, "client-key_pem"),
			testFilename,
		)
		bus, err := nats.NewSecureMessageBus(conf)
		Expect(err).ToNot(HaveOccurred())

		err = bus.Connect(fmt.Sprintf("localhost:%d", natsSession.Port))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to initialize root ca"))
		Expect(bus.IsConnected()).To(BeFalse())
		err = bus.Close()
		Expect(err).To(HaveOccurred())
	})
	Context("can communicate with NATs server", func() {
		var (
			conf = nats.NewOnDiskSecureConfig(
				"localhost",
				filepath.Join(testDataDir, "client-cert_pem"),
				filepath.Join(testDataDir, "client-key_pem"),
				filepath.Join(testDataDir, "ca_pem"),
			)
			bus *nats.SecureMessageBus
		)

		BeforeEach(func() {
			var err error
			bus, err = nats.NewSecureMessageBus(conf)
			Expect(err).ToNot(HaveOccurred())
		})
		It("fails to connect if NATs server is not running", func() {
			err := bus.Connect("not a valid url")
			Expect(err).To(HaveOccurred())
			Expect(bus.IsConnected()).To(BeFalse())
		})
		It("connects to NATS with tls enabled", func() {
			err := bus.Connect(fmt.Sprintf("localhost:%d", natsSession.Port))
			Expect(err).ToNot(HaveOccurred())
			Expect(bus.IsConnected()).To(BeTrue())
			Expect(bus.Conn.TLSRequired()).To(BeTrue())
			err = bus.Close()
			Expect(err).To(BeNil())
		})
		It("connects to NATS with default options ", func() {
			err := bus.Connect(fmt.Sprintf("localhost:%d", natsSession.Port))
			Expect(err).ToNot(HaveOccurred())
			Expect(bus.IsConnected()).To(BeTrue())
			Expect(bus.Conn.Opts.AllowReconnect).To(BeTrue())
			Expect(bus.Conn.Opts.MaxReconnect).To(Equal(gonats.DefaultMaxReconnect))
			err = bus.Close()
			Expect(err).To(BeNil())
		})
		It("connects to NATS with options", func() {
			err := bus.Connect(
				fmt.Sprintf("localhost:%d", natsSession.Port),
				gonats.MaxReconnects(10),
				gonats.Timeout(5*time.Second),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(bus.IsConnected()).To(BeTrue())
			Expect(bus.Conn.TLSRequired()).To(BeTrue())
			Expect(bus.Conn.Opts.Timeout).To(Equal(5 * time.Second))
			Expect(bus.Conn.Opts.MaxReconnect).To(Equal(10))
			err = bus.Close()
			Expect(err).To(BeNil())
		})
		It("returns error on connect if NATs option cannot be applied", func() {
			// passing multiple tls configs to the nats secure option will force an error
			err := bus.Connect(
				fmt.Sprintf("localhost:%d", natsSession.Port),
				gonats.Secure(&tls.Config{MinVersion: tls.VersionTLS12}, &tls.Config{MinVersion: tls.VersionTLS12}),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to apply NATs option"))
			Expect(bus.IsConnected()).To(BeFalse())
			err = bus.Close()
			Expect(err).To(HaveOccurred())
		})
		It("can publish and subscribe", func() {
			err := bus.Connect(fmt.Sprintf("localhost:%d", natsSession.Port))
			Expect(err).ToNot(HaveOccurred())
			// subscribe to subject
			msgCh := make(chan []byte, 64)
			err = bus.Subscribe(natsSubject, msgCh)
			Expect(err).ToNot(HaveOccurred())
			Expect(bus.Conn.NumSubscriptions()).To(Equal(1))
			// publish to subject
			msg := []byte("some message")
			err = bus.Publish(natsSubject, msg)
			Expect(err).ToNot(HaveOccurred())
			// wait for msg
			var receivedMsg []byte
			Eventually(func() bool {
				receivedMsg = <-msgCh

				return true
			}).Should(BeTrue())
			Expect(string(receivedMsg)).To(Equal("some message"))
			err = bus.Close()
			Expect(err).ToNot(HaveOccurred())
		})
		It("errors if NATs connection does not exist", func() {
			noNatsConnBus := &nats.SecureMessageBus{}
			// try to publish without a connection
			err := noNatsConnBus.Publish(natsSubject, []byte("no connection"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(nats.ErrNoConnection.Error()))

			// try to subscribe without a connection
			msgCh := make(chan []byte, 64)
			err = bus.Subscribe(natsSubject, msgCh)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(nats.ErrNoConnection.Error()))
		})
	})
	It("can reconnect to server with new certs", func() {
		tempDir, err := os.MkdirTemp(".", "testdata-*")
		Expect(err).ToNot(HaveOccurred())
		defer func() { Expect(os.RemoveAll(tempDir)).To(Succeed()) }()

		curSession := NewSecureNATSSession(tempDir)
		curSession.Start()
		config := nats.NewOnDiskSecureConfig(
			"localhost",
			filepath.Join(tempDir, "client-cert_pem"),
			filepath.Join(tempDir, "client-key_pem"),
			filepath.Join(tempDir, "ca_pem"),
		)

		bus, err := nats.NewSecureMessageBus(config)
		Expect(err).ToNot(HaveOccurred())

		err = bus.Connect(fmt.Sprintf("localhost:%d", curSession.Port))
		Expect(err).ToNot(HaveOccurred())
		Expect(bus.IsConnected()).To(BeTrue())

		curSession.Cleanup()
		// Create new session to initialize new certs
		curSession = NewSecureNATSSession(tempDir)
		curSession.Start()

		err = bus.Connect(fmt.Sprintf("localhost:%d", curSession.Port))
		Expect(err).ToNot(HaveOccurred())
		Expect(bus.IsConnected()).To(BeTrue())
		curSession.Cleanup()
	})
})
