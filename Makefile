GO ?= go

BUILD_PATH := build
COVERAGE_PATH := $(BUILD_PATH)/coverage

GOLANGCI_LINT := $(BUILD_PATH)/golangci-lint
GO_MODIFF := $(BUILD_PATH)/go-modiff
GO_MODIFF_STATIC := $(BUILD_PATH)/go-modiff.static
GINKGO := $(BUILD_PATH)/ginkgo

define go-build
	cd `pwd` && $(GO) build -ldflags '-s -w $(2)' \
		-o $(BUILD_PATH)/$(shell basename $(1)) $(1)
	@echo > /dev/null
endef

all: $(GO_MODIFF)

.PHONY: clean
clean:
	rm -rf $(BUILD_PATH)

.PHONY: codecov
codecov: SHELL := $(shell which bash)
codecov:
	bash <(curl -s https://codecov.io/bash) -f $(COVERAGE_PATH)/coverprofile

.PHONY: docs
docs: $(GO_MODIFF)
	$(GO_MODIFF) d --markdown > docs/go-modiff.8.md
	$(GO_MODIFF) d --man > docs/go-modiff.8
	$(GO_MODIFF) f > completions/go-modiff.fish

.PHONY: $(GO_MODIFF)
$(GO_MODIFF):
	$(call go-build,./cmd/go-modiff)

.PHONY: $(GO_MODIFF_STATIC)
$(GO_MODIFF_STATIC):
	$(call go-build,./cmd/go-modiff,-linkmode external -extldflags "-static -lm")

$(GOLANGCI_LINT):
	export \
		VERSION=v1.52.2 \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=$(BUILD_PATH) && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION

$(GINKGO):
	$(call go-build,./vendor/github.com/onsi/ginkgo/v2/ginkgo)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) linters
	GL_DEBUG=gocritic $(GOLANGCI_LINT) run

.PHONY: test
test: $(GINKGO)
	rm -rf $(COVERAGE_PATH) && mkdir -p $(COVERAGE_PATH)
	$(BUILD_PATH)/ginkgo run $(TESTFLAGS) \
		-r -p \
		--cover \
		--mod vendor \
		--randomize-all \
		--randomize-suites \
		--covermode atomic \
		--output-dir $(COVERAGE_PATH) \
		--coverprofile coverprofile \
		--junit-report junit.xml \
		--slow-spec-threshold 60s \
		--trace \
		--succinct
	$(GO) tool cover -html=$(COVERAGE_PATH)/coverprofile -o $(COVERAGE_PATH)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_PATH)/coverprofile

.PHONY: vendor
vendor:
	export GO111MODULE=on GOSUMDB= && \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify
