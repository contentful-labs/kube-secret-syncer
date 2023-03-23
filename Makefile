.PHONY: examples
# Image URL to use all building/pushing image targets
IMG ?= contentful-labs/kube-secret-syncer
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1"
# Directory for storing generated manifests
OP_OUT ?= out
# kind cluster context
KIND_CTX ?= kubernetes-admin@kind
# AWS Profile credentials to pass to kind cluster
AWS_KIND_PROFILE ?= preview

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# BSD vs GNU sed, fight!
PLATFORM := $(shell uname)
ifeq ($(PLATFORM),Linux)
	SED_I=sed -i
else
	SED_I=sed -i ''
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test -v ./... -coverprofile cover.out -coverpkg ./controllers/...,./pkg/...

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

operator: 
	@rm -rf ${OP_OUT}
	@mkdir -p ${OP_OUT}
	@kustomize build config/default -o ${OP_OUT}/
	@find ${OP_OUT} -type f -name "*.yaml" -print0 | xargs -0 ${SED_I} '/^  creationTimestamp: null/d'
	@echo "built operators in ${OP_OUT}"

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Run tests in a container
docker-test:
	docker build . -t ${IMG}-test --target=test
	docker run -t -v $(PWD):/repo --rm ${IMG}-test go test -v ./... -coverprofile /repo/cover.out -coverpkg ./controllers/...,./pkg/...

# Build the docker image
docker-build: 
	docker build . -t ${IMG}

kind:
	docker tag contentful-labs/kube-secret-syncer:latest kube-secret-syncer:kind
	kind load docker-image kube-secret-syncer:kind
	@kubectl --context=${KIND_CTX} apply -f config/samples/kube-secret-syncer-ns.yaml
	@kubectl --context=${KIND_CTX} -n kube-secret-syncer delete --ignore-not-found configmap aws-creds
	@kubectl --context=${KIND_CTX} -n kube-secret-syncer create configmap aws-creds \
	--from-literal=AWS_ACCESS_KEY_ID=$(shell aws configure get aws_access_key_id --profile ${AWS_KIND_PROFILE}) \
	--from-literal=AWS_SECRET_ACCESS_KEY=$(shell aws configure get aws_secret_access_key --profile ${AWS_KIND_PROFILE}) \
	--from-literal=AWS_SESSION_TOKEN=$(shell aws configure get aws_session_token --profile ${AWS_KIND_PROFILE})

	@kubectl --context=${KIND_CTX} -n kube-secret-syncer delete deployment kube-secret-syncer-controller --ignore-not-found=true
	kustomize build config/overlays/kind | kubectl apply --context=${KIND_CTX} -f -

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

examples:
	@rm -rf examples/*
	@kustomize build config/overlays/examples/ -o examples/
