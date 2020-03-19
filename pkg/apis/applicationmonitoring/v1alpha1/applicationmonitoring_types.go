package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationMonitoringSpec defines the desired state of ApplicationMonitoring
type ApplicationMonitoringSpec struct {
	// LabelSelector is a selector used by the Prometheus and Grafana
	// Operators to discover resources
	//
	// +kubebuilder:validation:MinLength=1
	LabelSelector string `json:"labelSelector"`

	// AdditionalScrapeConfigSecretName is the name of the secret from which
	// additional scrape configs will be passed to the prometheus operator
	AdditionalScrapeConfigSecretName string `json:"additionalScrapeConfigSecretName,omitempty"`

	// AdditionalScrapeConfigSecretKey is the key under which additional
	// scrape configs are stored within the secret
	AdditionalScrapeConfigSecretKey string `json:"additionalScrapeConfigSecretKey,omitempty"`

	// PrometheusRetention specifies retention time for prometheus data. See
	// https://prometheus.io/docs/prometheus/latest/storage/
	PrometheusRetention string `json:"prometheusRetention,omitempty"`

	// PrometheusStorageRequest is the amount of storage to assign to a
	// volume claim for persisting Prometheus data. See
	// https://github.com/coreos/prometheus-operator/blob/ca400fdc3edd0af0df896a338eca270e115b74d7/Documentation/api.md#storagespec
	PrometheusStorageRequest string `json:"prometheusStorageRequest,omitempty"`

	// PrometheusInstanceNamespaces is a list of namespaces to watch for
	// prometheus custom resources
	PrometheusInstanceNamespaces string `json:"prometheusInstanceNamespaces,omitempty"`

	// AlertmanagerInstanceNamespaces is a list of namespaces to watch for
	// alertmanager custom resources
	AlertmanagerInstanceNamespaces string `json:"alertmanagerInstanceNamespaces,omitempty"`

	// SelfSignedCerts is a flag indicating whether self-signed certs are
	// expected for routes in the blackbox target exporter in Prometheus
	SelfSignedCerts bool `json:"selfSignedCerts,omitempty"`
}

// ApplicationMonitoringStatus defines the observed state of ApplicationMonitoring
type ApplicationMonitoringStatus struct {

	// Phase is a number representing the installation progress of the
	// monitoring components
	Phase int `json:"phase"`

	// LastBlackboxConfig is an md5 hash of the Blackbox target config
	LastBlackboxConfig string `json:"lastblackboxconfig"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationMonitoring is the Schema for the applicationmonitorings API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=applicationmonitorings,scope=Namespaced
type ApplicationMonitoring struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationMonitoringSpec   `json:"spec,omitempty"`
	Status ApplicationMonitoringStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationMonitoringList contains a list of ApplicationMonitoring
type ApplicationMonitoringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationMonitoring `json:"items"`
}

type GrafanaDataSource struct {
	BasicAuthPassword string `json:"basicAuthPassword"`
	BasicAuthUser     string `json:"basicAuthUSer"`
}

type GrafanaDataSourceSecret struct {
	DataSources []GrafanaDataSource `json:"datasources"`
}

func init() {
	SchemeBuilder.Register(&ApplicationMonitoring{}, &ApplicationMonitoringList{})
}
