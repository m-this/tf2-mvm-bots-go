GO ?= go
GOLANGCI_VERSION ?= latest
GOLANGCI := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
UPSTREAM ?= ../tf2-mvm-bots

.PHONY: help gen check test lint vet clean

help:
	@grep -E '^[a-z-]+:' Makefile | cut -d: -f1 | grep -v help

gen:
	$(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen

check: vet lint test
	$(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen
	@cp -r gen .gen.first && $(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen \
		&& diff -r .gen.first gen >/dev/null \
		|| { rm -rf .gen.first; echo "generated output is not reproducible"; exit 1; }
	@rm -rf .gen.first

test:
	MVMBOTS_UPSTREAM=$(UPSTREAM) $(GO) test -race ./...

vet:
	$(GO) vet ./...

lint:
	$(GO) run $(GOLANGCI) run

clean:
	rm -rf gen bin
