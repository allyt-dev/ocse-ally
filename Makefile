.PHONY: build build-all clean test

# Default build for current platform
build:
	go build -o bin/ocse

# Build for all platforms
build-all: clean
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/ocse-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o bin/ocse-darwin-arm64
	GOOS=linux GOARCH=amd64 go build -o bin/ocse-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o bin/ocse-linux-arm64
	GOOS=windows GOARCH=amd64 go build -o bin/ocse-windows-amd64.exe

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test ./...

# Install locally
install:
	go install

# Development build with race detection
dev:
	go build -race -o bin/ocse-dev
