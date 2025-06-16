# Define the binary name
BINARY_NAME=go-to-kindle

# Define the build, test, and clean targets
.PHONY: build test clean

build:
	# Create the bin directory if it doesn't exist
	mkdir -p bin/
	mkdir -p ~/.go-to-kindle
	# Build the main package and place the executable in the bin directory
	go build -o bin/$(BINARY_NAME) *.go

test:
	# Run all tests
	go test ./...

clean:
	# Remove the binary from the bin directory
	rm -f bin/$(BINARY_NAME)
