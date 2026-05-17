# SourceVault Makefile

BINARY_NAME=sourcevault
MAIN_PATH=./cmd/sourcevault

# Standard Linux installation prefix.
# Fallback to SOURCEVAULT_BASE_DIR if PREFIX is not provided.
ifdef SOURCEVAULT_BASE_DIR
	PREFIX ?= $(SOURCEVAULT_BASE_DIR)
endif

# The directory where the binary will be installed.
BINDIR = $(PREFIX)/bin

# Git version information
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "UNKNOWN")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "UNKNOWN")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X 'sourcevault/internal/version.gitCommit=$(GIT_COMMIT)' \
           -X 'sourcevault/internal/version.gitBranch=$(GIT_BRANCH)' \
           -X 'sourcevault/internal/version.buildDate=$(BUILD_DATE)'

.PHONY: all build run clean test install uninstall

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)

install: build
ifeq ($(PREFIX),)
	$(error Installation path not set. Please set PREFIX=location or SOURCEVAULT_BASE_DIR=location)
endif
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

uninstall:
ifeq ($(PREFIX),)
	$(error Installation path not set. Please set PREFIX=location or SOURCEVAULT_BASE_DIR=location)
endif
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

run: build
	./$(BINARY_NAME)

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test ./...
