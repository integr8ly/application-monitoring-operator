// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the applicationmonitoring v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=applicationmonitoring.integreatly.org
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "applicationmonitoring.integreatly.org", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// SchemaGroupVersionKind ...
	SchemaGroupVersionKindApplicationMonitoring = schema.GroupVersionKind{Group: "applicationmonitoring.integreatly.org", Version: "v1alpha1", Kind: "ApplicationMonitoring"}

	// SchemaGroupVersionKind ...
	SchemaGroupVersionKindBlackboxTarget = schema.GroupVersionKind{Group: "applicationmonitoring.integreatly.org", Version: "v1alpha1", Kind: "BlackboxTarget"}

	// SchemaGroupVersionKind ...
	SchemaGroupVersionKindGrafana = schema.GroupVersionKind{Group: "integreatly.org", Version: "v1alpha1", Kind: "Grafana"}

	// SchemaGroupVersionKind ...
	SchemaGroupVersionKindGrafanaDashboard = schema.GroupVersionKind{Group: "integreatly.org", Version: "v1alpha1", Kind: "GrafanaDashboard"}

	// SchemaGroupVersionKind ...
	SchemaGroupVersionKindGrafanaDataSource = schema.GroupVersionKind{Group: "integreatly.org", Version: "v1alpha1", Kind: "GrafanaDataSource"}
)
