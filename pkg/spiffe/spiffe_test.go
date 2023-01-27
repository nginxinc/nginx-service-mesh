package spiffe_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	sc "github.com/nginxinc/nginx-service-mesh/pkg/spiffe"
	"github.com/nginxinc/nginx-service-mesh/pkg/spiffe/spiffefakes"
	"github.com/nginxinc/nginx-service-mesh/pkg/taskqueue"
)

var _ = Describe("SpiffeCert", func() {
	Describe("CertManager", func() {
		const (
			certWaitTimeout = 3 * time.Minute
		)
		var (
			reloader    *spiffefakes.FakeReloader
			writer      *spiffefakes.FakeSVIDWriter
			certFetcher *spiffefakes.FakeCertFetcher
			certManager *sc.CertManager
			certCh      chan *workloadapi.X509Context
			errCh       chan error
		)
		BeforeEach(func() {
			reloader = &spiffefakes.FakeReloader{}
			writer = &spiffefakes.FakeSVIDWriter{}
			certFetcher = &spiffefakes.FakeCertFetcher{}
			// create channels for certFetcher
			certCh = make(chan *workloadapi.X509Context, 10)
			errCh = make(chan error)
			// set response from fake cert fetcher
			certFetcher.StartReturns(certCh, errCh, nil)
		})
		JustBeforeEach(func() {
			certManager = sc.NewCertManagerWithReloader(reloader, writer, certFetcher, certWaitTimeout)
		})
		It("fetches certs, writes, and reloads until stop signal is received", func() {
			ctx, cancel := context.WithCancel(context.Background())
			var err error
			certCh <- &workloadapi.X509Context{} // initial trust bundle
			err = certManager.Run(ctx)
			Expect(err).ToNot(HaveOccurred())

			// write certs to channel
			for i := 0; i < 10; i++ {
				certCh <- &workloadapi.X509Context{}
			}
			fetchDone := make(chan struct{})
			// wait for all the certs to be read off the channel
			go func() {
				for {
					if len(certCh) == 0 {
						close(fetchDone)

						return
					}
				}
			}()
			<-fetchDone
			cancel()
			err = <-certManager.ErrCh
			Expect(err).Should(MatchError(context.Canceled))
			Expect(writer.WriteCallCount()).Should(BeNumerically(">=", 9))
			Expect(reloader.ReloadCallCount()).Should(BeNumerically(">=", 9))
		})
		It("exits on write error", func() {
			ctx, cancel := context.WithCancel(context.Background())
			var err error
			// stub writer error
			writer.WriteReturns(errFakeWriteFail)

			go func() {
				err = certManager.Run(ctx)
				cancel()
			}()
			// put a cert on the cert channel
			certCh <- &workloadapi.X509Context{}
			<-ctx.Done()
			Expect(err.Error()).To(ContainSubstring("fake write error"))
		})
		It("quits and logs error on certFetcher start error", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// stub error response from cert fetcher
			certFetcher.StartReturns(nil, nil, errFakeFetchFail)
			err := certManager.Run(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake fetch error"))
		})
		It("quits when context is canceled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			var err error
			certCh <- &workloadapi.X509Context{} // initial trust bundle
			err = certManager.Run(ctx)
			Expect(err).ToNot(HaveOccurred())
			cancel()
			err = <-certManager.ErrCh
			// check that certManager exits after context is canceled
			Expect(err).Should(MatchError(context.Canceled))
		})
	})

	Describe("Disk SVID Writer", func() {
		config := sc.DiskSVIDConfig{
			KeyFilename:      "tls.key",
			CertFilename:     "tls.crt",
			CABundleFilename: "ca.crt",
		}
		It("creates correct file paths from config", func() {
			// create directory
			dir, err := os.MkdirTemp("", "ssl")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(os.RemoveAll(dir)).To(Succeed())
			}()
			config.CertDir = dir
			writer, err := sc.NewDiskSVIDWriter(config)
			Expect(err).ToNot(HaveOccurred())
			Expect(writer).ToNot(BeNil())
			Expect(writer.CaBundleFile).To(Equal(fmt.Sprintf("%s/ca.crt", dir)))
			Expect(writer.KeyFile).To(Equal(fmt.Sprintf("%s/tls.key", dir)))
			Expect(writer.CertFile).To(Equal(fmt.Sprintf("%s/tls.crt", dir)))
		})
		It("errors if certificate directory does not exist", func() {
			config.CertDir = "/tmp/ssl"
			writer, err := sc.NewDiskSVIDWriter(config)
			Expect(err).To(HaveOccurred())
			Expect(writer).To(BeNil())
		})
		Context("from the x509Context", func() {
			var context *workloadapi.X509Context
			BeforeEach(func() {
				sid, err := spiffeid.FromString("spiffe://example.test/workload")
				Expect(err).ToNot(HaveOccurred())
				svids := []*x509svid.SVID{
					{
						ID:           sid,
						Certificates: testCert,
						PrivateKey:   testKey,
					},
				}
				bundle := x509bundle.FromX509Authorities(sid.TrustDomain(), testCACert)
				context = &workloadapi.X509Context{
					SVIDs:   svids,
					Bundles: x509bundle.NewSet(bundle),
				}
			})
			It("parses the certificate", func() {
				cert := context.DefaultSVID().Certificates
				Expect(cert).To(Equal(testCert))
			})
			It("parses the key", func() {
				key := context.DefaultSVID().PrivateKey
				Expect(key).To(Equal(testKey))
			})
			It("parses the ca cert", func() {
				bundle, err := sc.ParseCABundle(context)
				Expect(err).ToNot(HaveOccurred())
				cert := bundle.X509Authorities()
				Expect(cert).To(Equal(testCACert))
			})
			It("writes keys and certs to disk", func() {
				// create directory
				dir, err := os.MkdirTemp("", "ssl")
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					Expect(os.RemoveAll(dir)).To(Succeed())
				}()
				config.CertDir = dir
				writer, err := sc.NewDiskSVIDWriter(config)
				Expect(err).ToNot(HaveOccurred())
				Expect(writer).ToNot(BeNil())
				err = writer.Write(context)
				Expect(err).ToNot(HaveOccurred())

				// read in certs and keys from disk
				cert, err := os.ReadFile(writer.CertFile)
				Expect(err).ToNot(HaveOccurred())
				// Trim leading newline from cert and compare
				Expect(string(cert)).To(Equal(strings.TrimLeft(certPEM, "\n")))

				caCert, err := os.ReadFile(writer.CaBundleFile)
				Expect(err).ToNot(HaveOccurred())
				// Trim leading newline from cert and compare
				Expect(string(caCert)).To(Equal(strings.TrimLeft(rootPEM, "\n")))

				key, err := os.ReadFile(writer.KeyFile)
				Expect(err).ToNot(HaveOccurred())
				// convert PKCS #8 key to rsa private key
				block, _ := pem.Decode(key)
				Expect(block).ToNot(BeNil())
				privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
				Expect(err).ToNot(HaveOccurred())
				Expect(privKey).To(Equal(testKey))
			})
		})
	})

	Describe("CertFetcher", func() {
		It("creates a watcher that writes to cert channel on update", func() {
			ctx, cancel := context.WithCancel(context.Background())
			client := &spiffefakes.FakeClient{}
			client.WatchX509ContextStub = stubContextUpdateCall
			cf, err := sc.NewX509CertFetcher("spire-addr", client)
			Expect(err).ToNot(HaveOccurred())
			certCh, errCh, err := cf.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
			Eventually(certCh).Should(Receive())
			cancel()
			Consistently(errCh).ShouldNot(Receive())
			Expect(client.WatchX509ContextCallCount()).To(Equal(1))
		})
		It("writes to error channel on fatal error", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &spiffefakes.FakeClient{}
			client.WatchX509ContextReturns(errFakeWatchFail)
			cf, err := sc.NewX509CertFetcher("spire-addr", client)
			Expect(err).ToNot(HaveOccurred())
			_, errCh, err := cf.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
			Eventually(errCh).Should(Receive())
		})
	})

	Describe("SVID", func() {
		Context("Write", func() {
			cert, key, caBytes := getTestCACertKey()
			resp := makeSVIDResponse(cert, key, caBytes)

			It("writes a loadable CA bundle to a temp dir", func() {
				handler := sc.CABundleManager{
					CABundleFilepath: "/tmp/gotestrootca.pem",
					TaskQueue: taskqueue.NewTaskQueue(func(_ string, _ interface{}) error {
						return nil
					}),
				}
				channel := make(chan error, 1)
				go func() {
					channel <- handler.WaitForCABundle(5)
				}()
				Expect(handler.Write(resp)).To(Succeed())
				Expect(<-channel).To(Succeed())
			})

			It("initializes latestCABundleHash", func() {
				handler := sc.CABundleManager{
					CABundleFilepath: "/tmp/gotestrootca.pem",
					TaskQueue: taskqueue.NewTaskQueue(func(_ string, _ interface{}) error {
						return nil
					}),
				}
				Expect(handler.Write(resp)).To(Succeed())
				Expect(handler.TestAndUpdateCABundle(caBytes)).To(Equal(false))
			})
		})

		Context("TestAndUpdateCABundle", func() {
			cert, key, ca := getTestCACertKey()
			resp := makeSVIDResponse(cert, key, ca)

			handler := sc.CABundleManager{
				CABundleFilepath: "/tmp/gotestrootca.pem",
				TaskQueue: taskqueue.NewTaskQueue(func(_ string, _ interface{}) error {
					return nil
				}),
			}
			Expect(handler.Write(resp)).To(Succeed())

			It("new CA updates hash", func() {
				_, _, nca := getTestCACertKey()
				Expect(handler.TestAndUpdateCABundle(nca)).To(Equal(true))
			})
		})

		Context("CertKeyBytesFromSVIDResponse and GetCertificateGetter", func() {
			cert, key, ca := getTestCACertKey()
			resp := makeSVIDResponse(cert, key, ca)
			certPair, err := tls.X509KeyPair(cert, key)
			Expect(err).ToNot(HaveOccurred())

			handler := sc.CABundleManager{
				CABundleFilepath: "/tmp/gotestrootca.pem",
				TaskQueue: taskqueue.NewTaskQueue(func(_ string, _ interface{}) error {
					return nil
				}),
			}
			cert1, key1, err := handler.CertKeyBytesFromSVIDResponse(resp)
			Expect(err).ToNot(HaveOccurred())
			getCertKey := handler.NewCertificateGetter()

			It("sets cert and key", func() {
				Expect(cert1).To(Equal(cert))
				Expect(key1).To(Equal(key))
				cert, err := getCertKey(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(*cert).To(Equal(certPair))
			})

			It("updates cert and key", func() {
				// make another set of these
				cert, key, ca := getTestCACertKey()
				resp = makeSVIDResponse(cert, key, ca)
				certPair2, err := tls.X509KeyPair(cert, key)
				Expect(err).ToNot(HaveOccurred())
				c2, k2, err := handler.CertKeyBytesFromSVIDResponse(resp)
				Expect(err).ToNot(HaveOccurred())
				Expect(c2).To(Equal(cert))
				Expect(k2).To(Equal(key))
				cert2, err := getCertKey(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(*cert2).To(Equal(certPair2))
				Expect(*cert2).To(Not(Equal(certPair)))
			})
		})

		Context("SerialNumberFromSVIDResponse", func() {
			handler := sc.CABundleManager{
				CABundleFilepath: "/tmp/gotestrootca.pem",
				TaskQueue: taskqueue.NewTaskQueue(func(_ string, _ interface{}) error {
					return nil
				}),
			}
			cert, key, ca := getTestCACertKey()
			resp := makeSVIDResponse(cert, key, ca)

			It("provides a serial number", func() {
				serial, err := handler.SerialNumberFromSVIDResponse(resp)
				Expect(err).ToNot(HaveOccurred())
				Expect(serial).ToNot(BeNil())
				Expect(serial).ToNot(BeEmpty())
			})

			It("throws an error", func() {
				// break my SVID response
				resp.DefaultSVID().Certificates = []*x509.Certificate{}
				serial, err := handler.SerialNumberFromSVIDResponse(resp)
				Expect(err).To(HaveOccurred())
				Expect(serial).To(BeNil())
			})
		})
	})
})
