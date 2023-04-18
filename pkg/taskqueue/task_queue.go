// Package taskqueue contains task queue
package taskqueue

import (
	"log"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

// TaskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type TaskQueue struct {
	// queue is the work queue the worker polls
	queue *workqueue.Type
	// sync is called for each item in the queue
	sync func(string, interface{}) error
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(string, interface{}) error) *TaskQueue {
	return &TaskQueue{
		queue:      workqueue.New(),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

// Run begins running the worker for the given duration.
func (tq *TaskQueue) Run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(tq.worker, period, stopCh)
}

// Enqueue enqueues ns/name of the given api object in the task queue.
func (tq *TaskQueue) Enqueue(name string, data interface{}) {
	task := newTask(name, data)

	log.Printf("Enqueueing event: %s, key: %p", name, data)
	tq.queue.Add(task)
}

// Requeue adds the task to the queue again and logs the given error.
func (tq *TaskQueue) Requeue(task task, err error) {
	log.Printf("Requeuing %s, err %v", task.name, err)
	tq.queue.Add(task)
}

// RequeueAfter adds the task to the queue after the given duration.
func (tq *TaskQueue) RequeueAfter(t task, err error, after time.Duration) {
	log.Printf("Requeuing %s after %s, err %v", t.name, after.String(), err)
	go func(t task, after time.Duration) {
		time.Sleep(after)
		tq.queue.Add(t)
	}(t, after)
}

// Len returns the length of the queue.
func (tq *TaskQueue) Len() int {
	return tq.queue.Len()
}

// Worker processes work in the queue through sync.
func (tq *TaskQueue) worker() {
	for {
		currTask, quit := tq.queue.Get()
		if quit {
			close(tq.workerDone)

			return
		}
		event, ok := currTask.(task)
		if !ok {
			log.Printf("interface conversion: interface is %T, not task", currTask)
		}
		log.Printf("Dequeueing event: %s, key: %p", event.name, event.data)
		if err := tq.sync(event.name, event.data); err != nil {
			log.Printf("Error running callback for %s: %v", event.name, err)
		}
		tq.queue.Done(currTask)
	}
}

// Shutdown shuts down the work queue and waits for the worker to ACK.
func (tq *TaskQueue) Shutdown() {
	tq.queue.ShutDown()
	<-tq.workerDone
}

// task is an element of a TaskQueue.
type task struct {
	data interface{}
	name string
}

// newTask creates a new task.
func newTask(name string, data interface{}) task {
	return task{
		name: name,
		data: data,
	}
}
