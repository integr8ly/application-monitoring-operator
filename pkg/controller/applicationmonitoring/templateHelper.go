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
	GrafanaOperatorName               = "grafana-operator"
	GrafanaRoleName                   = "grafana-role"
	GrafanaRoleBindingName            = "grafana-role-binding"
	GrafanaServiceAccountName         = "grafana-service-account"
	GrafanaCrName                     = "grafana"
	GrafanaOperatorServiceAccountName = "grafana-operator-service-account"
	GrafanaOperatorRoleName           = "grafana-operator-role"
	GrafanaOperatorRoleBindingName    = "grafana-operator-role-binding"
)

type Parameters struct {
	PrometheusOperatorDeploymentName  string
	Namespace                         string
	GrafanaOperatorName               string
	GrafanaRoleName                   string
	GrafanaRoleBindingName            string
	GrafanaServiceAccountName         string
	GrafanaCrName                     string
	GrafanaImage                      string
	GrafanaOperatorServiceAccountName string
	GrafanaOperatorRoleName           string
	GrafanaOperatorRoleBindingName    string
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
		Namespace:                         cr.Namespace,
		GrafanaOperatorName:               GrafanaOperatorName,
		GrafanaRoleName:                   GrafanaRoleName,
		GrafanaRoleBindingName:            GrafanaRoleBindingName,
		GrafanaServiceAccountName:         GrafanaServiceAccountName,
		GrafanaCrName:                     GrafanaCrName,
		GrafanaOperatorServiceAccountName: GrafanaOperatorServiceAccountName,
		GrafanaOperatorRoleBindingName:    GrafanaOperatorRoleBindingName,
		GrafanaOperatorRoleName:           GrafanaOperatorRoleName,
		GrafanaImage:                      "quay.io/integreatly/grafana-operator:0.0.1",
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
