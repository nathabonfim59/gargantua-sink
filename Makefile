BINARY_NAME=gargantua-sink
VERSION=0.1.0
BUILD_DIR=build
MAIN_PATH=cmd/gargantua-sink/main.go

.PHONY: all build clean test run tidy

all: clean build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test ./...

run:
	@go run $(MAIN_PATH)

tidy:
	@echo "Tidying up modules..."
	@go mod tidy
