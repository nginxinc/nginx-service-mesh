GOLANGCI_LINT_VERSION ?= v1.51-alpine

OUTPUT_DIR = $(shell pwd)/build
VERSION ?= $(shell git describe --tags)

GIT_COMMIT = $(shell git rev-parse HEAD)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
    EXTENSION=.exe
endif

.DEFAULT_GOAL := help

.PHONY: help
help: Makefile ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "; printf "Usage:\n\n    make \033[36m<target>\033[0m\n\nTargets:\n\n"}; {printf "    \033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: deps generate test lint format build

.PHONY: generate
generate: ## Generate Go code for apis and fakes
	go generate ./...
	go run sigs.k8s.io/controller-tools/cmd/controller-gen object paths=./pkg/apis/...
	go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest -package mesh ./api/mesh.yaml > ./pkg/apis/mesh/mesh.gen.go
	scripts/generateSchema.sh api/upstream-ca-validation.json > internal/nginx-meshctl/upstreamauthority/schema.go
	$(MAKE) format

.PHONY: output-dir
output-dir:
	mkdir -p $(OUTPUT_DIR)

.PHONY: build build.cli

linker_flags = -s -w -extldflags "-fno-PIC -static" -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT)

build: build-cli ## Build all Go binaries

build-cli: output-dir ## Build the nginx-meshctl binary
	CGO_ENABLED=0 go build -ldflags '$(linker_flags) -X main.pkgName=nginx-meshctl' -o $(OUTPUT_DIR)/$(GOOS)-$(GOARCH)/nginx-meshctl$(EXTENSION) cmd/nginx-meshctl/main.go

.PHONY: test
test: output-dir ## Run unit tests for the Go code
	go test ./... -race -coverprofile=$(OUTPUT_DIR)/coverage.out
	go tool cover -func=$(OUTPUT_DIR)/coverage.out
	go tool cover -html=$(OUTPUT_DIR)/coverage.out -o $(OUTPUT_DIR)/coverage.html

.PHONY: lint
lint: ## Run golangci-lint against code
	docker run --rm -v $(shell pwd):/nginx-service-mesh -w /nginx-service-mesh -v $(shell go env GOCACHE):/cache/go -e GOCACHE=/cache/go -e GOLANGCI_LINT_CACHE=/cache/go golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint --color always run

.PHONY: format
format: ## Run go fmt against code
	go fmt ./...

.PHONY: deps
deps: ## Add missing and remove unused modules, verify deps, and download them to local cache
	go mod tidy
	go mod verify
	go mod download

.PHONY: clean
clean: ## Clean the build
	rm -rf $(OUTPUT_DIR)

.PHONY: clean-go-cache
clean-go-cache: ## Clean Go cache
	go clean -cache -modcache -testcache -i -r
