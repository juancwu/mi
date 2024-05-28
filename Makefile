# Makefile for building Go application for different platforms

APP_NAME := konbini
SRC := main.go

# List of supported platforms
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target to build for all platforms
all: $(PLATFORMS)

# Pattern rule to build for each platform
$(PLATFORMS):
	@GOOS=$(word 1, $(subst /, ,$@)) GOARCH=$(word 2, $(subst /, ,$@)) go build -o ./bin/$(APP_NAME)-$(word 1, $(subst /, ,$@))-$(word 2, $(subst /, ,$@)) $(SRC)
	@echo "Built $(APP_NAME) for $@"

# Clean up generated binaries
clean:
	@rm -f ./bin/$(APP_NAME)-*
	@echo "Cleaned up binaries"

.PHONY: all clean $(PLATFORMS)
