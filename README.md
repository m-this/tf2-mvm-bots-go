# tf2-mvm-bots-go

The decisions the RED bots make in Mann vs Machine, written in Go, and the
generators that turn them into SourcePawn.

The plugin is [tf2-mvm-bots](https://github.com/m-this/tf2-mvm-bots). It kept
the SourcePawn that had to be SourcePawn: `SDKCall`, `DHook`, raw address reads,
gamedata. About 250 sites in 24k lines. That boundary is gone: `internal/spbody`
emits all three, so what stays there is the gamedata, the map configs and the
popfiles, and this repository is meant to own everything else.

## Why

Three complaints. A name inserted in the wrong place in a parallel enum renamed
three convars, and an A/B armed the wrong feature and was read as a
measurement. The decision code could not be run without a game server, so
asking whether a change was right meant playing a wave, which takes minutes and
answers with noise. And a difference had no noise floor to be measured against.

None of those is a SourcePawn problem. They are a problem of one fact written
down twice, and of the interesting code being unreachable from a test.

## What is here

- `internal/tables` — the feature table and the wave record. One Go declaration
  per fact, emitting the SourcePawn side and the Go test-bed side, proved to
  round-trip against the plugin field for field.
- `internal/bindings` — 204 `.inc` files parsed into 1175 natives, 112
  methodmaps and 253 enums. Nothing hand transcribed. Counts are asserted
  against a grep of the declaration lines, so a parser that swallows a
  declaration fails rather than under-reports.
- `internal/gosubset` — the checker that refuses any construct the body
  generator does not support, with a line number and the replacement to write.
- `internal/spbody` — the body generator: a Go package that passes the subset
  becomes SourcePawn, engine calls included. It type checks with go/types
  first, so a width, a named type and an array length are read off the type
  rather than guessed.
- `internal/engine` — the one package a body may import. One Go function per
  engine call, each carrying the directive that says whether SourcePawn writes
  it as a native, an SDKCall or an address read. Nothing here means anything in
  a Go process: the differential test installs the answers.
- `internal/body/scan` — util.sp's client loop, ported one function at a time.
  The duplication it holds is collapsed once every variant is here, not on the
  way across.
- `internal/body` — the bodies themselves, one package each, and the list that
  says which are generated. `internal/body/roster` is the first, and it is
  proved twice: run under spshell against the same canned world as the Go, call
  traces compared, and its DHook callbacks compiled with the shipped compiler.
- `internal/actionsel` — action selection as a total function over 1425408
  reachable combinations, with exhaustiveness asserted. It found a hole that
  was shipping.
- `internal/spshell` — golden inputs through `spcomp` and SourcePawn's
  standalone VM, compared with the Go on `float32` bits, with no game server.

`gen/` is the output. It is gitignored and never edited by hand.

## The gate

`make check`: `go vet`, golangci-lint, `go test -race`, then generation run
twice with the two outputs diffed. Generated code that is not reproducible is
not generated code.

`make toolchain` builds SourcePawn's own compiler and VM at a pinned commit,
cached under `toolchain/`. Without it the differential tests skip and say so;
`make check` builds it first and sets `MVMBOTS_REQUIRE_SPSHELL`, so under the
gate an absent toolchain fails instead of quietly running nothing.

It sets `MVMBOTS_REQUIRE_UPSTREAM` for the same reason. Three packages resolved
the path to the plugin repository themselves, got it wrong, and skipped: the
binding and nav mesh proofs reported `ok` in under a second while running none
of them.

## What this does not fix

Bots stuck on geometry, nav mesh failures, engineers wedged in props. Reading
the whole bug record afterwards said that is most of the work, and
`docs/design.md` was corrected to say so rather than quietly dropped.

## Credit

The Go-to-SourcePawn mappings descend from
[SourceGo](https://github.com/Nirari-Technologies/Go2SourcePawn) by Kevin
Yonan, MIT licensed. Read for its passes, not forked; `docs/design.md` gives
the reasoning. Attribution is in `internal/gosubset/ATTRIBUTION.md`.
