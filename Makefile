GO ?= go

GOLANGCI_LINT_VERSION = 2.13.1
GOVULNCHECK_VERSION = v1.7.0

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

BUILD_DIR := build
COVERAGE_DIR := $(BUILD_DIR)/coverage
GOLANGCI_LINT := $(BUILD_DIR)/golangci-lint
GINKGO := $(BUILD_DIR)/ginkgo
GO_MODIFF := $(BUILD_DIR)/go-modiff
GO_MODIFF_STATIC := $(BUILD_DIR)/go-modiff.static

COLOR := \033[36m
NOCOLOR := \033[0m

.PHONY: all
all: $(GO_MODIFF) ## Build the go-modiff binary

.PHONY: help
help: ## Display this help
	@awk \
		-v "col=$(COLOR)" -v "nocol=$(NOCOLOR)" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "\nUsage:\n  make %s<target>%s\n\n", col, nocol; \
			} \
			/^[a-zA-Z0-9_-]+:.*?##/ { \
				printf "  %s%-25s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", col, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

##@ Build

.PHONY: $(GO_MODIFF)
$(GO_MODIFF): ## Build the go-modiff binary
	$(GO) build -ldflags '-s -w -X main.version=$(VERSION)' \
		-o $(GO_MODIFF) ./cmd/go-modiff

.PHONY: $(GO_MODIFF_STATIC)
$(GO_MODIFF_STATIC): ## Build the static go-modiff binary
	CGO_ENABLED=0 $(GO) build -trimpath \
		-ldflags '-s -w -X main.version=$(VERSION)' \
		-o $(GO_MODIFF_STATIC) ./cmd/go-modiff

##@ Development

.PHONY: completions
completions: $(GO_MODIFF) ## Generate shell completions
	$(GO_MODIFF) fish > completions/go-modiff.fish

.PHONY: test
test: $(GINKGO) ## Run tests with coverage
	rm -rf $(COVERAGE_DIR) && mkdir -p $(COVERAGE_DIR)
	$(GINKGO) run $(TESTFLAGS) \
		-r -p \
		--cover \
		--randomize-all \
		--randomize-suites \
		--covermode atomic \
		--output-dir $(COVERAGE_DIR) \
		--coverprofile coverprofile \
		--junit-report junit.xml \
		--poll-progress-after 60s \
		--trace \
		--succinct
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverprofile -o $(COVERAGE_DIR)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverprofile

$(GINKGO):
	$(GO) build -o $(GINKGO) github.com/onsi/ginkgo/v2/ginkgo

##@ Verification

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	@mkdir -p $(BUILD_DIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BUILD_DIR) v$(GOLANGCI_LINT_VERSION)

.PHONY: verify-tidy
verify-tidy: ## Verify go.mod is tidy
	$(GO) mod tidy && $(GO) mod verify
	git diff --exit-code go.mod go.sum

.PHONY: verify-completions
verify-completions: completions ## Verify completions are up to date
	git diff --exit-code completions/

.PHONY: govulncheck
govulncheck: ## Run govulncheck
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

##@ Maintenance

.PHONY: tidy
tidy: ## Run go mod tidy
	$(GO) mod tidy && $(GO) mod verify

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
