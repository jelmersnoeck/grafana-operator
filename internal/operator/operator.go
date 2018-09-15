package operator

import (
	"fmt"
	"log"
	"time"

	"github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/internal/kit"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/scheme"
	tv1alpha1 "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/typed/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/informers/externalversions"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	lv1 "k8s.io/client-go/listers/core/v1"
	ev1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Operator is the operator that handles configuring the Monitors.
type Operator struct {
	*kit.Operator

	// clients
	kubeClient kubernetes.Interface
	gClient    tv1alpha1.GrafanaV1alpha1Interface

	// cache informers
	gInformer cache.SharedIndexInformer

	// listers
	dplLister ev1beta1.DeploymentLister
	svcLister lv1.ServiceLister
}

// NewOperator sets up a new IngressMonitor Operator which will watch for
// providers and monitors.
func NewOperator(kc kubernetes.Interface, gc versioned.Interface, resync time.Duration) (*Operator, error) {
	// Register the scheme with the client so we can use it through the API
	crdscheme.AddToScheme(scheme.Scheme)

	gsInformer := externalversions.NewSharedInformerFactory(gc, resync).Grafana().V1alpha1()
	gInformer := gsInformer.Grafanas().Informer()

	k8sInformer := informers.NewSharedInformerFactory(kc, resync)
	dplInformer := k8sInformer.Extensions().V1beta1().Deployments().Informer()
	svcInformer := k8sInformer.Core().V1().Services().Informer()

	dplLister := ev1beta1.NewDeploymentLister(dplInformer.GetIndexer())
	svcLister := lv1.NewServiceLister(svcInformer.GetIndexer())

	infs := []kit.NamedInformer{
		{"Grafanas", gInformer},
		{"Deployments", dplInformer},
		{"Services", svcInformer},
	}

	op := &Operator{
		Operator: &kit.Operator{
			KubeClient: kc,
			Informers:  infs,
		},

		gInformer: gInformer,

		dplLister: dplLister,
		svcLister: svcLister,
	}

	op.Operator.Queues = []kit.NamedQueue{
		kit.NewNamedQueue("Grafana", op.handleGrafana),
	}

	// Add EventHandlers for all objects we want to track
	gInformer.AddEventHandler(op)

	return op, nil
}

func (o *Operator) handleGrafana(key string) error {
	item, exists, err := o.gInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// it's been deleted before we start handling it
	if !exists {
		return nil
	}

	obj := item.(*v1alpha1.Grafana)

	if err := o.ensureDeployment(obj); err != nil {
		return err
	}

	if err := o.ensureService(obj); err != nil {
		return err
	}

	return nil
}

func (o *Operator) ensureService(obj *v1alpha1.Grafana) error {
	gsvc, err := newService(obj)
	if err != nil {
		return err
	}

	svc, err := o.svcLister.Services(gsvc.Namespace).Get(gsvc.Name)
	if kerrors.IsNotFound(err) {
		svc, err = o.KubeClient.Core().Services(gsvc.Namespace).Create(gsvc)
	} else if err == nil {
		gsvc.ObjectMeta = svc.ObjectMeta
		gsvc.TypeMeta = svc.TypeMeta
		gsvc.Status = svc.Status
		gsvc.Spec.ClusterIP = svc.Spec.ClusterIP

		svc, err = o.KubeClient.Core().Services(gsvc.Namespace).Update(gsvc)
	}

	return err
}

func newService(obj *v1alpha1.Grafana) (*v1.Service, error) {
	name := grafanaName(obj)

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: obj.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				grafanaOwner(obj),
			},
		},
		Spec: v1.ServiceSpec{
			Selector: grafanaLabels(obj),
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       3000,
					TargetPort: intstr.FromString("web"),
				},
			},
			Type: v1.ServiceTypeNodePort,
		},
	}, nil
}

func (o *Operator) ensureDeployment(obj *v1alpha1.Grafana) error {
	gdpl, err := newDeployment(obj)
	if err != nil {
		return err
	}

	dpl, err := o.dplLister.Deployments(gdpl.Namespace).Get(gdpl.Name)
	if kerrors.IsNotFound(err) {
		dpl, err = o.KubeClient.Extensions().Deployments(gdpl.Namespace).Create(gdpl)
	} else if err == nil {
		gdpl.ObjectMeta = dpl.ObjectMeta
		gdpl.TypeMeta = dpl.TypeMeta
		gdpl.Status = dpl.Status

		dpl, err = o.KubeClient.Extensions().Deployments(gdpl.Namespace).Update(gdpl)
	}

	return err
}

func newDeployment(obj *v1alpha1.Grafana) (*v1beta1.Deployment, error) {
	name := grafanaName(obj)
	labels := grafanaLabels(obj)
	image := "grafana/grafana"

	// XXX get config map hashes and set as annotations
	// XXX link volumes
	// XXX set service proxy name
	return &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: obj.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				grafanaOwner(obj),
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: obj.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []v1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: 3000,
									Protocol:      v1.ProtocolTCP,
								},
							},
							Resources: obj.Spec.Resources,
							Env: []v1.EnvVar{
								{
									Name:  "GF_SERVER_ROOT_URL",
									Value: fmt.Sprintf("/api/v1/namespaces/%s/services/%s/proxy/", obj.Namespace, name),
								},
							},
						},
					},
				},
			},
		},
	}, nil
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
