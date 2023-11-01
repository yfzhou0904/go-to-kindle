# Define the binary name
BINARY_NAME=go-to-kindle

# Define the build and clean targets
.PHONY: build clean

build:
	# Create the bin directory if it doesn't exist
	mkdir -p bin/
	# Build the main package and place the executable in the bin directory
	go build -o bin/$(BINARY_NAME) *.go

clean:
	# Remove the binary from the bin directory
	rm -f bin/$(BINARY_NAME)