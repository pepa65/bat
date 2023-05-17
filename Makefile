git_tag = $(shell git describe --always --dirty --tags --long)
ldflags = "-s -X 'github.com/pepa65/bat/internal/cli.tag=${git_tag}'"

## help: Display this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -s ':' -t | sed -e 's/^/ /'

## audit: Format, vet, and test code
.PHONY: audit
audit: test
	@echo "Formatting code:"
	gofumpt -w .
	@echo "Vetting code:"
	go vet ./...
	staticcheck ./...

## build: Build the cmd/bat application
.PHONY: build
build:
	@echo "Building bat:"
	GOOS=linux GOARCH=amd64 go build -ldflags=${ldflags} ./cmd/bat/

## install: Build and install the cmd/bat application
.PHONY: install
install:
	@echo "Building and installing bat:"
	GOOS=linux GOARCH=amd64 go build -ldflags=${ldflags} ./cmd/bat/
	-sudo mv bat /usr/local/bin/

## clean: Delete build artifacts
.PHONY: clean
clean:
	@echo "Deleting build artifacts:"
	-rm -f bat cover.out
