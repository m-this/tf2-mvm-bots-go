# tf2-mvm-bots-go

The MvM defender bots: the decisions in Go, the generators that turn them into
SourcePawn, the plugin they become, and the test-bed that plays them. Everything
is here, plugin tree included. SourcePawn is a build artifact, not a place
anybody writes.

`plugin/source` holds 221 lines of hand-written SourcePawn and not one function:
`tf2_defenderbots.sp` is the include list, the two `_disable_actions_*` defines
and `public Plugin myinfo`, and `archipelago.sp` is one `native` declaration of
another plugin's export with the paragraph saying why it is optional. The mod
ships one build; the compile-time toggles the SourcePawn had were resolved by
the port. The gamedata seam went the same way: an offset read, an SDKCall
preparation, a DHook callback and a methodmap over an `Address` are all things
the emitter writes.

tf2-archipelago deploys this as a Go module and nothing else. Its go.mod
requirement is the only pin and `go get` is how it moves.

`../tf2-mvm-bots` is archived and nothing here reads it. Nothing is proved
against it either: a claim about what that repository shipped is a claim about a
tree nobody can change. What is proved is this one against itself, generator
against table and generated SourcePawn against the Go it came from.

The design and the reasoning, including why SourceGo is read and not forked, is
`docs/design.md`. Read it with the epic `mvm-z83`: the design was drafted under
a narrower aim and says so where it has been overtaken.

## Layout

- `internal/bindings`, `internal/bindgen` — parse the `.inc` files under
  `plugin/testbed/build/` and emit Go declarations for every native. `bindings`
  reads one include, `bindgen` orders a whole tree and writes the package.
  Mechanical. Nothing here is hand transcribed.
- `internal/tables` — the feature table and the wave record. One Go declaration
  per fact, emitting both the SourcePawn side and the Go test-bed side.
- `internal/gosubset` — the checker that refuses any Go construct the body
  generator does not support, with a line number.
- `internal/spbody` — the body generator: a Go package that passes the subset
  becomes SourcePawn, engine calls included. It type checks with go/types
  first, so a width, a named type and an array length are read off the type
  rather than guessed.
- `internal/engine` — how a body reaches anything it does not own. One Go
  function per engine call, each carrying the directive that says whether
  SourcePawn writes it as a native, an SDKCall or an address read. Nothing here
  means anything in a Go process: the differential test installs the answers,
  and `Fill` puts a panic naming itself behind every one the caller left out.
  A body may also import another generated package: the registry knows what
  each emits, so a shared decision is an import rather than an extern.
- `internal/spaction`, `internal/action` — a behaviour. A Go package with the
  callbacks becomes the `BehaviorAction` subclass, the constructor and the
  wiring; the bodies come from `internal/spbody`, which is why this part is
  small.
- `internal/body` — the bodies themselves, one package each. `internal/body/scan`
  is util.sp's client loop, two loops and the named questions each variant asks,
  pinned in `internal/body/testdata/scan_cells.golden`. `internal/body/roster` is
  proof and ships nowhere: run under spshell against the same canned world as the
  Go, call traces compared, DHook callbacks compiled with the shipped compiler.
- `internal/generated` is the whole output and `internal/adopt` is what the
  plugin tree commits. `cmd/gen` writes all of it; `gen/` is where it lands,
  gitignored.
- `internal/sp`, `internal/tf` — the shared vocabulary. `sp` is what SourcePawn's
  syntax requires of emitted text, `tf` is the game's own enums in the plugin's
  order, so two decisions that branch on a class branch on the same class.
- `internal/spshell`, `tools/spshell.sh` — generated SourcePawn run under
  SourcePawn's standalone VM, so a generated function can be compared with the Go
  it came from on golden inputs. The script clones and builds SourcePawn at a
  pinned commit into `toolchain/`: `make toolchain`, cached, gitignored.
- `internal/navmesh`, `internal/threat`, `internal/upgrade`, `internal/actionsel`
  — the decisions that are ordinary Go rather than a body: the mesh a spot is
  checked against, what a robot is worth killing first, what a bot buys, and
  which behaviour it is handed. `internal/spgen` emits the last two as tables.
- `internal/runmap`, `cmd/mapview` — a run drawn over the nav mesh it was played
  on.
- `cmd/testbed`, `cmd/rc`, `internal/lab`, `internal/rcon`, `internal/wave`,
  `internal/machine`, `report`, `sweepreport` — the test-bed, below.
- `cmd/checkspots` — which dispenser spot each authored nest would take, read off
  the configs with no server. `cmd/deadsweep` — the unreachable-function gate.

## The test-bed

One runner, and it is Go. `cmd/testbed` holds the bed, builds, recreates the
server, checks the plugin it loaded against the source, interleaves the arms,
watches the wave and writes the results. The shell runners are gone: `run.sh`,
`ab.sh`, `sweep.sh`, `batch.sh` and `run-native.sh` were deleted, and a sweep is
`-maps`, an A/B is two `-arm` flags, a batch is `-attempts` and a native run is
`-native`. Prose that still names them is stale, not a path that exists. No
Python is left in the bed.

    go run ./cmd/testbed -map mvm_decoy -mission mvm_decoy_advanced \
        -waves 2 -attempts 3 -tag x \
        -arm on:sm_redbots_feature_x=1 -arm off:sm_redbots_feature_x=0

