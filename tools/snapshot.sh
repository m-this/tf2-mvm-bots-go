#!/bin/sh
# Refresh internal/upstream/shipped from the plugin repository.
#
# The proofs compare what this repository generates with what the plugin
# shipped, and reading that out of the plugin's git history ties every proof to
# a repository this one is meant to replace. The snapshot is the copy those
# proofs read instead: 700 KiB of text, versioned beside what it proves, and
# checked against the repository by TestSnapshotMatchesTheRepository for as
# long as the repository is still around.
#
# Run it after adding a Body with a Shipped path, or after moving a pin. It is
# not part of `make check`, because writing evidence is a thing somebody
# decides to do.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

go run ./cmd/snapshot
