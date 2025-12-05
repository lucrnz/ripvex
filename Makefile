BINARY_NAME := ripvex
BUILD_DIR := build
CMD_PATH := ./cmd/ripvex

# Static binary flags
CGO_ENABLED := 0

# Priority: ENV var first, then git, then "unknown"
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION_PREFIX ?= dev
VERSION_DATE ?= $(shell date +%Y%m%d)
CURL_VERSION ?=
LDFLAGS := -s -w -X ripvex/internal/version.CommitHash=$(COMMIT_HASH) -X ripvex/internal/version.VersionPrefix=$(VERSION_PREFIX) -X ripvex/internal/version.VersionDate=$(VERSION_DATE)
ifneq ($(CURL_VERSION),)
LDFLAGS += -X ripvex/internal/version.CurlVersion=$(CURL_VERSION)
endif

.PHONY: all build clean

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

clean:
	rm -rf $(BUILD_DIR)
