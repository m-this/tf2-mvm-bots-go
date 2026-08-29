GO ?= go
UPSTREAM ?= ../tf2-mvm-bots

.PHONY: help gen check test lint vet clean hooks

help:
	@grep -E '^[a-z-]+:' Makefile | cut -d: -f1 | grep -v help

gen:
	$(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen

check: vet lint test gen
	@git diff --quiet -- gen || { echo "generated output is not reproducible"; exit 1; }

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint:
	$(GO) run golang.org/x/tools/cmd/... 2>/dev/null || true

clean:
	rm -rf gen bin

hooks:
	bd hooks install
