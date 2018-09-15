package operator

import (
	"fmt"

	"github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func grafanaName(obj *v1alpha1.Grafana) string {
	return fmt.Sprintf("grafana-%s", obj.Name)
}

func grafanaLabels(obj *v1alpha1.Grafana) map[string]string {
	return map[string]string{"app": grafanaName(obj)}
}

func grafanaOwner(obj *v1alpha1.Grafana) metav1.OwnerReference {
	return *metav1.NewControllerRef(
		obj,
		v1alpha1.SchemeGroupVersion.WithKind("Grafana"),
	)
}

func datasourceName(obj *v1alpha1.Grafana) string {
	return fmt.Sprintf("grafana-datasources-%s", obj.Name)
}
