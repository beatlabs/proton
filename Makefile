GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

.PHONY: build
build:
	$(GOBUILD) -o proton -v

.PHONY: lint
lint:
	golint -set_exit_status=1 `go list ./... | grep -v tools`

.PHONY: test
test:
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic -tags=integration ./...

.PHONY: test
download:
	@echo Download go.mod dependencies
	@go mod download
