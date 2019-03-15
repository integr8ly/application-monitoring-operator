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
	ApplicationMonitoringName            = "application-monitoring"
	GrafanaCrName                        = "grafana"
	GrafanaOperatorName                  = "grafana-operator"
	GrafanaOperatorRoleName              = "grafana-operator-role"
	GrafanaOperatorRoleBindingName       = "grafana-operator-role-binding"
	GrafanaOperatorServiceAccountName    = "grafana-operator-service-account"
	PrometheusOperatorName               = "prometheus-operator"
	PrometheusOperatorServiceAccountName = "prometheus-operator-service-account"
	PrometheusCrName                     = "prometheus"
	PrometheusRouteName                  = "prometheus-route"
	PrometheusServiceAccountName         = "prometheus-service-account"
	PrometheusServiceName                = "prometheus-service"
	AlertManagerServiceAccountName       = "alertmanager-service-account"
	AlertManagerCrName                   = "alertmanager"
	AlertManagerServiceName              = "alertmanager-service"
	AlertManagerSecretName               = "alertmanager-secret"
	PrometheusServiceMonitorName         = "prometheus-servicemonitor"
	PrometheusRuleName                   = "prometheus-rule"
	AlertManagerRouteName                = "alertmanager-route"
	GrafanaServiceMonitorName            = "grafana-servicemonitor"
)

type Parameters struct {
	ApplicationMonitoringName      string
	PrometheusOperatorName         string
	Namespace                      string
	GrafanaOperatorName            string
	GrafanaCrName                  string
	GrafanaImage                   string
	GrafanaOperatorRoleName        string
	GrafanaOperatorRoleBindingName string
	PrometheusCrName               string
	PrometheusRouteName            string
	PrometheusServiceName          string
	AlertManagerServiceAccountName string
	AlertManagerCrName             string
	AlertManagerServiceName        string
	AlertManagerRouteName          string
	GrafanaServiceMonitorName      string
	PrometheusServiceMonitorName   string
	MonitoringKey                  string
	ExtraParams                    map[string]string
}

type TemplateHelper struct {
	Parameters   Parameters
	TemplatePath string
}

// Creates a new templates helper and populates the values for all
// templates properties. Some of them (like the hostname) are set
// by the user in the custom resource
func newTemplateHelper(cr *applicationmonitoring.ApplicationMonitoring, extraParams map[string]string) *TemplateHelper {
	param := Parameters{
		Namespace:                      cr.Namespace,
		GrafanaOperatorName:            GrafanaOperatorName,
		GrafanaCrName:                  GrafanaCrName,
		GrafanaOperatorRoleBindingName: GrafanaOperatorRoleBindingName,
		GrafanaOperatorRoleName:        GrafanaOperatorRoleName,
		GrafanaImage:                   "quay.io/integreatly/grafana-operator:0.0.1",
		PrometheusOperatorName:         PrometheusOperatorName,
		ApplicationMonitoringName:      ApplicationMonitoringName,
		PrometheusCrName:               PrometheusCrName,
		PrometheusRouteName:            PrometheusRouteName,
		PrometheusServiceName:          PrometheusServiceName,
		AlertManagerServiceAccountName: AlertManagerServiceAccountName,
		AlertManagerCrName:             AlertManagerCrName,
		AlertManagerServiceName:        AlertManagerServiceName,
		AlertManagerRouteName:          AlertManagerRouteName,
		GrafanaServiceMonitorName:      GrafanaServiceMonitorName,
		PrometheusServiceMonitorName:   PrometheusServiceMonitorName,
		MonitoringKey:                  "middleware",
		ExtraParams:                    extraParams,
	}

	templatePath, exists := os.LookupEnv("TEMPLATE_PATH")
	if !exists {
		templatePath = "./templates"
	}

	monitoringKey, exists := os.LookupEnv("MONITORING_KEY")
	if exists {
		param.MonitoringKey = monitoringKey
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
