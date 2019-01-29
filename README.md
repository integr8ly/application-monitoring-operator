# Application Monitoring Operator

A Kubernetes Operator based on the Operator SDK that installs the Integr8ly Application Monitoring Stack.

# Current status

This is a PoC / alpha version. Most functionality is there but it is highly likely there are bugs and improvements needed.

# Supported Custom Resources

The following Grafana resources are supported:

* ApplicationMonitoring

## ApplicationMonitoring

Triggers the installation of the monitoring stack when created. This is achieved by deploying two other operators that install Prometheus and Grafana respectively.

# Running locally

You can run the Operator locally against a remote namespace. The name of the namespace should be `application-monitoring`. To run the operator execute:

```sh
$ make setup/dep
$ make code/run
```