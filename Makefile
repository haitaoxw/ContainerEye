.PHONY: all build clean test lint

# Build settings
BINARY_DIR=bin
SERVER_BINARY=$(BINARY_DIR)/containereye-server
CLI_BINARY=$(BINARY_DIR)/containereye
GO=go

# Get the current operating system
ifeq ($(OS),Windows_NT)
	BINARY_EXT=.exe
	RM=del /Q
	MKDIR=mkdir
else
	BINARY_EXT=
	RM=rm -f
	MKDIR=mkdir -p
endif

SERVER_BINARY_NAME=$(SERVER_BINARY)$(BINARY_EXT)
CLI_BINARY_NAME=$(CLI_BINARY)$(BINARY_EXT)

all: build

build: $(BINARY_DIR) $(SERVER_BINARY_NAME) $(CLI_BINARY_NAME)

$(BINARY_DIR):
	$(MKDIR) $(BINARY_DIR)

$(SERVER_BINARY_NAME): $(wildcard cmd/main.go)
	$(GO) build -o $(SERVER_BINARY_NAME) ./cmd/main.go

$(CLI_BINARY_NAME): $(wildcard cmd/cli/main.go)
	$(GO) build -o $(CLI_BINARY_NAME) ./cmd/cli/main.go

test:
	$(GO) test -v ./...

lint:
	$(GO) vet ./...
	golangci-lint run

clean:
	$(RM) $(SERVER_BINARY_NAME)
	$(RM) $(CLI_BINARY_NAME)
	$(RM) $(BINARY_DIR)
