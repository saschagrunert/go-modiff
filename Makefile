GO ?= go

# test for go module support
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
export GO_BUILD=GO111MODULE=on $(GO) build -mod=vendor
else
export GO_BUILD=$(GO) build
endif

BUILD_PATH := build
GOLANGCI_LINT := $(BUILD_PATH)/golangci-lint
GO_MODIFF := go-modiff

define go-build
	$(shell cd `pwd` && $(GO) build -ldflags '-s -w' \
		-o $(BUILD_PATH)/$(shell basename $(1)) $(1))
	@echo > /dev/null
endef

all: $(GO_MODIFF)

.PHONY: clean
clean:
	git clean -fdx

.PHONY: docs
docs: $(GO_MODIFF)
	$(GO_MODIFF) d --markdown > docs/go-modiff.8.md
	$(GO_MODIFF) d --man > docs/go-modiff.8

.PHONY: $(GO_MODIFF)
$(GO_MODIFF):
	$(call go-build,./cmd/go-modiff)

$(GOLANGCI_LINT):
	$(call go-build,./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify
