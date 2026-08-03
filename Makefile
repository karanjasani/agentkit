.PHONY: build test lint vet vuln cover tidy fmt clean install golden

BINARY := repomap
PKG := ./...

build:
	go build -o $(BINARY) ./cmd/repomap

install:
	go install ./cmd/repomap

test:
	go test $(PKG)

golden:
	go test ./... -run Golden -update

cover:
	go test -coverprofile=coverage.txt -covermode=atomic $(PKG)
	go tool cover -func=coverage.txt

vet:
	go vet $(PKG)

lint:
	golangci-lint run

vuln:
	govulncheck $(PKG)

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) coverage.txt coverage.html
	rm -rf dist/
