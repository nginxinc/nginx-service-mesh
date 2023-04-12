package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

type handler func(k8s.Client, os.Signal, io.Writer)

type deploySignalHandle struct {
	out       io.Writer
	handle    handler
	waitCond  *sync.Cond
	checked   chan struct{}
	sigs      chan os.Signal
	k8sClient k8s.Client
	wait      bool
}

func newDeploySignalHandle(k8sClient k8s.Client, h handler, w io.Writer) *deploySignalHandle {
	return &deploySignalHandle{
		handle:    h,
		out:       w,
		waitCond:  sync.NewCond(&sync.Mutex{}),
		checked:   make(chan struct{}, 1),
		sigs:      make(chan os.Signal, 1),
		k8sClient: k8sClient,
	}
}

// Watch the provided signals, this behavior is specific to the deploy signal handling.
// The first signal will enable the handler and run the handler once checked. The second
// signal will exit the process.
func (sh *deploySignalHandle) Watch(ctx context.Context, signals ...os.Signal) {
	go func() {
		var signaled bool
		var sig os.Signal
		signal.Notify(sh.sigs, signals...)
		defer signal.Stop(sh.sigs)
		for {
			select {
			case sig = <-sh.sigs:
				if signaled {
					s := fmt.Sprintf("\nExiting immediately on signal %v...\n", sig)
					_, _ = sh.out.Write([]byte(s))

					// intentional, reserved return, 128 + signal
					os.Exit((1 << 7) | int(sig.(syscall.Signal))) //nolint // signals are signals
				}

				sh.waitCond.L.Lock()
				sh.wait = true
				sh.waitCond.L.Unlock()

				signaled = true
				_, _ = sh.out.Write([]byte("\nReceived signal while deploying, waiting for a predictable state before aborting\n"))
				_, _ = sh.out.Write([]byte("(To exit immediately, press ^C again)\n"))
			case <-sh.checked:
				if signaled {
					sh.handle(sh.k8sClient, sig, sh.out)

					sh.waitCond.L.Lock()
					sh.wait = false
					sh.waitCond.Signal()
					sh.waitCond.L.Unlock()

					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Check signal after command processing, cannot be called after Context cancellation.
func (sh *deploySignalHandle) Check() {
	sh.checked <- struct{}{}

	sh.waitCond.L.Lock()
	for sh.wait {
		sh.waitCond.Wait()
	}
	sh.waitCond.L.Unlock()
}

func cleanOnSignal(k8sClient k8s.Client, sig os.Signal, out io.Writer) {
	_, _ = out.Write([]byte("Cleaning up NGINX Service Mesh after signal...\n"))
	deleteNamespace := true
	if err := newRemover(k8sClient).remove("nginx-service-mesh", deleteNamespace); err != nil {
		s := fmt.Sprintf("Failed cleaning, manual intervention necessary: %s\n", err)
		_, _ = out.Write([]byte(s))
	}

	// intentional, reserved return, 128 + signal
	os.Exit((1 << 7) | int(sig.(syscall.Signal))) //nolint // signals are signals
}
