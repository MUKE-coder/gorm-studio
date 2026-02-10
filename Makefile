.PHONY: build test demo lint clean

# Build the demo binary
build:
	go build -o bin/gorm-studio-demo .

# Run all tests
test:
	go test -v -race -cover ./...

# Run the demo application
demo:
	go run main.go

# Run linters
lint:
	go vet ./...
	@echo "Lint passed"

# Clean build artifacts
clean:
	rm -rf bin/ demo.db
