GO ?= go
GOLANGCI_VERSION ?= latest
GOLANGCI := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
PLUGIN ?= plugin

# The standalone SourcePawn build tools/spshell.sh caches. Present means the
# differential tests run; absent means they skip and say so.
SPWORK ?= $(CURDIR)/toolchain
SPROOT := $(SPWORK)/sourcepawn
SPENV := SPCOMP=$(SPROOT)/objdir/spcomp/linux-x86_64/spcomp \
	SPSHELL=$(SPROOT)/objdir/spshell/linux-x86_64/spshell \
	SPINCLUDE=$(SPROOT)/include/core

.PHONY: help gen check test lint vet toolchain clean

help:
	@grep -E '^[a-z-]+:' Makefile | cut -d: -f1 | grep -v help

gen:
	$(GO) run ./cmd/gen -plugin $(PLUGIN) -out gen

# The target-specific variable reaches test through the prerequisite, which is
# the whole point: the gate runs the same tests and refuses to skip the ones
# that need spcomp.
check: REQUIRE := MVMBOTS_REQUIRE_SPSHELL=1 MVMBOTS_REQUIRE_PLUGIN=1
check: toolchain gen vet lint test
	$(GO) run ./cmd/gen -plugin $(PLUGIN) -out gen
	@cp -r gen .gen.first && $(GO) run ./cmd/gen -plugin $(PLUGIN) -out gen \
		&& diff -r .gen.first gen >/dev/null \
		|| { rm -rf .gen.first; echo "generated output is not reproducible"; exit 1; }
	@rm -rf .gen.first

# gen first, and not only in check: report imports the generated wave record, so
# a clone that has never generated does not build at all. It is cheap and
# reproducible, so there is no reason to make anybody remember it.
test: gen
	MVMBOTS_PLUGIN=$(PLUGIN) $(SPENV) $(REQUIRE) $(GO) test -race ./...

# Builds SourcePawn's own compiler and VM at the pinned commit. Idempotent: a
# second run finds the binaries and exits.
toolchain:
	SPWORK=$(SPWORK) tools/spshell.sh

# Generated code is proved by compiling, not by vet and the linter. Its shape is
# SourcePawn's: bitbuffer.inc really does have ReadByte and WriteByte, and vet
# reads those as broken io interfaces. Renaming them would make the binding
# wrong to be tidy.
vet: gen
	$(GO) vet ./cmd/... ./internal/... ./report/... ./sweepreport/...
	$(GO) build ./gen/...

lint: gen
	$(GO) run $(GOLANGCI) run ./cmd/... ./internal/... ./report/... ./sweepreport/...

clean:
	rm -rf gen bin toolchain
