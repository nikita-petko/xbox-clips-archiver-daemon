IMAGE_TAG := $(shell git rev-parse HEAD 2> /dev/null)

ifeq ($(OS),Windows_NT)
	NUL = nul
	IMAGE_TAG := $(shell git rev-parse HEAD 2> $(NUL))
else
	NUL = /dev/null
	IMAGE_TAG := $(shell git rev-parse HEAD 2> $(NUL))
endif

COMPANY_NAME ?= mfdlabs
PROJECT_NAME ?= $(notdir $(CURDIR))
IMAGE_NAME ?= $(COMPANY_NAME)/$(PROJECT_NAME)

ifeq ($(OS),Windows_NT)
	ARCH := $(PROCESSOR_ARCHITECTURE)

	GOOS := windows
else 

	UNAME := $(shell uname)
	ARCH := $(shell uname -m)


	ifeq ($(UNAME),Linux)
		GOOS := linux
	else ifeq ($(UNAME),Darwin)
		GOOS := darwin
	endif

endif

# Convert the ARCH value to the corresponding GOARCH string
ifeq ($(ARCH),x86_64)
	GOARCH := amd64
else ifeq ($(ARCH),i386)
	GOARCH := 386
else ifeq ($(ARCH),armv7l)
	GOARCH := arm
else ifeq ($(ARCH),aarch64)
	GOARCH := arm64
else ifeq ($(ARCH),AMD64)
	GOARCH := amd64
else ifeq ($(ARCH),ARM64)
	GOARCH := arm64
else ifeq ($(ARCH),x86)
	GOARCH := 386
endif

ARCH_OUT = $(if $(filter 386,$(GOARCH)),x86,$(if $(filter amd64,$(GOARCH)),x64,$(GOARCH)))

ifeq ($(GOOS),windows)
	OUTFILE := $(PROJECT_NAME).exe
else
	OUTFILE := $(PROJECT_NAME)
endif

.DEFAULT_GOAL := all
.PHONY: all fmt test vendor build-debug-x86 build-debug-x64 build-debug-arm build-debug-arm64
.PHONY: build-release-x86 build-release-x64 build-release-arm build-release-arm64
.PHONY: build-debug build-release build-debug-all build-release-all build-docker

all:
	$(info  make <cmd>)
	$(info )
	$(info commands:)
	$(info )
	$(info  fmt                  - format Go source files.)
	$(info  test                 - vet and run all unit tests.)
	$(info  vendor               - update go.mod, go.sum, and vendor/.)
	$(info  build-docker         - build a docker image, IMAGE_NAME will default to $(COMPANY_NAME)/$(PROJECT_NAME) if not specified!)
	$(info  build-debug          - build a debug build. If GOARCH or GOOS were not specified "$(GOOS)/$(GOARCH)" will be used.)
	$(info  build-release        - build a release build that strips all symbols. If GOARCH or GOOS were not specified, "$(GOOS)/$(GOARCH)" will be used.)
	$(info  build-debug-<arch>   - helper for building a debug build for a specific arch, <arch> can be either x86, x64, arm or arm64. If GOOS was not specified, "$(GOOS)" will be used.)
	$(info  build-release-<arch> - helper for building a release build for a specific arch that strips all symbols, <arch> can be either x86, x64, arm or arm64. If GOOS was not specified, "$(GOOS)" will be used.)
	$(info )

fmt:
	@cd ./src && go fmt ./...

test:
	$(info Running Tests...)

	@cd ./src && go vet ./... && go test -race -count=1 ./...

vendor:
	$(info Go Vendor being applied to $(CURDIR)/src/vendor.)

ifeq ($(strip $(SKIP_VENDOR)),)
	@cd ./src && go mod tidy && go mod vendor
else
	$(info Go Vendor skipped because SKIP_VENDOR specified.)
endif

