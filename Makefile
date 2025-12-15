# Makefile for YaSwag - Yet Another Swagger Tool for Go

GOCMD:=$(shell which go)
GOVET:=$(GOCMD) vet
GOFMT:=$(GOCMD) fmt
GOTEST:=$(GOCMD) test
GOBUILD:=$(GOCMD) build
GITCMD:=$(shell which git)
GOLANGCI_LINT:=$(shell which golangci-lint)
STATICCHECK:=$(shell which staticcheck)

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD)
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build test fmt vet gocyclo lint clean release release-push release-snapshot release-check

all: build

build:
	@echo "Building YaSwag..."
	@mkdir -p ./bin
	@$(GOBUILD) $(LDFLAGS) -o ./bin/yaswag ./cmd/yaswag
	@echo "Build complete. Binary located at ./bin/yaswag"

test:
	@echo "Running tests..."
	@$(GOTEST) ./...
	@echo "Tests complete."

fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted."

vet:
	@echo "Running go vet..."
	@$(GOVET) ./...
	@echo "go vet complete."

gocyclo:
	@echo "Calculating cyclomatic complexity..."
	@which gocyclo > /dev/null || (echo "gocyclo not found! Please install it." && exit 1)
	@gocyclo -over 10 .
	@echo "Cyclomatic complexity calculation complete."

lint:
	@echo "Running linter..."
	@if [ -x "$(GOLANGCI_LINT)" ]; then \
		$(GOLANGCI_LINT) run; \
	elif [ -x "$(STATICCHECK)" ]; then \
		$(STATICCHECK) ./...; \
	else \
		echo "No linter found! Please install golangci-lint or staticcheck."; \
		exit 1; \
	fi
	@echo "Linting complete."

clean:
	@echo "Cleaning up..."
	@rm -rf ./bin
	@echo "Clean complete."

release-test: lint test gocyclo fmt vet release-check release-snapshot

release:
	@echo "Starting release process..."
	@goreleaser build --clean
	@echo "Release build complete, pushing release..."
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "GITHUB_TOKEN is not set! Please set it to proceed with the release."; \
		exit 1; \
	fi
	@goreleaser release --clean
	@echo "Release process complete."

release-snapshot:
	@echo "Preparing for release..."
	@goreleaser build --clean
	@echo "Release preparation complete."
	@echo "Creating snapshot release..."
	@goreleaser release --snapshot --clean
	@echo "Snapshot release complete."

release-check:
	@echo "Checking release configuration..."
	@goreleaser check
	@goreleaser healthcheck
	@echo "Release configuration is valid."