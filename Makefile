BINARY := bin/infraforge

.PHONY: build test lint fmt clean

build:
	go build -o $(BINARY) ./cmd/infraforge

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -l -w cmd internal

clean:
	rm -rf bin
