# Registry and image settings
CHX_DOCKER_REGISTRY ?= registry.compiuta.com
CHX_DOCKER_IMAGE_NAME_PREFIX ?= connhex
BUILD_DIR = build
SERVICE = cli

# Build information
GOARCH ?= $(shell uname -m)
CGO_ENABLED ?= 0
GOOS ?= $(shell uname | tr A-Z a-z)

# Multi-architecture settings
PLATFORMS ?= linux/amd64,linux/arm64
MULTIARCH_BUILDER_NAME := multiarch-builder

# Go build function for service
define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -trimpath -ldflags "-s -w" \
	-o ${BUILD_DIR}/connhex-$(1) cmd/main.go
endef

# Make sure buildx builder exists and is properly set up
check_multiarch_builder:
	@if ! docker buildx inspect $(MULTIARCH_BUILDER_NAME) >/dev/null 2>&1; then \
		echo "Creating multiarch builder named '$(MULTIARCH_BUILDER_NAME)'..."; \
		docker buildx create --use --name $(MULTIARCH_BUILDER_NAME) --platform $(PLATFORMS); \
		docker buildx inspect --bootstrap; \
	else \
		echo "Builder '$(MULTIARCH_BUILDER_NAME)' already exists."; \
		docker buildx use $(MULTIARCH_BUILDER_NAME); \
	fi

# Target to ensure the builder is available
ensure_builder: check_multiarch_builder

# Function to build and push multi-architecture images
define docker_push_multi
	@echo "Building and pushing images with tag: $(1)"
	@for svc in $(SERVICE); do \
		echo "Processing service: $$svc"; \
		docker buildx build --platform $(PLATFORMS) \
			--build-arg SVC=$$svc \
			--tag=$(CHX_DOCKER_REGISTRY)/$(CHX_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(1) \
			-f Dockerfile --push .; \
	done
endef

# Individual service build targets
$(SERVICE):
	$(call compile_service,$(@))

.PHONY: $(SERVICE) latest release-dev release ensure_builder check_multiarch_builder

# Push images with 'latest' tag
latest: ensure_builder
	$(call docker_push_multi,latest)

# Push images with 'development' tag
release-dev: ensure_builder
	$(call docker_push_multi,development)

# Push images with version tag from git
release: ensure_builder
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(call docker_push_multi,$(version))
