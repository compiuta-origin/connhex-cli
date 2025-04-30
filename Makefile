BUILD_DIR = build

GOARCH ?= $(shell uname -m)
CGO_ENABLED ?= 0
GOOS ?= $(shell uname | tr A-Z a-z)

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -trimpath -ldflags "-s -w" \
	-o ${BUILD_DIR}/connhex-cli cmd/main.go
endef

cli:
	$(call compile_service,$(@))

.PHONY: cli
