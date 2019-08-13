#!/bin/bash

# create the project and add a label
oc new-project application-monitoring
oc label namespace application-monitoring monitoring-key=middleware

# Grafana CRDs
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/Grafana.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDashboard.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDataSource.yaml
oc apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/podmonitor.crd.yaml
oc apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/prometheus.crd.yaml
oc apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/alertmanager.crd.yaml
oc apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/prometheusrule.crd.yaml
oc apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/example/prometheus-operator-crd/servicemonitor.crd.yaml

# BlackboxTarget
oc apply -f ./deploy/crds/BlackboxTarget.yaml

# Application Monitoring Operator
# AMO CRD
oc apply -f ./deploy/crds/ApplicationMonitoring.yaml

# Cluster Roles & RoleBindings
oc apply -f ./deploy/roles
oc apply -f ./deploy/operator_roles/service_account.yaml
oc apply -f ./deploy/operator_roles/role.yaml
oc apply -f ./deploy/operator_roles/role_binding.yaml

# AMO Deployment
until oc auth can-i create prometheus -n application-monitoring --as system:serviceaccount:application-monitoring:application-monitoring-operator; do
    echo "Waiting for all CRDs and SA permissions to be applied before deploying operator..." && sleep 1
done

oc apply -f ./deploy/operator.yaml
oc apply -f ./deploy/examples/ApplicationMonitoring.yaml
