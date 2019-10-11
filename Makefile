ORG ?= integreatly
NAMESPACE ?= application-monitoring
PROJECT ?= application-monitoring-operator
REG=quay.io
SHELL=/bin/bash
TAG ?= 0.0.28
PKG=github.com/integr8ly/application-monitoring-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=application-monitoring-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)

.PHONY: setup/dep
setup/dep:
	@echo Installing dep
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	@echo setup complete

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

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)

.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/compile
	@operator-sdk build ${REG}/${ORG}/${PROJECT}:${TAG}

.PHONY: image/push
image/push:
	docker push ${REG}/${ORG}/${PROJECT}:${TAG}

.PHONY: image/build/push
image/build/push: image/build image/push

.PHONY: image/build/test
image/build/test:
	operator-sdk build --enable-tests ${REG}/${ORG}/${PROJECT}:${TAG}

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
	-kubectl delete -f ./deploy/roles
	-kubectl delete crd grafanas.integreatly.org
	-kubectl delete crd grafanadashboards.integreatly.org
	-kubectl delete crd grafanadatasources.integreatly.org
	-kubectl delete crd podmonitors.monitoring.coreos.com
	-kubectl delete crd prometheuses.monitoring.coreos.com
	-kubectl delete crd alertmanagers.monitoring.coreos.com
	-kubectl delete crd prometheusrules.monitoring.coreos.com
	-kubectl delete crd servicemonitors.monitoring.coreos.com
	-kubectl delete crd blackboxtargets.applicationmonitoring.integreatly.org
	-kubectl delete crd applicationmonitorings.applicationmonitoring.integreatly.org
	-kubectl delete namespace $(NAMESPACE)

.PHONY: cluster/create/examples
cluster/create/examples:
		-kubectl create -f deploy/examples/ApplicationMonitoring.yaml -n $(NAMESPACE)

.PHONY: cluster/install
cluster/install:
		./scripts/install.sh
