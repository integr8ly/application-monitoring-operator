#!/bin/bash

# create the project and add a label
oc new-project application-monitoring
oc label namespace application-monitoring monitoring-key=middleware

# Grafana CRDs
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/Grafana.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDashboard.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDataSource.yaml

# BlackboxTarget
oc apply -f ./deploy/crds/BlackboxTarget.yaml

# Application Monitoring Operator
# AMO CRD
oc apply -f ./deploy/crds/ApplicationMonitoring.yaml

# Cluster Roles & RoleBindings
oc apply -f ./deploy/roles
oc apply -f ./deploy/operator_roles/

# AMO Deployment
oc apply -f ./deploy/operator.yaml
oc apply -f ./deploy/examples/ApplicationMonitoring.yaml
