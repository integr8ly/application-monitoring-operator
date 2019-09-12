# Application Monitoring Operator

An OpenShift Operator using the Operator SDK that installs an Application Monitoring Stack consisting of Grafana, Prometheus & Alertmanager.

# Supported Custom Resources

The following resources are supported:

## ApplicationMonitoring
Triggers the installation of the monitoring stack when created. This is achieved by deploying two other operators that install Prometheus and Grafana respectively.

The Application Monitoring CR accepts the following properties in the spec:

* *labelSelector*: The value of the `middleware-monitoring` label that has to be present on all imported resources (prometheus rules, service monitors, grafana dashboards).
* *prometheusRetention*: Retention time for prometheus data. See https://prometheus.io/docs/prometheus/latest/storage/
* *prometheusStorageRequest*: How much storage to assign to a volume claim for persisting Prometheus data. See https://github.com/coreos/prometheus-operator/blob/ca400fdc3edd0af0df896a338eca270e115b74d7/Documentation/api.md#storagespec



## BlackboxTarget
The Blackbox Target CR accepts the following properties in the spec:

* *blackboxTargets*: A list of targets for the blackbox exporter to probe.

The `blackboxTargets` should be provided as an array in the form of:

```yaml
  blackboxTargets:
    - service: example
      url: https://example.com
      module: http_extern_2xx
```

where `service` will be added as a label to the metric, `url` is the URL of the route to probe and `module` can be one of:

* *http_2xx*: Probe http or https targets via GET using the cluster certificates
* *http_post_2xx*: Probe http or https targets via POST using the cluster certificates
* *http_extern_2xx*: Probe http or https targets via GET relying on a valid external certificate

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

```bash
make cluster/install
```
You can access Grafana, Prometheus & AlertManager web consoles using the Routes in the project.

## Verify installation
Run the following commands
```bash
# Check the project exists
$ oc project
Using project "application-monitoring" on server "https://your-cluster-ip:8443"

# Check the pods exist e.g.
$ oc get pods
NAME                                              READY     STATUS    RESTARTS   AGE
alertmanager-application-monitoring-0             2/2       Running   0          1h
application-monitoring-operator-77cdbcbff-fbrnr   1/1       Running   0          1h
grafana-deployment-6dc8df6bb4-rxdjs               1/1       Running   0          49m
grafana-operator-7c4869cfdc-6sdv9                 1/1       Running   0          1h
prometheus-application-monitoring-0               4/4       Running   1          36m
prometheus-operator-7547bb757b-46lwh              1/1       Running   0          1h
``` 

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

# Releasing

1. Bump the version number (e.g. `0.0.29`) (in more files [Makefile](https://github.com/integr8ly/application-monitoring-operator/blob/master/Makefile#L6), [deploy/operator.yaml](https://github.com/integr8ly/application-monitoring-operator/blob/master/deploy/operator.yaml), [version/version.go](https://github.com/integr8ly/application-monitoring-operator/blob/master/version/version.go#L4]))
2. Create pull request with the bump, let it be approved and merge it to `master`
3. Pull the master locally and tag it with a tag named as the version number 
```sh
$ git pull origin master
$ git tag 0.0.29
$ git push origin --tags
```
4. Create *Release* through github release page, name it as the version number - `0.0.29` and state what's new in the release
5. Login to [quay.io](https://quay.io) with `docker login quay.io`
6. Build docker image for the new version and push it to the `quay.io`
```sh
$ make image/build
$ make image/push
```
6. In [Integr8ly installer](https://github.com/integr8ly/installation/pulls) `fh-automatron` automatically creates pull request named `middleware-monitoring update` with the necessary modifications of the version for the new release 
7. Approve and merge the PR