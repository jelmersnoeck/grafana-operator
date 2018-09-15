package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Grafana is the actual configuration used to configure a Grafana Deployment.
type GrafanaSpec struct {
	// The number of instances that should be deployed.
	Replicas *int32 `json:"replicas,omitempty"`

	// Base image to use to deploy Grafana.
	BaseImage string `json:"baseImage,omitempty"`

	// The tag to use for the BaseImage to deploy Grafana.
	Tag string `json:"tag,omitempty"`

	// An list of references to secrets in the same Namespace to use for pulling
	// the Grafana image from the registry.
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// The Service Accout Name within the same Namespace to use to deploy
	// Grafana.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Grafana defines the configuration which will deploy a Grafana instance.
type Grafana struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GrafanaSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GrafanaList is a list of Grafana configurations.
type GrafanaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Grafana `json:"items"`
}
