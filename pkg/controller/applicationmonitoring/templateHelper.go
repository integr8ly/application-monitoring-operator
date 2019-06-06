package applicationmonitoring

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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
	GrafanaDataSourceName                = "grafana-datasource"
	GrafanaRouteName                     = "grafana-route"
	GrafanaProxySecretName               = "grafana-proxy-secret"
	PrometheusOperatorName               = "prometheus-operator"
	PrometheusOperatorServiceAccountName = "prometheus-operator-service-account"
	PrometheusCrName                     = "prometheus"
	PrometheusRouteName                  = "prometheus-route"
	PrometheusProxySecretsName           = "prometheus-proxy-secret"
	PrometheusServiceAccountName         = "prometheus-service-account"
	PrometheusServiceName                = "prometheus-service"
	AlertManagerProxySecretsName         = "alertmanager-proxy-secret"
	AlertManagerServiceAccountName       = "alertmanager-service-account"
	AlertManagerCrName                   = "alertmanager"
	AlertManagerServiceName              = "alertmanager-service"
	AlertManagerSecretName               = "alertmanager-secret"
	PrometheusServiceMonitorName         = "prometheus-servicemonitor"
	PrometheusRuleName                   = "prometheus-rule"
	AlertManagerRouteName                = "alertmanager-route"
	GrafanaServiceMonitorName            = "grafana-servicemonitor"
	GrafanaServiceName                   = "grafana-service"
	BlackboxExporterConfigmapName        = "blackbox-exporter-config"
	BlackboxExporterJobName              = "blackbox-exporter-job"
	ScrapeConfigSecretName               = "additional-scrape-configs"
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
	GrafanaSessionSecret           string
	GrafanaProxySecretName         string
	GrafanaRouteName               string
	PrometheusCrName               string
	PrometheusRouteName            string
	PrometheusServiceName          string
	PrometheusSessionSecret        string
	AlertManagerSessionSecret      string
	AlertManagerServiceAccountName string
	AlertManagerCrName             string
	AlertManagerServiceName        string
	AlertManagerRouteName          string
	GrafanaServiceMonitorName      string
	PrometheusServiceMonitorName   string
	MonitoringKey                  string
	GrafanaServiceName             string
	BlackboxExporterConfigmapName  string
	ScrapeConfigSecretName         string
	BlackboxTargets                []string
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
		GrafanaImage:                   "quay.io/integreatly/grafana-operator:v1.0.0",
		GrafanaProxySecretName:         GrafanaProxySecretName,
		GrafanaServiceName:             GrafanaServiceName,
		GrafanaRouteName:               GrafanaRouteName,
		PrometheusOperatorName:         PrometheusOperatorName,
		ApplicationMonitoringName:      ApplicationMonitoringName,
		PrometheusCrName:               PrometheusCrName,
		PrometheusRouteName:            PrometheusRouteName,
		PrometheusServiceName:          PrometheusServiceName,
		PrometheusSessionSecret:        PopulateSessionProxySecret(),
		AlertManagerSessionSecret:      PopulateSessionProxySecret(),
		GrafanaSessionSecret:           PopulateSessionProxySecret(),
		AlertManagerServiceAccountName: AlertManagerServiceAccountName,
		AlertManagerCrName:             AlertManagerCrName,
		AlertManagerServiceName:        AlertManagerServiceName,
		AlertManagerRouteName:          AlertManagerRouteName,
		GrafanaServiceMonitorName:      GrafanaServiceMonitorName,
		PrometheusServiceMonitorName:   PrometheusServiceMonitorName,
		MonitoringKey:                  cr.Spec.LabelSelector,
		BlackboxExporterConfigmapName:  BlackboxExporterConfigmapName,
		BlackboxTargets:                cr.Spec.BlackboxTargets,
		ScrapeConfigSecretName:         ScrapeConfigSecretName,
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

// Populate the PrometheusServiceName values
func PopulateSessionProxySecret() string {
	p, err := GeneratePassword(43)
	if err != nil {
		log.Info("Error executing PopulateSessionProxySecret")
	}
	return p
}

// Takes a list of strings, wraps each string in double quotes and joins them
// Used for building yaml arrays
func joinQuote(values []string) string {
	var result []string
	for _, s := range values {
		result = append(result, fmt.Sprintf("\"%s\"", s))
	}
	return strings.Join(result, ", ")
}

// load a templates from a given resource name. The templates must be located
// under ./templates and the filename must be <resource-name>.yaml
func (h *TemplateHelper) loadTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s.yaml", h.TemplatePath, name)
	tpl, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := template.New("application-monitoring")
	parser.Funcs(template.FuncMap{
		"JoinQuote": joinQuote,
	})

	parsed, err := parser.Parse(string(tpl))
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

// GeneratePassword returns a base64 encoded securely random bytes.
func GeneratePassword(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), err
}
