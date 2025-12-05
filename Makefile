BINARY_NAME := simple-downloader
BUILD_DIR := build
CMD_PATH := ./cmd/simple-downloader

# Static binary flags
CGO_ENABLED := 0

# Priority: ENV var first, then git, then "unknown"
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
CURL_VERSION ?=
LDFLAGS := -s -w -X simple-downloader/internal/version.CommitHash=$(COMMIT_HASH)
ifneq ($(CURL_VERSION),)
LDFLAGS += -X simple-downloader/internal/version.CurlVersion=$(CURL_VERSION)
endif

.PHONY: all build clean

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

clean:
	rm -rf $(BUILD_DIR)

