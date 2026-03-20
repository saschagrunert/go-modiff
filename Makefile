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

.PHONY: completions
completions: $(GO_MODIFF)
	$(GO_MODIFF) fish > completions/go-modiff.fish

.PHONY: $(GO_MODIFF)
$(GO_MODIFF):
	$(call go-build,./cmd/go-modiff)

.PHONY: $(GO_MODIFF_STATIC)
$(GO_MODIFF_STATIC):
	$(call go-build,./cmd/go-modiff,-linkmode external -extldflags "-static -lm")

$(GOLANGCI_LINT):
	export \
		VERSION=v2.11.3 \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=$(BUILD_PATH) && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION

$(GINKGO):
	$(GO) build -o $(BUILD_PATH)/ginkgo github.com/onsi/ginkgo/v2/ginkgo

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

.PHONY: tidy
tidy:
	$(GO) mod tidy && $(GO) mod verify
