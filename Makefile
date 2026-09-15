BINARY := bin/infraforge

.PHONY: build test test-integration lint fmt clean

build:
	go build -o $(BINARY) ./cmd/infraforge

test:
	go test ./...

test-integration:
	go test -tags=integration -timeout=10m ./test/integration/...

lint:
	golangci-lint run

fmt:
	gofmt -l -w cmd internal

clean:
	rm -rf bin
