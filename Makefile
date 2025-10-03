GOLANGCI_LINT_VERSION ?= v1.64.6
OSNAME ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

.PHONY: mockgen
mockgen:
	go generate ./...

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --allow-parallel-runners

.PHONY: unit_test
unit_test:
	go test -parallel 6 -race -count=1 -coverpkg=./... -coverprofile=unit_coverage.out -v `go list ./... | grep -v /test/`

.PHONY: integration_test
integration_test:
	go test -count=1 -v --tags=integration -coverpkg=./... -coverprofile=int_coverage.out ./test/...
