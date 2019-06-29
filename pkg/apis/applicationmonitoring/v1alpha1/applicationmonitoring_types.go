package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// A target (url, module and service name) to be probed by the
// blackbox exporter
type BlackboxTarget struct {
	Url     string `json:"url"`
	Service string `json:"service"`
	Module  string `json:"module"`
}

// ApplicationMonitoringSpec defines the desired state of ApplicationMonitoring
type ApplicationMonitoringSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	LabelSelector                    string            `json:"labelSelector"`
	BlackboxTargets                  []BlackboxTarget  `json:"blackboxTargets,omitempty"`
	AdditionalScrapeConfigSecretName string            `json:"additionalScrapeConfigSecretName,omitempty"`
	AdditionalScrapeConfigSecretKey  string            `json:"additionalScrapeConfigSecretKey,omitempty"`
	PrometheusRetention              string            `json:"prometheusRetention"`
	PrometheusStorage                PrometheusStorage `json:"prometheusStorage,omitempty"`
}

// ApplicationMonitoringStatus defines the observed state of ApplicationMonitoring
type ApplicationMonitoringStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Phase int `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationMonitoring is the Schema for the applicationmonitorings API
// +k8s:openapi-gen=true
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

// PrometheusStorage is StorageSpec from https://github.com/coreos/prometheus-operator/blob/cf974475f699386b1fc0bb6eff6808913adcfa61/pkg/apis/monitoring/v1/types.go#L302-L308
type PrometheusStorage struct {
	EmptyDir            *v1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	VolumeClaimTemplate v1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ApplicationMonitoring{}, &ApplicationMonitoringList{})
}
