GO ?= go
GOLANGCI_VERSION ?= latest
GOLANGCI := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
UPSTREAM ?= ../tf2-mvm-bots

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
	$(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen

# The target-specific variable reaches test through the prerequisite, which is
# the whole point: the gate runs the same tests and refuses to skip the ones
# that need spcomp.
check: REQUIRE := MVMBOTS_REQUIRE_SPSHELL=1
check: toolchain vet lint test
	$(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen
	@cp -r gen .gen.first && $(GO) run ./cmd/gen -upstream $(UPSTREAM) -out gen \
		&& diff -r .gen.first gen >/dev/null \
		|| { rm -rf .gen.first; echo "generated output is not reproducible"; exit 1; }
	@rm -rf .gen.first

# The toolchain is a soft dependency here on purpose: a clone with no clang and
# no network still runs every test that does not need spcomp. make check is
# where it is mandatory.
test:
	MVMBOTS_UPSTREAM=$(UPSTREAM) $(SPENV) $(REQUIRE) $(GO) test -race ./...

# Builds SourcePawn's own compiler and VM at the pinned commit. Idempotent: a
# second run finds the binaries and exits.
toolchain:
	SPWORK=$(SPWORK) tools/spshell.sh

# Generated code is proved by compiling, not by vet and the linter. Its shape is
# SourcePawn's: bitbuffer.inc really does have ReadByte and WriteByte, and vet
# reads those as broken io interfaces. Renaming them would make the binding
# wrong to be tidy.
vet:
	$(GO) vet ./cmd/... ./internal/...
	$(GO) build ./gen/...

lint:
	$(GO) run $(GOLANGCI) run ./cmd/... ./internal/...

clean:
	rm -rf gen bin toolchain
