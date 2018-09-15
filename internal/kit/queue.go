package kit

import "k8s.io/client-go/util/workqueue"

type NamedQueue struct {
	Name  string
	Func  func(string) error
	Queue workqueue.RateLimitingInterface
}

func NewNamedQueue(name string, fn func(string) error) NamedQueue {
	return NamedQueue{
		Name:  name,
		Func:  fn,
		Queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
	}
}
