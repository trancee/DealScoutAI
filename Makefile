BINARY  := dealscout
MODULE  := github.com/trancee/DealScout
CMD     := ./cmd/dealscout

# Build settings
GOFLAGS := -trimpath
LDFLAGS := -s -w

.PHONY: all build clean test lint vet fmt check

all: check build

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) $(CMD)

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet lint test
