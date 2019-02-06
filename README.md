# Application Monitoring Operator

A Kubernetes Operator based on the Operator SDK that installs the Integr8ly Application Monitoring Stack.

# Current status

This is a PoC / alpha version. Most functionality is there but it is highly likely there are bugs and improvements needed.

# Supported Custom Resources

The following resources are supported:

## ApplicationMonitoring

Triggers the installation of the monitoring stack when created. This is achieved by deploying two other operators that install Prometheus and Grafana respectively.

## PrometheusRule

Represents a set of alert rules for Prometheus/Alertmanager. See the [https://github.com/coreos/prometheus-operator/blob/f9bc0aa0fd9aa936f500d9d241098863c60d873d/Documentation/user-guides/alerting.md#alerting](prometheus operator docs) for more details about this resource.
An example PrometheusRule can be seen in the example app [template](https://github.com/david-martin/example-prometheus-nodejs/blob/d647b83116519b650e00401f04c8868280c47778/template.yaml#L92-L111)


## ServiceMonitor

Represents a Service to pull metrics from. See the [https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md#related-resources](prometheus operator docs) for more details about this resource.
An example ServiceMonitor can be seen in the example app [template](https://github.com/david-martin/example-prometheus-nodejs/blob/d647b83116519b650e00401f04c8868280c47778/template.yaml#L79-L91)

## GrafanaDashboard

Represents a Grafana dashboard. You typically create this in the namespace of the service the dashboard is associated with.
The Grafana operator reconciles this resource into a dashboard.
An example GrafanaDashboard can be seen in the example app [template](https://github.com/david-martin/example-prometheus-nodejs/blob/d647b83116519b650e00401f04c8868280c47778/template.yaml#L112-L734)


# Installation

You will need cluster admin permissions to create CRDs, ClusterRoles & ClusterRoleBindings.
ClusterRoles are needed to allow the operators to watch multiple namespaces.

```
oc new-project application-monitoring
oc label namespace application-monitoring monitoring-key=middleware
```

Grafana CRDs

```
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/Grafana.yaml
oc apply -f https://raw.githubusercontent.com/integr8ly/grafana-operator/master/deploy/crds/GrafanaDashboard.yaml
```

Cluster Roles & RoleBindings

```
oc apply -f ./deploy/roles
```

Application Monitoring Operator

```
oc apply -f ./deploy/operator_roles/
oc apply -f ./deploy/crds/ApplicationMonitoring.yaml
oc apply -f ./deploy/operator.yaml
oc apply -f ./deploy/examples/ApplicationMonitoring.yaml
```

You can access Grafana, Prometheus & AlertManager web consoles using the Routes in the project.

# Example Monitored Project

These steps create a new project with a simple server exposing a metrics endpoint.
The template also includes a simple PrometheusRule, ServiceMonitor & GrafanaDashboard custom resource that the application-monitoring stack detects and reconciles.

```
oc new-project example-prometheus-nodejs
oc label namespace example-prometheus-nodejs monitoring-key=middleware
oc process -f https://raw.githubusercontent.com/david-martin/example-prometheus-nodejs/master/template.yaml | oc create -f -
```

You should see the following things once reconciled:

* New Grafana Dashboard showing memory usage of the app (among other things)
* A new Target `example-prometheus-nodejs/example-prometheus-nodejs` in Prometheus
* A new Alert Rule `APIHighMedianResponseTime`

The example application provides three endpoints that will produce more metrics:

* `/` will return Hello World after a random response time
* `/checkout` Will create random checkout metrics
* `/bad` Can be used to create error rate metrics

# Running locally (for development)

You can run the Operator locally against a remote namespace. The name of the namespace should be `application-monitoring`. To run the operator execute:

```sh
$ make setup/dep
$ make code/run
```

