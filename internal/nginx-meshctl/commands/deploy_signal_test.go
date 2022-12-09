package commands

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Signal Handler", func() {
	var (
		buf     *gbytes.Buffer
		sig     os.Signal
		sendSig os.Signal
		msg     string

		handle        handler
		signalHandler *deploySignalHandle
	)
	BeforeEach(func() {
		fakeK8s := &fake.FakeClient{}
		buf = gbytes.NewBuffer()
		sig = os.Interrupt
		sendSig = syscall.SIGUSR1
		msg = "called in handler"

		handle = func(client k8s.Client, s os.Signal, w io.Writer, _ string) {
			defer GinkgoRecover()
			Expect(s).To(Equal(sendSig))

			_, innerErr := w.Write([]byte(msg))
			Expect(innerErr).ToNot(HaveOccurred())
		}
		signalHandler = newDeploySignalHandle(fakeK8s, handle, buf, "test")
		Expect(signalHandler).ToNot(BeNil())
	})

	It("watches and checks without a signal", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		signalHandler.Watch(ctx, sig)
		signalHandler.Check()
		Eventually(buf).ShouldNot(gbytes.Say(msg))
	})
	It("can check without watch", func() {
		success := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			signalHandler.Check()
			Eventually(buf).ShouldNot(gbytes.Say(msg))
			success <- struct{}{}
		}()

		nukeIt := time.NewTimer(60 * time.Second)
		select {
		case <-nukeIt.C:
			Expect(false).To(BeTrue(), "Timed out, expected Check deadlock")
		case <-success:
		}
	})
	It("watches and handles signals on check", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		signalHandler.Watch(ctx, sendSig)

		signalHandler.Check()
		Eventually(buf).ShouldNot(gbytes.Say(msg))
		signalHandler.Check()
		Eventually(buf).ShouldNot(gbytes.Say(msg))

		// wait for the signal to process
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, sendSig)

		err := syscall.Kill(syscall.Getpid(), sendSig.(syscall.Signal)) //nolint:forcetypeassert // signals are signals
		Expect(err).ToNot(HaveOccurred())

		<-sig

		signalHandler.Check()
		Eventually(buf).Should(gbytes.Say(msg))
	})
})
