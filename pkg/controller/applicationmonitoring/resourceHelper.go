package applicationmonitoring

import (
	"github.com/ghodss/yaml"
	applicationmonitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceHelper struct {
	templateHelper *TemplateHelper
	cr             *applicationmonitoring.ApplicationMonitoring
}

func newResourceHelper(cr *applicationmonitoring.ApplicationMonitoring) *ResourceHelper {
	return &ResourceHelper{
		templateHelper: newTemplateHelper(cr),
		cr:             cr,
	}
}

func (r *ResourceHelper) createResource(template string) (runtime.Object, error) {
	tpl, err := r.templateHelper.loadTemplate(template)
	if err != nil {
		return nil, err
	}

	resource := unstructured.Unstructured{}
	err = yaml.Unmarshal(tpl, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
