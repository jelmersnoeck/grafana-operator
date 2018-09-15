package operator

import (
	"log"
	"time"

	"github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/internal/kit"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/scheme"
	tv1alpha1 "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/typed/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/informers/externalversions"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

// Operator is the operator that handles configuring the Monitors.
type Operator struct {
	*kit.Operator
	kubeClient kubernetes.Interface
	gClient    tv1alpha1.GrafanaV1alpha1Interface
	gInformer  cache.SharedIndexInformer
}

// NewOperator sets up a new IngressMonitor Operator which will watch for
// providers and monitors.
func NewOperator(
	kc kubernetes.Interface, gc versioned.Interface,
	namespace string, resync time.Duration) (*Operator, error) {

	// Register the scheme with the client so we can use it through the API
	crdscheme.AddToScheme(scheme.Scheme)

	gsInformer := externalversions.NewSharedInformerFactory(gc, resync).Grafana().V1alpha1()
	gInformer := gsInformer.Grafanas().Informer()

	informers := []kit.NamedInformer{
		{"Grafana", gInformer},
	}

	op := &Operator{
		Operator: &kit.Operator{
			KubeClient: kc,
			Informers:  informers,
		},
		gInformer: gInformer,
	}

	op.Operator.Queues = []kit.NamedQueue{
		kit.NewNamedQueue("Grafana", op.handleGrafana),
	}

	// Add EventHandlers for all objects we want to track
	gInformer.AddEventHandler(op)

	return op, nil
}

func (o *Operator) handleGrafana(key string) error {
	log.Printf("handling grafana %s", key)
	return nil
}

func (o *Operator) enqueueGrafana(obj *v1alpha1.Grafana) {
	log.Printf("enqueuing grafana")
	if err := o.EnqueueItem("Grafana", obj); err != nil {
		log.Printf("Error adding Grafana %s:%s to the queue: %s", obj.Namespace, obj.Name, err)
	}
}

// OnAdd handles adding of IngressMonitors and Ingresses and sets up the
// appropriate monitor with the configured providers.
func (o *Operator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.Grafana:
		o.enqueueGrafana(obj)
	}
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
	switch obj := new.(type) {
	case *v1alpha1.Grafana:
		o.enqueueGrafana(obj)
	}
}

// OnDelete handles deletion of IngressMonitors and Ingresses and deletes
// monitors from the configured providers.
func (o *Operator) OnDelete(obj interface{}) {
}
