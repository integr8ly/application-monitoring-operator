ORG ?= integreatly
NAMESPACE ?= application-monitoring
PROJECT ?= application-monitoring-operator
REG=quay.io
SHELL=/bin/bash
PKG=github.com/integr8ly/application-monitoring-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=application-monitoring-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
# PROMETHEUS_OPERATOR_VERSION is used at install time to import crds
# After v0.34.0 the file names for the resources change
# If you are updating this verion you will need to update the file names in ./scripts/install.sh too
# You can delete this comment afterwards.
PROMETHEUS_OPERATOR_VERSION=v0.34.0
LOCAL=local
GRAFANA_OPERATOR_VERSION=v3.2.0
AMO_VERSION?=v1.1.6
PREV_AMO_VERSION=v1.1.5

AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "${QUAY_PASSWORD}"}}' | jq -r '.token')


.PHONY: setup/gomod
setup/gomod:
	@echo Running go.mod tidy
	@go mod tidy
	@echo Running go.mod vendor
	@go mod vendor

.PHONY: setup/travis
setup/travis:
	@echo Installing Operator SDK
	@curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/v0.8.1/operator-sdk-v0.8.1-x86_64-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/

.PHONY: code/run
code/run:
	@operator-sdk up local --namespace=${NAMESPACE}

.PHONY: code/compile
code/compile:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) ./cmd/manager

.PHONY: code/gen
code/gen:
	operator-sdk generate k8s

.PHONY: gen/csv
gen/csv:
	sed -i.bak 's/image:.*/image: quay\.io\/integreatly\/application-monitoring-operator:v$(AMO_VERSION)/g' deploy/operator.yaml && rm deploy/operator.yaml.bak
	@operator-sdk olm-catalog gen-csv --operator-name=application-monitoring-operator --csv-version $(AMO_VERSION) --from-version $(PREV_AMO_VERSION) --update-crds --csv-channel=integreatly --default-channel
	@sed -i.bak 's/$(PREV_AMO_VERSION)/$(AMO_VERSION)/g' deploy/olm-catalog/application-monitoring-operator/application-monitoring-operator.package.yaml && rm deploy/olm-catalog/application-monitoring-operator/application-monitoring-operator.package.yaml.bak
	@sed -i.bak s/application-monitoring-operator:v$(PREV_AMO_VERSION)/application-monitoring-operator:v$(AMO_VERSION)/g deploy/olm-catalog/application-monitoring-operator/$(AMO_VERSION)/application-monitoring-operator.v$(AMO_VERSION).clusterserviceversion.yaml && rm deploy/olm-catalog/application-monitoring-operator/$(AMO_VERSION)/application-monitoring-operator.v$(AMO_VERSION).clusterserviceversion.yaml.bak

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/compile
	@operator-sdk build ${REG}/${ORG}/${PROJECT}:${AMO_VERSION}

.PHONY: image/push
image/push:
	docker push ${REG}/${ORG}/${PROJECT}:${AMO_VERSION}

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: image/build/test
image/build/test:
	operator-sdk build --enable-tests ${REG}/${ORG}/${PROJECT}:${AMO_VERSION}

.PHONY: test/unit
test/unit:
	@echo Running tests:
	go test -v -race -cover ./pkg/...

.PHONY: test/e2e
test/e2e:
	kubectl apply -f deploy/test-e2e-pod.yaml -n ${PROJECT}
	${SHELL} ./scripts/stream-pod ${TEST_POD_NAME} ${PROJECT}

.PHONY: cluster/prepare
cluster/prepare:
	-kubectl apply -f deploy/crds/
	-oc new-project $(NAMESPACE)

.PHONY: cluster/clean
cluster/clean:
	-kubectl delete -n $(NAMESPACE) --all blackboxtargets
	-kubectl delete -n $(NAMESPACE) --all grafanadashboards
	-kubectl delete -n $(NAMESPACE) --all grafanadatasources
	-kubectl delete -n $(NAMESPACE) --all applicationmonitorings
	-kubectl delete -f ./deploy/cluster-roles
	-kubectl delete crd grafanas.integreatly.org
	-kubectl delete crd grafanadashboards.integreatly.org
	-kubectl delete crd grafanadatasources.integreatly.org
	-kubectl delete crd blackboxtargets.applicationmonitoring.integreatly.org
	-kubectl delete crd applicationmonitorings.applicationmonitoring.integreatly.org
	-kubectl delete namespace $(NAMESPACE)

.PHONY: cluster/create/examples
cluster/create/examples:
	-kubectl create -f deploy/examples/ApplicationMonitoring.yaml -n $(NAMESPACE)

.PHONY: cluster/install
cluster/install:
	./scripts/install.sh  ${PROMETHEUS_OPERATOR_VERSION} ${GRAFANA_OPERATOR_VERSION}

.PHONY: cluster/install/local
cluster/install/local:
	./scripts/install.sh  ${PROMETHEUS_OPERATOR_VERSION} ${GRAFANA_OPERATOR_VERSION} ${LOCAL}

.PHONY: manifest/push
manifest/push:
	@operator-courier --verbose push deploy/olm-catalog/application-monitoring-operator/ $(ORG) $(PROJECT) $(AMO_VERSION) "$(AUTH_TOKEN)"
