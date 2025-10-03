GOLANGCI_LINT_VERSION ?= v1.64.6
OSNAME ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --allow-parallel-runners