package taskqueue_test

import (
	"sync"
	"time"

	"github.com/nginxinc/nginx-service-mesh/pkg/taskqueue"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task Queue", func() {
	var taskQ *taskqueue.TaskQueue
	var mutex sync.Mutex

	It("queues events", func() {
		taskQ = taskqueue.NewTaskQueue(func(name string, data interface{}) error {
			return nil
		})
		taskQ.Enqueue("queues events", 5)
		taskQ.Enqueue("queues events", 6)
		Expect(taskQ.Len()).To(Equal(2))
	})
	It("does not queue duplicate events", func() {
		taskQ = taskqueue.NewTaskQueue(func(name string, data interface{}) error {
			return nil
		})
		taskQ.Enqueue("does not queue duplicate events", 5)
		taskQ.Enqueue("does not queue duplicate events", 5)
		taskQ.Enqueue("does not queue duplicate events", 5)
		Expect(taskQ.Len()).To(Equal(1))
	})
	It("processes events", func() {
		var eventData interface{}
		var eventMsg string
		stopCh := make(chan struct{})

		taskQ = taskqueue.NewTaskQueue(func(name string, data interface{}) error {
			mutex.Lock()
			defer mutex.Unlock()
			eventData = data
			eventMsg = name

			return nil
		})
		go taskQ.Run(time.Second, stopCh)
		taskQ.Enqueue("processes events", 5)
		Eventually(func() string {
			mutex.Lock()
			defer mutex.Unlock()
			msg := eventMsg

			return msg
		}, 5*time.Second).Should(Equal("processes events"))
		Expect(eventData).To(Equal(5))
		close(stopCh)
		taskQ.Shutdown()
	})
	It("processes events in order", func() {
		var eventData []interface{}
		stopCh := make(chan struct{})

		taskQ = taskqueue.NewTaskQueue(func(name string, data interface{}) error {
			mutex.Lock()
			defer mutex.Unlock()
			eventData = append(eventData, data)

			return nil
		})
		go taskQ.Run(time.Second, stopCh)
		taskQ.Enqueue("processes events in order", 0)
		taskQ.Enqueue("processes events in order", 1)
		taskQ.Enqueue("processes events in order", 2)
		taskQ.Enqueue("processes events in order", 3)
		Eventually(func() int {
			mutex.Lock()
			defer mutex.Unlock()
			length := len(eventData)

			return length
		}, 5*time.Second).Should(Equal(4))
		for i := range eventData {
			mutex.Lock()
			Expect(eventData[i]).To(Equal(i))
			mutex.Unlock()
		}
		close(stopCh)
		taskQ.Shutdown()
	})
})
