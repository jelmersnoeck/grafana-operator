package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatasourceSpec represents the configuration to add a provisioned datasource
// set to Grafana.
type DatasourceSpec struct {
	Type            string            `json:"type"`
	Access          string            `json:"access"`
	Version         int               `json:"version"`
	Editable        bool              `json:"editable"`
	OrganisationID  int               `json:"orgId,omitempty"`
	URL             string            `json:"url,omitempty"`
	Database        *Database         `json:"database"`
	BasicAuth       *BasicAuth        `json:"basicAuth"`
	WithCredentials bool              `json:"withCredentials"`
	IsDefault       bool              `json:"isDefault"`
	JSONData        map[string]string `json:"jsonData"`
	SecureJSONData  map[string]string `json:"secureJsonData"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Database struct {
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Datasource defines a datasource which can be used to fetch metrics from to
// make dashboards out of.
type Datasource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DatasourceSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatasourceList is a list of Datasource configurations.
type DatasourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Datasource `json:"items"`
}
