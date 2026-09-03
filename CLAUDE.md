# tf2-mvm-bots-go

The MvM defender bots: the decisions in Go, the generators that turn them into
SourcePawn, the plugin they become, and the test-bed that plays them.

Everything ends up here: the plugin, the test-bed, the statistics plugins and
the tooling around them. SourcePawn is a build artifact and not a place anybody
writes. `plugin/` is the SourcePawn tree, generated files and all: it moved here
from `../tf2-mvm-bots`, which is being archived.

That is wider than this repository was opened with, and it is most of the way
there. `plugin/source` holds 1774 lines of hand-written SourcePawn, down from
27005. The menus, the aiming, the shopping trip and the preferences are all
generated now; `nextbot_behavior.sp` and `player_pref.sp` are gone from the
tree entirely. What remains is `tf2_defenderbots.sp` and the gamedata seam:
`dhooks.sp`, `sdkcalls.sp`, `tf_upgrades.sp`, `offsets.sp` and the one native
declaration in `archipelago.sp`.

`tf2_defenderbots.sp` is 766 lines, six functions and the declarations. Every
ordinary function in it is generated, along with every SourceMod forward,
`OnPlayerRunCmd` included, the tournament readiness listener, the sound hook
and thirteen of its fifteen console commands. What is left is the registration
table, two debug commands that want shapes the emitter has not got, and two
dead functions kept on purpose. `mvm-z83.66` says which for each, and the file
will not reach zero: it is the plugin's entry point.

The proofs no longer read the old repository. Every shipped file a comparison
needs is snapshotted under `internal/upstream/shipped` at the revision that
comparison reads it at, so the port keeps its evidence after the repository it
came from is archived.

The design and the reasoning, including why SourceGo is read and not forked, is
`docs/design.md`. Read it first, and read the epic `mvm-z83` with it: the design
was drafted under the narrower aim and its own text says so where it has been
overtaken.

## Layout

- `internal/bindings` — parse the `.inc` files under
  `plugin/testbed/build/` and emit Go declarations for every native.
  Mechanical. Nothing here is hand transcribed.
- `internal/tables` — the feature table and the wave record. One Go declaration
  per fact, emitting both the SourcePawn side and the Go test-bed side.
- `internal/gosubset` — the checker that refuses any Go construct the body
  generator does not support, with a line number.
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
- `internal/spaction`, `internal/action` — a behaviour. A Go package with the
  callbacks becomes the `BehaviorAction` subclass, the constructor and the
  wiring; the bodies come from `internal/spbody`, which is why this part is
  small. `internal/action/spysap` is the first one across, compared against the
  file it replaces on every callback's declaration and call sequence.
- `internal/body` — the bodies themselves, one package each, and the list that
  says which are generated. `internal/body/roster` is the first, and it is
  proved twice: run under spshell against the same canned world as the Go, call
  traces compared, and its DHook callbacks compiled with the shipped compiler.
- `internal/spshell` — running generated SourcePawn under SourcePawn's
  standalone VM, so a generated function can be compared with the Go it came
  from on golden inputs.
- `tools/spshell.sh` — clones and builds SourcePawn at a pinned commit into
  `toolchain/`, which is what `spcomp` and `spshell` above are. `make
  toolchain`, cached, gitignored.
- `gen/` — output. Gitignored. `make gen` rebuilds it, `make check` proves it
  regenerates clean and compiles.
- `internal/runmap`, `cmd/mapview` — a run drawn over the nav mesh it was played
  on.
- `cmd/testbed`, `cmd/rc`, `internal/lab`, `internal/rcon`, `internal/wave`,
  `report`, `sweepreport` — the test-bed. It runs the mission, watches the
  waves and reports what happened. It drives the plugin repository through
  `internal/upstream`: build.sh, the compose file, the popfiles and the map
  configs all live there, and none of them are code.

## Beads

The tracker lives here. It came over from `../tf2-mvm-bots` whole, 161 issues
and every note, by `bd init -p mvm --from-jsonl` on the tracked export; the
counts were checked both ways. Everything is under the epic `mvm-z83`.

- The prefix is `mvm`. P0 crash, P1 costs a player a run, P2 bug, P3 polish.
- Git tracks `.beads/issues.jsonl`. Git ignores the Dolt database.
- There is no Dolt remote. A new clone needs `bd init --from-jsonl`.
- Two sessions share one database and can overwrite each other. Check with
  `bd show` before you set a status.

## Rules

- Generated code is gitignored and never edited by hand.
- `make check` is the gate: `go vet`, the linter, `go test -race`, then
  generation, then `spcomp` over the output. It sets `MVMBOTS_REQUIRE_SPSHELL`
  and `MVMBOTS_REQUIRE_UPSTREAM`, so a test that needs the toolchain or the
  plugin repository fails there rather than skipping.
- The plugin repository is reached through `internal/upstream` and nowhere else.
  It owns the pinned revision and the path resolution, because three packages
  doing it themselves got it wrong and their proofs skipped in silence.
- CI runs Make targets, never raw commands, so the gate runs the same locally.
- No dependency without a reason written down. The standard library first.
- Every generator has golden files. A generator without a test is decoration.
- The port is behaviour identical. No functionality lost, no new bugs, and no
  fix riding along with a port: a defect the port finds goes in a bead and
  stays in the code, because a run that moves cannot say whether the port or
  the fix moved it. `mvm-z83.41` is the standing version of this.


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:6cd5cc61 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Beads

The tracker lives here. It came over from `../tf2-mvm-bots` whole, 161 issues
and every note, by `bd init -p mvm --from-jsonl` on the tracked export; the
counts were checked both ways. Everything is under the epic `mvm-z83`.

- The prefix is `mvm`. P0 crash, P1 costs a player a run, P2 bug, P3 polish.
- Git tracks `.beads/issues.jsonl`. Git ignores the Dolt database.
- There is no Dolt remote. A new clone needs `bd init --from-jsonl`.
- Two sessions share one database and can overwrite each other. Check with
  `bd show` before you set a status.

## Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Agent Context Profiles

The managed Beads block is task-tracking guidance, not permission to override repository, user, or orchestrator instructions.

- **Conservative (default)**: Use `bd` for task tracking. Do not run git commits, git pushes, or Dolt remote sync unless explicitly asked. At handoff, report changed files, validation, and suggested next commands.
- **Minimal**: Keep tool instruction files as pointers to `bd prime`; use the same conservative git policy unless active instructions say otherwise.
- **Team-maintainer**: Only when the repository explicitly opts in, agents may close beads, run quality gates, commit, and push as part of session close. A current "do not commit" or "do not push" instruction still wins.

## Session Completion

This protocol applies when ending a Beads implementation workflow. It is subordinate to explicit user, repository, and orchestrator instructions.

1. **File issues for remaining work** - Create beads for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Handle git/sync by active profile**:
   ```bash
   # Conservative/minimal/default: report status and proposed commands; wait for approval.
   git status

   # Team-maintainer opt-in only, unless current instructions forbid it:
   git pull --rebase
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->
