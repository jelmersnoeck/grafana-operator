package operator

import (
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/internal/kit"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned"
	crdscheme "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/scheme"
	tv1alpha1 "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/clientset/versioned/typed/grafana-operator/v1alpha1"
	"github.com/jelmersnoeck/grafana-operator/pkg/client/generated/informers/externalversions"
	lv1alpha1 "github.com/jelmersnoeck/grafana-operator/pkg/client/generated/listers/grafana-operator/v1alpha1"
	yaml "gopkg.in/yaml.v2"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	lv1 "k8s.io/client-go/listers/core/v1"
	ev1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

const (
	deleteDatasourcesAnnotation = "grafana.sphc.io/deleteDatasources"
	datasourcesAnnotation       = "grafana.sphc.io/datasources"
)

// Operator is the operator that handles configuring the Monitors.
type Operator struct {
	*kit.Operator

	// clients
	kubeClient kubernetes.Interface
	gClient    tv1alpha1.GrafanaV1alpha1Interface

	// cache informers
	gInformer cache.SharedIndexInformer
	dInformer cache.SharedIndexInformer

	// listers
	gLister   lv1alpha1.GrafanaLister
	dplLister ev1beta1.DeploymentLister
	svcLister lv1.ServiceLister
	cmLister  lv1.ConfigMapLister
}

// NewOperator sets up a new IngressMonitor Operator which will watch for
// providers and monitors.
func NewOperator(kc kubernetes.Interface, gc versioned.Interface, resync time.Duration) (*Operator, error) {
	// Register the scheme with the client so we can use it through the API
	crdscheme.AddToScheme(scheme.Scheme)

	gsInformer := externalversions.NewSharedInformerFactory(gc, resync).Grafana().V1alpha1()
	gInformer := gsInformer.Grafanas().Informer()
	dInformer := gsInformer.Datasources().Informer()

	k8sInformer := informers.NewSharedInformerFactory(kc, resync)
	dplInformer := k8sInformer.Extensions().V1beta1().Deployments().Informer()
	svcInformer := k8sInformer.Core().V1().Services().Informer()
	cmInformer := k8sInformer.Core().V1().ConfigMaps().Informer()

	gLister := lv1alpha1.NewGrafanaLister(gInformer.GetIndexer())
	dplLister := ev1beta1.NewDeploymentLister(dplInformer.GetIndexer())
	svcLister := lv1.NewServiceLister(svcInformer.GetIndexer())
	cmLister := lv1.NewConfigMapLister(cmInformer.GetIndexer())

	infs := []kit.NamedInformer{
		{"Grafanas", gInformer},
		{"Datasources", dInformer},
		{"Deployments", dplInformer},
		{"Services", svcInformer},
		{"ConfigMaps", cmInformer},
	}

	op := &Operator{
		Operator: &kit.Operator{
			KubeClient: kc,
			Informers:  infs,
		},

		gInformer: gInformer,
		dInformer: dInformer,

		dplLister: dplLister,
		svcLister: svcLister,
		gLister:   gLister,
		cmLister:  cmLister,
	}

	op.Operator.Queues = []kit.NamedQueue{
		kit.NewNamedQueue("Grafana", op.handleGrafana),
		kit.NewNamedQueue("Datasource", op.handleDatasource),
	}

	// Add EventHandlers for all objects we want to track
	gInformer.AddEventHandler(op)
	dInformer.AddEventHandler(op)

	return op, nil
}

func (o *Operator) handleDatasource(key string) error {
	item, exists, err := o.dInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	// it's been deleted before we start handling it
	if !exists {
		return nil
	}

	ds := item.(*v1alpha1.Datasource)

	cache.ListAllByNamespace(o.gInformer.GetIndexer(), ds.Namespace, labels.Everything(), func(gObj interface{}) {
		obj := gObj.(*v1alpha1.Grafana)

		lblSelector, err := metav1.LabelSelectorAsSelector(&obj.Spec.ConfigSelector)
		if err != nil {
			log.Printf("Could not get label selector for %s:%s: %s", obj.Namespace, obj.Name, err)
			return
		}

		if lblSelector.Matches(labels.Set(ds.Labels)) {
			o.enqueueGrafana(obj)
		}
	})
	return nil
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

	if err := o.ensureDatasources(obj); err != nil {
		return err
	}

	if err := o.ensureDeployment(obj); err != nil {
		return err
	}

	if err := o.ensureService(obj); err != nil {
		return err
	}

	return nil
}

func (o *Operator) ensureDatasources(gf *v1alpha1.Grafana) error {
	lblSelector, err := metav1.LabelSelectorAsSelector(&gf.Spec.ConfigSelector)
	if err != nil {
		return err
	}

	ds := []Datasource{}
	err = cache.ListAllByNamespace(o.dInformer.GetIndexer(), gf.Namespace, lblSelector, func(dObj interface{}) {
		obj := dObj.(*v1alpha1.Datasource)
		ds = append(ds, v1alpha1_to_datasource(obj))
	})

	if err != nil {
		return err
	}

	gdsc, err := newDatasourceConfig(gf, ds)
	if err != nil {
		return err
	}

	dsc, err := o.cmLister.ConfigMaps(gdsc.Namespace).Get(gdsc.Name)
	if kerrors.IsNotFound(err) {
		dsc, err = o.KubeClient.Core().ConfigMaps(gdsc.Namespace).Create(gdsc)
	} else if err == nil {
		yamlData := gdsc.Data["grafana-operated.yaml"]
		md5Sum := gdsc.Annotations[datasourcesAnnotation]

		gdsc.ObjectMeta = dsc.ObjectMeta
		gdsc.Annotations[datasourcesAnnotation] = md5Sum
		gdsc.TypeMeta = dsc.TypeMeta
		data := dsc.Data
		data["grafana-operated.yaml"] = yamlData
		gdsc.Data = data

		dsc, err = o.KubeClient.Core().ConfigMaps(gdsc.Namespace).Update(gdsc)
	}

	return err
}

func newDatasourceConfig(gf *v1alpha1.Grafana, ds []Datasource) (*v1.ConfigMap, error) {
	dsc := DatasourceConfig{
		APIVersion:  1,
		Datasources: ds,
	}

	configYaml, err := yaml.Marshal(dsc)
	if err != nil {
		return nil, err
	}

	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datasourceName(gf),
			Namespace: gf.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				grafanaOwner(gf),
			},
			Annotations: map[string]string{
				datasourcesAnnotation: fmt.Sprintf("%x", md5.Sum(configYaml)),
			},
		},
		Data: map[string]string{
			"grafana-operated.yaml": string(configYaml),
		},
	}, nil
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

	cm, err := o.cmLister.ConfigMaps(obj.Namespace).Get(datasourceName(obj))
	if err != nil {
		return err
	}

	// make sure grafana is loading the latest config
	gdpl.Spec.Template.Annotations[datasourcesAnnotation] = cm.Annotations[datasourcesAnnotation]
	gdpl.Spec.Template.Annotations[deleteDatasourcesAnnotation] = cm.Annotations[deleteDatasourcesAnnotation]

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
					Labels:      labels,
					Annotations: map[string]string{},
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
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "datasources",
									MountPath: "/etc/grafana/provisioning/datasources",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "datasources",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: datasourceName(obj),
									},
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
	if err := o.EnqueueItem("Grafana", obj); err != nil {
		log.Printf("Error adding Grafana %s:%s to the queue: %s", obj.Namespace, obj.Name, err)
	}
}