define BUILD_RECIPE
	$(info Go $(BUILD_MODE) build for $(GOOS)/$(GOARCH) -> $(CURDIR)/$(BUILD_OUTPUT)/$(OUTFILE))
	@test -n "$(GOOS)" || { echo "Could not detect GOOS; specify it with make GOOS=<goos>." >&2; exit 1; }
	@test -n "$(GOARCH)" || { echo "Unsupported host architecture '$(ARCH)'; specify it with make GOARCH=<goarch>." >&2; exit 1; }
	@mkdir -p "$(BUILD_OUTPUT)"
	@cd ./src && GOARCH="$(GOARCH)" GOOS="$(GOOS)" go build -mod=vendor -ldflags "$(if $(filter release,$(BUILD_MODE)),-s -w) -X main.commitSha=$(IMAGE_TAG) -X main.buildMode=$(BUILD_MODE) -X main.applicationName=$(PROJECT_NAME)" -v -o ../$(BUILD_OUTPUT)/$(OUTFILE) ./main.go
endef

BUILD_OUTPUT = bin/$(BUILD_MODE)/$(GOOS)/$(ARCH_OUT)
GO_SOURCE_DIRS := $(shell find src -path src/.output -prune -o -type d -print)
GO_SOURCE_FILES := $(shell find src -path src/.output -prune -o -type f -name '*.go' -print)
GO_BUILD_INPUTS := $(GO_SOURCE_DIRS) $(GO_SOURCE_FILES) src/go.mod src/go.sum src/vendor/modules.txt Makefile
arch_dir = $(if $(filter 386,$(1)),x86,$(if $(filter amd64,$(1)),x64,$(1)))

define BUILD_OUTPUT_RULE
$(CURDIR)/bin/$(1)/$(GOOS)/$(call arch_dir,$(2))/$(OUTFILE): BUILD_MODE := $(1)
$(CURDIR)/bin/$(1)/$(GOOS)/$(call arch_dir,$(2))/$(OUTFILE): GOARCH := $(2)
$(CURDIR)/bin/$(1)/$(GOOS)/$(call arch_dir,$(2))/$(OUTFILE): $(GO_BUILD_INPUTS)
	$$(BUILD_RECIPE)
endef

$(foreach arch,386 amd64 arm arm64,$(eval $(call BUILD_OUTPUT_RULE,debug,$(arch))))
$(foreach arch,386 amd64 arm arm64,$(eval $(call BUILD_OUTPUT_RULE,release,$(arch))))

build-debug: $(CURDIR)/bin/debug/$(GOOS)/$(ARCH_OUT)/$(OUTFILE)
build-release: $(CURDIR)/bin/release/$(GOOS)/$(ARCH_OUT)/$(OUTFILE)

build-debug-x86: $(CURDIR)/bin/debug/$(GOOS)/x86/$(OUTFILE)
build-debug-x64: $(CURDIR)/bin/debug/$(GOOS)/x64/$(OUTFILE)
build-debug-arm: $(CURDIR)/bin/debug/$(GOOS)/arm/$(OUTFILE)
build-debug-arm64: $(CURDIR)/bin/debug/$(GOOS)/arm64/$(OUTFILE)

build-release-x86: $(CURDIR)/bin/release/$(GOOS)/x86/$(OUTFILE)
build-release-x64: $(CURDIR)/bin/release/$(GOOS)/x64/$(OUTFILE)
build-release-arm: $(CURDIR)/bin/release/$(GOOS)/arm/$(OUTFILE)
build-release-arm64: $(CURDIR)/bin/release/$(GOOS)/arm64/$(OUTFILE)

build-debug-all: build-debug-x86 build-debug-x64 build-debug-arm build-debug-arm64
build-release-all: build-release-x86 build-release-x64 build-release-arm build-release-arm64

build-docker:
ifndef IMAGE_NAME
	$(info IMAGE_NAME is not defined)
else
ifndef COMPANY_NAME
	$(info COMPANY_NAME is not defined)
else
ifndef PROJECT_NAME
	$(info PROJECT_NAME is not defined)
else
	@docker build --build-arg PROJECT_NAME=$(PROJECT_NAME) -t $(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile .
	@docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

ifdef CI
	@docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@docker push $(IMAGE_NAME):latest
endif

ifndef KEEP_IMAGES
	@docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
	@docker rmi $(IMAGE_NAME):latest
endif
endif
endif
endif