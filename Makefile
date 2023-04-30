git_tag = $(shell git describe --always --dirty --tags --long)
ldflags = "-s -X 'github.com/pepa65/bat/internal/cli.tag=${git_tag}'"

## help: print this help message
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -s ':' -t | sed -e 's/^/ /'

## audit: format, vet, and test code
.PHONY: audit
audit: test
	#@echo "Formatting code:"
	#gofumpt -w .
	@echo "Vetting code:"
	go vet ./...
	#staticcheck ./...

## build: build the cmd/bat application
.PHONY: build
build:
	@echo "Building bat:"
	GOOS=linux GOARCH=amd64 go build -ldflags=${ldflags} ./cmd/bat/

## install: install the cmd/bat application
.PHONY: install
install:
	@echo "Building and installing bat:"
	GOOS=linux GOARCH=amd64 go build -ldflags=${ldflags} ./cmd/bat/
	-sudo mv bat /usr/local/bin/

## clean: delete build artefacts
.PHONY: clean
clean:
	@echo "Deleting build artefacts:"
	-rm -f bat cover.out

## test: runs tests
.PHONY: test
test:
	@echo "Running tests:"
	go test -v -race -vet=off -ldflags=${ldflags} ./...

## cover: shows application coverage in browser
.PHONY: cover
cover:
	@echo "Running coverage:"
	go test -coverprofile=cover.out -ldflags=${ldflags} ./...
	go tool cover -html=cover.out