Nothing below plays anything or takes the lock, so a second shell can use them
while a run is in flight, and each follows a bed rather than a process:

    go run ./cmd/testbed -status      # what the bed is playing, and until when
    go run ./cmd/testbed -follow      # each wave result as it lands, with a tally
    go run ./cmd/testbed -wait        # block, then exit with the run's verdict
    go run ./cmd/testbed -bed list    # every bed on this machine and who holds it
    go run ./cmd/testbed -crashes     # what killed the server, over -since
    go run ./cmd/testbed -reread x    # the verdict again, from the files
    go run ./cmd/rc status            # one console command
    go run ./report results/x-on-1.jsonl -field robot_kills -where result=lost

`-json` on any of them is for a reader that is not a person. The run in flight
is `$TMPDIR/<bed>-run.json`, rewritten at every poll and left behind with the
verdict on it.

The bed is one server for the whole machine, held by a lock under TMPDIR. A
second bed is `TESTBED_PROJECT` with a `TESTBED_PORT` of its own, built as a
symlink tree over the first bed's game volume. Somebody else's bed is never
recreated to make room.

Things that have each cost a day, and are not to be paid for twice:

- The bed is driven through the runner, not through `docker compose`, `docker
  logs` or `pkill -f`. `compose.yml` defaults its project name to the first bed,
  so a compose command typed without `TESTBED_PROJECT` recreates another
  session's server, and `pkill -f` on a test-bed pattern kills their runner.
  `-bed list`, `-bed up` and `-bed down` exist so the name cannot be left off.
- `docker logs` keeps a container's whole life across restarts, and `srcds_run`
  prints its "add -debug" restart line every thirty seconds during an install. A
  crash count over that window counts crashes from hours ago and reads restarts
  as crashes, which is the wrong inference `mvm-427` was filed on. `-crashes`
  scopes the window and names the fault.
- srcds stops printing to stdout just after the Steam init lines and says
  nothing more until the map is up. A log that has stopped moving is not a hung
  server. Ask rcon, or read `-status`, which is written off the runner's own poll.
- The refusals are the point, not an obstacle. `ErrPrecondition` means the
  server is fine and the run is not, and it does not fail intermittently: an
  empty RED, a stale plugin, a mission the server never took, a machine under
  the memory or disk floor. Fix the cause, never the flag, and never
  `-build=false` to get past one.
- A crash in one arm and not the other is not evidence until it reproduces. The
  runner replays a crashed attempt once on the same arm; only a crash that
  happens again is charged to the arm, and the rest are reported as bed crashes.
- The machine decides the numbers, and it is read before the lock is taken.
  Under a gibibyte of MemAvailable the server pages and the watchdog reads a page
  fault as an infinite loop; a full volume truncates an extension copy and every
  map load dies on SIGBUS.
- `-replay` plays a player's `server.cfg`, and that file carries their
  `rcon_password`. The runner refuses a path git would commit; it goes under
  `plugin/testbed/replays/`, which is ignored.
- Measure the configuration the player runs, not one chosen here. Mode 2 with
  their lineup is what their server plays, and the gap between that and the
  default lineup hid the stock sniper bug for a day.

A results file carries three kinds of line: the waves the server wrote, a run
record saying what was played and on what machine, and an attempt record saying
what the runner made of it. The last one is why `-reread` on an old tag says the
same thing the run said when it finished.

## Beads

The tracker lives here, everything under the epic `mvm-z83`.

- The prefix is `mvm`. P0 crash, P1 costs a player a run, P2 bug, P3 polish.
- Git tracks `.beads/issues.jsonl`. Git ignores the Dolt database.
- There is no Dolt remote. A new clone needs `bd init --from-jsonl`, and the
  `bd dolt push/pull` the generated block below talks about has nowhere to go.
- Two sessions share one database and can overwrite each other. Check with
  `bd show` before you set a status.

## Rules

- Generated code is gitignored and never edited by hand. The copies the plugin
  tree commits are written by `make adopt` and checked by `make check`;
  `internal/adopt` is the one list of which files those are.
- `make check` is the gate: `go vet`, the linter, `go test -race`, then
  generation, then `spcomp` over the output. It sets `MVMBOTS_REQUIRE_SPSHELL`
  and `MVMBOTS_REQUIRE_PLUGIN`, so a test that needs the toolchain or the
  plugin tree fails there rather than skipping.
- The plugin tree is reached through `internal/plugin` and nowhere else. It owns
  the path resolution, because three packages doing it themselves got it wrong
  and their proofs skipped in silence.
- CI runs Make targets, never raw commands, so the gate runs the same locally.
- `make deadcode` refuses a function nothing reaches, minus `internal/engine`,
  `internal/action` and `internal/body`, which become SourcePawn and are called
  from no Go main. There is no allow list: an entry is decided, not silenced.
- No dependency without a reason written down. The standard library first.
- Every generator has golden files. A generator without a test is decoration.
- An extern says which of three things it is, and the three are not the same
  work. `//sp:body` names SourcePawn this port generates: its Go signature is
  checked against the function that generates it, because int, float, every tag
  and every handle are one cell there and spcomp cannot see the difference.
  `//sp:plugin` names SourcePawn the port has not reached, and there are six
  left. `//sp:library` names somebody else's include and is not work at all.
- A per-client array is cleared between bots or says `//sp:keep <reason>`.
  `internal/body/reset.go` walks out from the three functions that put a seat
  back and refuses a new one that neither.
- One declaration per shared thing. SourcePawn has one flat namespace, so a
  constant declared in two packages is two `#define`s of the same name.
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
