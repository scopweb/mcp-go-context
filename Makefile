.PHONY: build run test clean install

# Variables
BINARY_NAME=mcp-go-context
VERSION?=1.0.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT}"

# Build the binary
build:
	go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/mcp-go-context

# Run the server
run: build
	./bin/${BINARY_NAME}

# Run with specific transport
run-stdio: build
	./bin/${BINARY_NAME} --transport stdio

run-http: build
	./bin/${BINARY_NAME} --transport http --port 3000

run-sse: build
	./bin/${BINARY_NAME} --transport sse --port 3000

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install globally
install: build
	cp bin/${BINARY_NAME} $(GOPATH)/bin/

# Cross-compilation
build-all:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 ./cmd/mcp-go-context
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 ./cmd/mcp-go-context
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 ./cmd/mcp-go-context
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-arm64 ./cmd/mcp-go-context
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe ./cmd/mcp-go-context

# Development with hot reload (requires air)
dev:
	air

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Generate documentation
docs:
	godoc -http=:6060

# Docker build
docker-build:
	docker build -t mcp-go-context:${VERSION} .

# Docker run
docker-run:
	docker run -it --rm mcp-go-context:${VERSION}