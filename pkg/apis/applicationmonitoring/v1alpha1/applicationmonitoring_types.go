package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationMonitoringSpec defines the desired state of ApplicationMonitoring
type ApplicationMonitoringSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// +kubebuilder:default:="middleware"
	LabelSelector string `json:"labelSelector"`

	// +kubebuilder:default:="integreatly-additional-scrape-configs"
	AdditionalScrapeConfigSecretName string `json:"additionalScrapeConfigSecretName,omitempty"`

	// +kubebuilder:default:="integreatly-additional.yaml"
	AdditionalScrapeConfigSecretKey string `json:"additionalScrapeConfigSecretKey,omitempty"`

	// +kubebuilder:default:="15d"
	PrometheusRetention string `json:"prometheusRetention,omitempty"`

	// +kubebuilder:default:="10Gi"
	PrometheusStorageRequest string `json:"prometheusStorageRequest,omitempty"`

	// +kubebuilder:default:="application-monitoring"
	PrometheusInstanceNamespaces string `json:"prometheusInstanceNamespaces,omitempty"`

	// +kubebuilder:default:="application-monitoring"
	AlertmanagerInstanceNamespaces string `json:"alertmanagerInstanceNamespaces,omitempty"`
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

func init() {
	SchemeBuilder.Register(&ApplicationMonitoring{}, &ApplicationMonitoringList{})
}
