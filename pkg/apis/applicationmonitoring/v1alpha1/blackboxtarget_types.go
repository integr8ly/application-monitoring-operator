package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BlackboxtargetStructure contains:
// A target (url, module and service name) to be probed by the
type BlackboxtargetData struct {
	Url     string `json:"url"`
	Service string `json:"service"`
	Module  string `json:"module"`
}

// BlackboxTargetSpec defines the desired state of BlackboxTarget
type BlackboxTargetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	BlackboxTargets []BlackboxtargetData `json:"blackboxTargets,omitempty"`
}

// BlackboxTargetStatus defines the observed state of BlackboxTarget
type BlackboxTargetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
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
