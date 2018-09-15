package operator

import "github.com/jelmersnoeck/grafana-operator/apis/grafana-operator/v1alpha1"

// DatasourceConfig is the representation of a Datasource Configuration file for
// Grafana. It will be used to store the data in a configmap.
type DatasourceConfig struct {
	APIVersion        int          `yaml:"apiVersion"`
	DeleteDatasources []Datasource `yaml:"deleteDatasources"`
	Datasources       []Datasource `yaml:"datasources"`
}

// Datasource represents the spec for a Datasource in Grafana.
type Datasource struct {
	Name              string            `yaml:"name"`
	Type              string            `yaml:"type"`
	Access            string            `yaml:"access"`
	OrgID             int               `yaml:"orgId,omitempty"`
	URL               string            `yaml:"url,omitempty"`
	Password          string            `yaml:"password,omitempty"`
	User              string            `yaml:"user,omitempty"`
	Database          string            `yaml:"database,omitempty"`
	BasicAuth         bool              `yaml:"basicAuth,omitempty"`
	BasicAuthUser     string            `yaml:"basicAuthUser,omitempty"`
	BasicAuthPassword string            `yaml:"basicAuthPassword,omitempty"`
	WithCredentials   bool              `yaml:"withCredentials,omitempty"`
	IsDefault         bool              `yaml:"isDefault,omitempty"`
	JSONData          map[string]string `yaml:"jsonData,omitempty"`
	SecureJSONData    map[string]string `yaml:"secureJsonData,omitempty"`
	Version           int               `yaml:"version"`
	Editable          bool              `yaml:"editable,omitempty"`
}

func v1alpha1_to_datasource(ds *v1alpha1.Datasource) Datasource {
	var ba bool
	var baUser, baPassword, dbDatabase, dbUsername, dbPassword string
	if ds.Spec.BasicAuth != nil {
		ba = true
		baUser = ds.Spec.BasicAuth.Username
		baPassword = ds.Spec.BasicAuth.Password
	}

	if ds.Spec.Database != nil {
		dbDatabase = ds.Spec.Database.Database
		dbUsername = ds.Spec.Database.Username
		dbPassword = ds.Spec.Database.Password
	}

	return Datasource{
		Name:              ds.Name,
		Type:              ds.Spec.Type,
		Access:            ds.Spec.Access,
		OrgID:             ds.Spec.OrganisationID,
		URL:               ds.Spec.URL,
		Database:          dbDatabase,
		User:              dbUsername,
		Password:          dbPassword,
		BasicAuth:         ba,
		BasicAuthUser:     baUser,
		BasicAuthPassword: baPassword,
		WithCredentials:   ds.Spec.WithCredentials,
		IsDefault:         ds.Spec.IsDefault,
		JSONData:          ds.Spec.JSONData,
		SecureJSONData:    ds.Spec.SecureJSONData,
		Version:           ds.Spec.Version,
		Editable:          ds.Spec.Editable,
	}
}