func (o *Operator) enqueueDatasource(obj *v1alpha1.Datasource) {
	if err := o.EnqueueItem("Datasource", obj); err != nil {
		log.Printf("Error adding Datasource %s:%s to the queue: %s", obj.Namespace, obj.Name, err)
	}
}

// OnAdd handles adding of IngressMonitors and Ingresses and sets up the
// appropriate monitor with the configured providers.
func (o *Operator) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1alpha1.Grafana:
		o.enqueueGrafana(obj)
	case *v1alpha1.Datasource:
		o.enqueueDatasource(obj)
	}
}

// OnUpdate handles updates of IngressMonitors anad Ingresses and configures the
// checks with the configured providers.
func (o *Operator) OnUpdate(old, new interface{}) {
	switch obj := new.(type) {
	case *v1alpha1.Grafana:
		o.enqueueGrafana(obj)
	case *v1alpha1.Datasource:
		oObj := old.(*v1alpha1.Datasource)

		// nothing changed, no need to enqueue again
		if oObj.ResourceVersion == obj.ResourceVersion {
			return
		}

		o.enqueueDatasource(obj)
	}
}

// OnDelete handles deletion of IngressMonitors and Ingresses and deletes
// monitors from the configured providers.
func (o *Operator) OnDelete(obj interface{}) {
	// XXX when a provider gets deleted, we should find all the Grafana
	// instances it's used by and update the config file with a
	// `grafana-deleted` entry to mark the current provider deleted. This way
	// we don't have to influence the other file and we can keep it a separate
	// concern.
	switch obj := obj.(type) {
	case *v1alpha1.Datasource:
		err := cache.ListAllByNamespace(o.gInformer.GetIndexer(), obj.Namespace, labels.Everything(), func(gObj interface{}) {
			gf := gObj.(*v1alpha1.Grafana)

			lblSelector, err := metav1.LabelSelectorAsSelector(&gf.Spec.ConfigSelector)
			if err != nil {
				log.Printf("Could not get label selector for %s:%s: %s", gf.Namespace, gf.Name, err)
				return
			}

			// The deleted Datasource is selected by this Grafana, add the
			// provider to the deleted list.
			if lblSelector.Matches(labels.Set(obj.Labels)) {
				// make sure that we resync grafana after this
				defer o.enqueueGrafana(gf)

				odsc, err := o.cmLister.ConfigMaps(gf.Namespace).Get(datasourceName(gf))
				if kerrors.IsNotFound(err) {
					log.Printf("Configmap not found!")
					// No ConfigMap installed, skip it!
					return
				}
				dsc := odsc.DeepCopy()

				if err != nil {
					log.Printf("Could not get Datasource ConfigMap for %s:%s: %s", gf.Namespace, gf.Name, err)
					return
				}

				dlc := DatasourceConfig{APIVersion: 1, DeleteDatasources: []Datasource{}}

				// already deleted datasources present, let's load them!
				if deletes, ok := dsc.Data["grafana-deleted.yaml"]; ok {
					if err := yaml.Unmarshal([]byte(deletes), &dlc); err != nil {
						log.Printf("Could not unmarshal configmap for %s:%s: %s", gf.Namespace, datasourceName(gf), err)
						return
					}
				}

				dlc.DeleteDatasources = append(dlc.DeleteDatasources, Datasource{Name: obj.Name, OrgID: obj.Spec.OrganisationID})
				bts, err := yaml.Marshal(dlc)
				if err != nil {
					log.Printf("Could not marshal %s:%s to yaml: %s", dsc.Namespace, dsc.Name, err)
					return
				}

				data := dsc.Data
				data["grafana-deleted.yaml"] = string(bts)
				dsc.Data = data
				dsc.Annotations[deleteDatasourcesAnnotation] = fmt.Sprintf("%x", md5.Sum(bts))
				if _, err := o.KubeClient.Core().ConfigMaps(dsc.Namespace).Update(dsc); err != nil {
					log.Printf("Could not update ConfigMap %s:%s: %s", dsc.Namespace, dsc.Name, err)
				}
			}
		})

		if err != nil {
			log.Printf("Could not list grafanas for datasource %s:%s: %s", obj.Namespace, obj.Name, err)
		}
	}
}
