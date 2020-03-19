package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlackboxtargetData contains: A target (url, module and service name) to be
// probed by the BlackboxTarget Exporter
type BlackboxtargetData struct {
	Url     string `json:"url"`
	Service string `json:"service"`
	Module  string `json:"module"`
}

// BlackboxTargetSpec defines the desired state of BlackboxTarget
type BlackboxTargetSpec struct {
	// BlackboxTargets defines endpoints which can be probed using the
	// Prometheus Blackbox exporter
	BlackboxTargets []BlackboxtargetData `json:"blackboxTargets,omitempty"`
}

// BlackboxTargetStatus defines the observed state of BlackboxTarget
type BlackboxTargetStatus struct {
	// Phase is a status field indicating which phase the controller is with
	// regards to reconciling blackbox target resources
	Phase int `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlackboxTarget is the Schema for the blackboxtargets API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=blackboxtargets,scope=Namespaced
type BlackboxTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlackboxTargetSpec   `json:"spec,omitempty"`
	Status BlackboxTargetStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlackboxTargetList contains a list of BlackboxTarget
type BlackboxTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlackboxTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlackboxTarget{}, &BlackboxTargetList{})
}
