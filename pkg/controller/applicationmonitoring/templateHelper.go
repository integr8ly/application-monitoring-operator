package applicationmonitoring

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	applicationmonitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
)

const (
	GrafanaCRName                        = "grafana-cr"
	GrafanaOperatorDeploymentName        = "grafana-operator-deployment"
	GrafanaOperatorRoleBindingName       = "grafana-operator-rolebinding"
	GrafanaOperatorRoleName              = "grafana-operator-role"
	GrafanaOperatorServiceAccountName    = "grafana-operator-serviceaccount"
	AlertManagerCRName                   = "prometheus-alertmanager-cr"
	AlertManagerSecretName               = "prometheus-alertmanager-secret"
	AlertManagerServiceAccountName       = "prometheus-alertmanager-serviceaccount"
	AlertManagerServiceName              = "prometheus-alertmanager-service"
	PrometheusOperatorDeploymentName     = "prometheus-operator-deployment"
	PrometheusOperatorServiceAccountName = "prometheus-operator-serviceaccount"
	PrometheusCRName                     = "prometheus-prometheus-cr"
	PrometheusRuleCRName                 = "prometheus-prometheusrule-cr"
	PrometheusServiceAccountName         = "prometheus-serviceaccount"
	PrometheusServiceName                = "prometheus-service"
	ServiceMonitorGrafanaCRName          = "prometheus-servicemonitor-grafana-cr"
	ServiceMonitorPrometheusCRName       = "prometheus-servicemonitor-prometheus-cr"
)

type Parameters struct {
	PrometheusOperatorDeploymentName string
	Namespace                        string
}

type TemplateHelper struct {
	Parameters   Parameters
	TemplatePath string
}

// Creates a new templates helper and populates the values for all
// templates properties. Some of them (like the hostname) are set
// by the user in the custom resource
func newTemplateHelper(cr *applicationmonitoring.ApplicationMonitoring) *TemplateHelper {
	param := Parameters{
		PrometheusOperatorDeploymentName: PrometheusOperatorDeploymentName,
		Namespace:                        cr.Namespace,
	}

	templatePath := os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "./templates"
	}

	return &TemplateHelper{
		Parameters:   param,
		TemplatePath: templatePath,
	}
}

// load a templates from a given resource name. The templates must be located
// under ./templates and the filename must be <resource-name>.yaml
func (h *TemplateHelper) loadTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.yaml", h.TemplatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parsed, err := template.New("application-monitoring").Parse(string(tpl))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	err = parsed.Execute(&buffer, h.Parameters)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
