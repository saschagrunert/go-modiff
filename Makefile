GO ?= go

# test for go module support
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
export GO_BUILD=GO111MODULE=on $(GO) build -mod=vendor
else
export GO_BUILD=$(GO) build
endif

BUILD_PATH := build
COVERAGE_PATH := $(BUILD_PATH)/coverage
JUNIT_PATH := $(BUILD_PATH)/junit

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
	$(call go-build,./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint)

$(GINKGO):
	$(call go-build,./vendor/github.com/onsi/ginkgo/ginkgo)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

.PHONY: test
test: $(GINKGO)
	rm -rf $(COVERAGE_PATH) && mkdir -p $(COVERAGE_PATH)
	rm -rf $(JUNIT_PATH) && mkdir -p $(JUNIT_PATH)
	$(BUILD_PATH)/ginkgo $(TESTFLAGS) \
		-r -p \
		--cover \
		--randomizeAllSpecs \
		--randomizeSuites \
		--covermode atomic \
		--outputdir $(COVERAGE_PATH) \
		--coverprofile coverprofile \
		--slowSpecThreshold 60 \
		--succinct
	# fixes https://github.com/onsi/ginkgo/issues/518
	sed -i '2,$${/mode: atomic/d;}' $(COVERAGE_PATH)/coverprofile
	$(GO) tool cover -html=$(COVERAGE_PATH)/coverprofile -o $(COVERAGE_PATH)/coverage.html
	$(GO) tool cover -func=$(COVERAGE_PATH)/coverprofile | sed -n 's/\(total:\).*\([0-9][0-9].[0-9]\)/\1 \2/p'
	find . -name '*_junit.xml' -exec mv -t $(JUNIT_PATH) {} +

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify
