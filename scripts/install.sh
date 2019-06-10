#!/bin/bash

# create the project and add a label
oc new-project application-monitoring
oc label namespace application-monitoring monitoring-key=middleware

# Grafana CRDs
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/Grafana.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDashboard.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDataSource.yaml

# Cluster Roles & RoleBindings
oc apply -f ./deploy/roles

# Application Monitoring Operator
oc apply -f ./deploy/operator_roles/
oc apply -f ./deploy/crds/ApplicationMonitoring.yaml
oc apply -f ./deploy/operator.yaml
oc apply -f ./deploy/examples/ApplicationMonitoring.yaml
