package kit

import (
	"errors"
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	// ErrCouldNotSyncCache is used when there is an error waiting for cache
	// syncs. This could be due to not being able to connect to the API Server
	// or due to a stop signal being sent.
	ErrCouldNotSyncCache = errors.New("could not sync caches")

	// ErrQueueNotFound is used when the operator is instructed to fetch
	// or append something from or to a queue, but the queue is not present.
	ErrQueueNotFound = errors.New("queue not found")
)

// Operator represents the base functionality required to run an operator.
type Operator struct {
	Name       string
	Queues     []NamedQueue
	Informers  []NamedInformer
	KubeClient kubernetes.Interface
}

// Run starts the Operator and blocks until a message is received on stopCh.
func (o *Operator) Run(stopCh <-chan struct{}) error {
	defer o.shutdownQueues()

	log.Printf("Starting %s Operator", o.Name)
	if err := o.connectToCluster(stopCh); err != nil {
		return err
	}

	log.Printf("Starting the informers")
	if err := o.startInformers(stopCh); err != nil {
		return err
	}

	log.Printf("Starting the workers")
	for _, q := range o.Queues {
		for i := 0; i < 4; i++ {
			log.Printf("Starting worker %d for queue %s", i, q.Name)
			go wait.Until(runWorker(o.processNextItem(q)), time.Second, stopCh)
		}
	}

	<-stopCh
	log.Printf("Stopping Grafana Operator")

	return nil
}

func (o *Operator) processNextItem(nq NamedQueue) func() bool {
	return func() bool {
		return o.handleQueuedItem(nq.Name, nq.Queue, nq.Func)
	}
}

func runWorker(queue func() bool) func() {
	return func() {
		for queue() {
		}
	}
}

func (o *Operator) connectToCluster(stopCh <-chan struct{}) error {
	errCh := make(chan error)
	go func() {
		v, err := o.KubeClient.Discovery().ServerVersion()
		if err != nil {
			errCh <- fmt.Errorf("Could not communicate with the server: %s", err)
			return
		}

		log.Printf("Connected to the cluster (version %s)", v)
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-stopCh:
		return nil
	}

	return nil
}

func (o *Operator) startInformers(stopCh <-chan struct{}) error {
	for _, inf := range o.Informers {
		log.Printf("Starting informer %s", inf.Name)
		go inf.Informer.Run(stopCh)
	}

	if err := o.waitForCaches(stopCh); err != nil {
		return err
	}

	log.Printf("Synced all caches")
	return nil
}

func (o *Operator) handleQueuedItem(name string, queue workqueue.RateLimitingInterface, handlerFunc func(string) error) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	// wrap this in a function so we can use defer to mark processing the item
	// as done.
	err := func(obj interface{}) error {
		defer queue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			queue.Forget(obj)

			log.Printf("Expected object name in %s workqueue, got %#v", name, obj)
			return nil
		}

		if err := handlerFunc(key); err != nil {
			return fmt.Errorf("Error handling '%s' in %s workqueue: %s", key, name, err)
		}

		queue.Forget(obj)
		log.Printf("Synced '%s' in %s workqueue", key, name)
		return nil
	}(obj)

	if err != nil {
		log.Printf(err.Error())
	}

	return true
}

func (o *Operator) waitForCaches(stopCh <-chan struct{}) error {
	var syncFailed bool
	for _, inf := range o.Informers {
		log.Printf("Waiting for cache sync for %s", inf.Name)
		if !cache.WaitForCacheSync(stopCh, inf.Informer.HasSynced) {
			log.Printf("Could not sync cache for %s", inf.Name)
			syncFailed = true
		} else {
			log.Printf("Synced cache for %s", inf.Name)
		}
	}

	if syncFailed {
		return ErrCouldNotSyncCache
	}

	return nil
}

func (o *Operator) shutdownQueues() {
	for _, q := range o.Queues {
		log.Printf("Shutting down %s queue", q.Name)
		q.Queue.ShutDown()
	}
}

// EnqueueItem adds the given object to the queue associated with the given
// name.
func (o *Operator) EnqueueItem(name string, obj interface{}) error {
	var ok bool
	var queue workqueue.RateLimitingInterface
	for _, q := range o.Queues {
		if q.Name == name {
			queue = q.Queue
			ok = true
		}
	}

	if !ok {
		return ErrQueueNotFound
	}

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		return err
	}

	queue.AddRateLimited(key)
	return nil
}
