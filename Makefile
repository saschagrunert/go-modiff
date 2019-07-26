export GO111MODULE=off

GO ?= go

BUILD_PATH := $(shell pwd)/build
BUILD_BIN_PATH := ${BUILD_PATH}/bin

GOLANGCI_LINT := ${BUILD_BIN_PATH}/golangci-lint
GO_MODIFF := ${BUILD_BIN_PATH}/go-modiff

define go-build
	$(shell cd `pwd` && ${GO} build -ldflags '-s -w' \
		-o ${BUILD_BIN_PATH}/$(shell basename ${1}) ${1})
	@echo > /dev/null
endef

all: ${GO_MODIFF}

.PHONY: clean
clean:
	git clean -fdx

.PHONY: docs
docs: ${GO_MODIFF}
	${GO_MODIFF} d --markdown > docs/go-modiff.8.md
	${GO_MODIFF} d --man > docs/go-modiff.8

${GO_MODIFF}:
	$(call go-build,./cmd/go-modiff)

${GOLANGCI_LINT}:
	$(call go-build,./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint)

.PHONY: lint
lint: ${GOLANGCI_LINT}
	${GOLANGCI_LINT} run

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify
