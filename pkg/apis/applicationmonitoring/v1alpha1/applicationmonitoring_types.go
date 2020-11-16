package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationMonitoringSpec defines the desired state of ApplicationMonitoring
type ApplicationMonitoringSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// +kubebuilder:validation:MinLength=1
	LabelSelector                    string           `json:"labelSelector"`
	AdditionalScrapeConfigSecretName string           `json:"additionalScrapeConfigSecretName,omitempty"`
	AdditionalScrapeConfigSecretKey  string           `json:"additionalScrapeConfigSecretKey,omitempty"`
	PriorityClassName                string           `json:"priorityClassName,omitempty"`
	PrometheusRetention              string           `json:"prometheusRetention,omitempty"`
	PrometheusStorageRequest         string           `json:"prometheusStorageRequest,omitempty"`
	PrometheusInstanceNamespaces     string           `json:"prometheusInstanceNamespaces,omitempty"`
	AlertmanagerInstanceNamespaces   string           `json:"alertmanagerInstanceNamespaces,omitempty"`
	SelfSignedCerts                  bool             `json:"selfSignedCerts,omitempty"`
	Affinity                         *corev1.Affinity `json:"affinity,omitempty"`
}

// ApplicationMonitoringStatus defines the observed state of ApplicationMonitoring
type ApplicationMonitoringStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	Phase              int    `json:"phase"`
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
