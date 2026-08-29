# The open bugs, read against this design

Every open bead in `../tf2-mvm-bots` as of 2026-08-29, 54 of them, 21 of which
are this epic's own children. The question asked of each: what would change,
is that change a pure decision or does it need the engine, and does anything in
`docs/design.md` help fix it, help measure it, or do nothing.

The honest answer first. Of the fourteen open P1 bugs that are not this epic's
children, the design as drafted helps with two, and only to measure. `mvm-tin`,
`mvm-513`, `mvm-ed0`, `mvm-ipf`, `mvm-6rt`, `mvm-78m`, `mvm-zx0`, `mvm-0am`,
`mvm-fgs`, `mvm-jmo`, `mvm-bk8` and `mvm-ds3` are engine, nav mesh, runner or
game-side faults, and generating a threat score in Go does not touch any of
them. design.md says this in "What this does not fix" and it is right. The
interesting question is not whether the design helps them today, it is which
of them a named new capability would convert from "play a wave and squint" into
"run a test", and there are three such capabilities and six such beads.

Nothing below is measured. It is a reading of the beads and of the code they
name.

## The vocabulary used

- **Pure decision**: arithmetic over numbers already in hand. No entity index
  lookup, no nav query, no hook. This is what the body generator can carry.
- **Engine**: nav mesh, entity properties, SDKCall, DHook, geometry traces,
  path computation. Stays hand written SourcePawn.
- **Helps fix**: the fault is in code the generator would own.
- **Helps measure**: the fault is elsewhere, but a table or a report column
  from `internal/tables` makes the difference visible.

## P0

### mvm-w9b, the medic ignores a player calling for him

The code is written and off. The change was in `BiggestBody`
(`source/redbots3/nextbot_behavior.sp:1890`) and the hook in
`source/redbots3/events.sp`.

`BiggestBody` is half pure. The ranking, caller over player over the
maximum-health order, with `MEDIC_PATIENT_MARGIN` and the keep-the-beam rule,
is arithmetic over a small record per candidate. Gathering that record,
`IsClientInGame`, class, maximum health, is engine.

The design helps if `BiggestBody` is split the way `mvm-z83.6` proposes for
threat priority: a pure `PatientRank(candidate)` in Go, an engine-side loop
that fills the struct. Then the churn property that `mvm-72s` measured, seven
patient changes a wave rather than eighteen, becomes a Go property test over a
synthetic candidate list rather than a wave. `mvm-z83.7` already names medic
patient choice as a later pass.

It does not help with what keeps this bead open. The bead says it plainly: the
test-bed plays with nobody on it, `isPlayer` is false for every seat, so the
feature is a no-op there by construction. No Go capability changes that. This
needs a person on RED. Out of scope, and it should stay out of scope: building
a fake calling client is more machinery than the thing being tested.

## The P1 bugs

### mvm-666, arm order biases every A/B result

Fixed in `testbed/internal/lab`, interleaved arms and flipped round order,
tested in `order_test.go`. Already Go, already the testbed. The design's only
contribution here is `mvm-z83.8`, the noise floor, without which the re-runs
this bead asks for still have nothing to be sized against. Helps measure.

### mvm-81n, the test-bed produced runs nobody should have believed

Also already Go. Four things are listed as still open and exactly one of them
is a table problem: "results carry no record of the arm, the build or the
preconditions". That is `internal/tables` work, one Go declaration for the run
record emitting both the writer and the reader, the same shape as the wave
record already generated. The other three, crash detection by container log,
map verification, `run-native.sh` drift, are runner work with no generator in
them. Helps, narrowly.

### mvm-0lo, the test-bed cannot reproduce the faults the fixes target

The most important non-epic bead for this design, and the design has no answer
for it as drafted. Five debug convars exist as ad-hoc hand written switches:
`sm_redbots_debug_wedge_seconds`, `_refuse_ammo_paths`, `_unreachable_goal`,
`_old_wedge_recovery`, plus an unbuilt action-stack emptier for `mvm-6rt`.
They are exactly the same kind of fact as a feature: a name, a convar, a
default, and a meaning the testbed needs to arm. Named capability below.

### mvm-513, testbed refuses the second map after a changelevel

`LoadMission` execs `server.cfg` and the exec reads EOF because the rcon
connection dropped. Runner and rcon plumbing, `testbed/internal/rcon` and the
entrypoint. Engine-adjacent, no decision in it. The design does nothing.

### mvm-tin, the engineer nest relocate trips the watchdog

A reproducible watchdog trip with `sm_redbots_manager_engineer_nest_relocate`
on. Fixing it means reading what the relocate does inside a frame, which is
`ScoreNestArea`, `BestNestArea` and the path computes under them. Engine. The
design does nothing for the crash. It would help for the scoring half once
`ScoreNestArea` is split (see `mvm-dop`), but a wrong score does not trip a
watchdog.

### mvm-ed0, Mannhunt does not play in the test-bed

The population does not build. RED verified at six, map confirmed. This is
the game's population system, the popfile or the container. Runner work at
best. The design does nothing.

### mvm-bk8, medics do not heal between waves

The gate on whether the medic heals at all is engine state: is anybody hurt,
is the wave running, is the medigun out. The likely fix is overhealing, which
is a behaviour change in `behavior/medicheal.sp`. Not a pure decision.

The bead says the measurement already exists: samples every five seconds with
a healing field, healing per wave in the wave record. So this is measurable
today. The design's contribution is only that the field name and the report
column would come from one declaration rather than two. Helps measure, weakly.

### mvm-0am, the teleporter exit on Rottenburg drops you off a ledge

### mvm-fgs, the engineer walks the map to place his teleporter exit

### mvm-778, bots fall off Rottenburg

Read these three together, because they are one missing capability. `mvm-0am`
is a spot picked without regard to a drop beside it, and the bead notes it is
the third map with that shape after `mvm-fgs` on Bigrock and `mvm-1pq`.
`mvm-fgs` has the cause written out: `BuildStandPoint` snaps to the nearest nav
area within 120 units and gets the floor below a 70-unit rock, so every side is
refused from down there. `mvm-778` is a route the nav allows and a body cannot
survive.

All three are geometry. None is a pure decision today. All three are
arithmetic over nav data, which is a different statement, and it is the one
the capability below rests on.

### mvm-78m, bots never leave spawn on some community maps

kelly-cs's PR watches a prepared defender that makes no progress leaving spawn
and teleports it to the objective. The watcher is a state machine over
position over time, which is pure once the positions are in hand. The
teleport target search, capture trigger then `func_capturezone` then control
point then shortest bomb travel by nav distance, is engine.

This is the bead where a recorded trace pays best, because the fault is
defined entirely in terms of what the samples already record. Named below.

### mvm-6rt, engineer freezes during setup on Decoy

Already half solved by Go: `wave.IdleReport` in
`testbed/internal/wave/idle.go` reads the bot samples every run writes and
names any defender whose action stack holds only monitors. That is the design's
thesis working, a fault detector written in Go over data the plugin already
emits, validated against an injector. What is missing is the sweep, class by
custom loadout by manager mode, and the injector for the empty stack is
unbuilt.

The prior-art note is the real finding and it is not a Go problem: Valve's
`tf_bot_mvm_engineer_idle.cpp` never returns Done except on death and re-searches
its nest hint on a one-to-two second cooldown. If our pick runs once and
schedules nothing when it comes back empty, that is the freeze. Fixing it is
engine-side action plumbing, which is `mvm-z83.11`'s territory rather than the
body generator's.

### mvm-ds3, bots ignore the tank

The cause is written out in the notes and it is the most design-relevant cause
in the whole list: every threat scan in `nextbot_behavior.sp` is
`for (int i = 1; i <= MaxClients; i++)`, and `tank_boss` holds no player slot,
so nothing ever offers it as a target. `ThreatPriority`
(`nextbot_behavior.sp:799`) is the function `mvm-z83.6` picks as the first to
move, and its signature today is `(int threat, float rangeSq)`, an entity
index.

The design helps here in a way worth stating exactly. `mvm-z83.14` requires the
golden-input harness to take a struct rather than loose floats, which forces
`ThreatPriority` to be rewritten as a function of a threat *record*: range,
class, is-giant, is-carrier, is-player. A record has no player-slot assumption
in it. The tank then becomes a record the engine side fills from
`tank_boss` rather than a special case bolted beside the loop. That is a real
architectural win the epic gets for free, and it should be written into
`mvm-z83.6` as an acceptance criterion.

The second half of `mvm-ds3`, whether a lineup clears the tank's damage floor,
is pure arithmetic: tank health over the time it takes to walk its path against
the lineup's damage per second. That is a Go function with no engine in it and
it can be unit tested against the class damage numbers the sweeps already
produced. Helps fix.

### mvm-jmo, the red team caps at six

Cause found and fixed: nothing ever wrote `tf_mvm_defenders_team_size`, so the
game refused past six. What remains is one testbed run at twelve to see what
reads back. Convar plumbing, engine, one run. The design does nothing beyond
`mvm-z83.2`'s convar generation covering the bounds and description this convar
was missing, which is the `features.sp` class of bug and is already the epic's
core claim.

### mvm-ipf, engineer stuck inside a prop on Mannhunt

Reproduces on demand, recovery fixed and measured, cause of the crash refuted.
The next step named is symbolising core dumps. Nothing here is a decision
function. The design does nothing.

### mvm-zx0, engineers leave the nest hunting metal and get stuck

The fix shipped and the A/B measured nothing, because `path_failed` was zero
across 568 samples in both arms. The injector `sm_redbots_debug_refuse_ammo_paths`
exists and has never been run. So this is not blocked on code, it is blocked
on arming an injector and reading a specific sample field. Same capability as
`mvm-0lo`.

The candidate ranking inside `ComputeHealthAndAmmoVectors`, keeping the runners
up and advancing after three refusals, is pure once the candidate list is
built. `IsPathToVectorPossible` (`nextbot_behavior.sp:2450`) is not.

### mvm-72s, Bavarian Botbash wave 3 wipes the bot team

The longest bead and the one that best states the design's own case. Its
closing lesson, "an arm that is not replicated is a story", is `mvm-z83.8`
written from the other end. The baseline band in `docs/testbed-metrics.md`,
160s held with quartiles 154 to 192, is the noise floor for one mission, done
by hand. `mvm-z83.8` is that generalised.

Several of the things this bead measured and reverted are pure decisions that
would have been cheaper to reject in Go: the `charge_first` threat rank, the
`MEDIC_PATIENT_MARGIN` tie rule, the spacing rule. Not because Go would have
said they were wrong, it would not, but because "nothing proved the rule ever
fired" is a golden-input assertion, not a wave. Helps measure, and helps refuse
a rule that cannot fire.

## The P2 and P3 items, grouped

### Pure decisions the generator would own

- **mvm-dop**, a nest is scored for the bomb and not for the team.
  `ScoreNestArea` (`source/redbots3/util.sp:2160`) is *nearly* pure: range to
  target, height, area size, sight score. `NestCrowdingPenalty` is not, it
  loops `MaxClients` and reads sentry origins. Moving this to Go forces the
  split the bead wants anyway, a scorer over a candidate record that already
  carries the other engineers' positions and, once "where the team holds"
  exists, the team's. Helps fix.
- **mvm-eil**, the medic's patient. Same split as `mvm-w9b`. Note the warning
  in it: `action.SetHandleEntity(ACTION_HEAL_PATIENT_OFFSET, ...)` segfaulted
  the server. Generated code must never write a game offset, which is exactly
  the design's rule that generated functions take plain values and return plain
  values.
- **mvm-lfh**, the Demoman is the weakest seat. The change named is a gate,
  the sticky trap only running when no threat is visible on a twenty second
  cooldown. A gate over visibility and a timer is a pure predicate. The
  measurement is a testbed A/B against the band. Helps both.
- **mvm-abf**, **mvm-mqy**, **mvm-y8b**: upgrade priorities and loadouts.
  These are table rows, the per-class loadouts and upgrade priorities design.md
  names. Blocked behind `mvm-z83.17`, attribute names becoming int32 ids,
  because the priority chains are forty `StrEqual` tests and the subset has no
  strings. Helps fix, once that lands.
- **mvm-15t**, confirm the disposable sentry result on a second map. Pure
  measurement, one testbed run, and it is the exact case `mvm-z83.8` exists to
  size. Helps measure.
- **mvm-7hs**, sticky launchers that want playing differently. Either new
  behaviour, which is `mvm-z83.11` action templating, or a loadout table
  refusal, which is `internal/tables`. Helps.
- **mvm-d62**, Spy tells about behaviour. "A RED Heavy on a team with no Heavy
  in it is a Spy" is a comparison against the lineup the mod itself chose,
  which is table data. Cheap and pure. Helps fix.

### Needs the engine, design does nothing

- **mvm-6gi** and **mvm-ih3**, cosmetics throwing in `WearHat`
  (`cosmetics.sp:221`). A tf2utils assertion and a leaked `g_iEquipping`.
  Engine and gamedata.
- **mvm-631**, the pyro dies at close range. Airblast and retreat are new
  behaviours with traces and timings in them.
- **mvm-wb0**, something walks the engineer into 1014 885 274 on Mannworks.
  Geometry. See the nav capability below, which is the one thing that would
  change this.
- **mvm-6ak**, the ready does not register. Unconfirmed, game-side.
- **mvm-yu8**, play-test the Bot Switcher with a human on RED. Same class as
  `mvm-w9b`: needs a person.
- **mvm-ptr**, teleporters. Built, and what remains is a measurement of exit
  first against entrance first, which `mvm-72s` has already partly run.
- **mvm-dq4**, 140 ms frame hitches on Mannhattan. Path computation cost.
- **mvm-nza**, nothing senses which way the robots came. The proposed fix,
  sampling live BLU positions onto nav areas and counting visits, is engine
  gathering with a pure histogram on top. The histogram is a Go function; the
  nav area lookup is not. Half helps.
- **mvm-a0q**, nothing records what killed a defender or how far an upgrade
  got. Two fields in the wave record, which is precisely `internal/tables`
  output. This is the clearest "helps measure" in the list: until those fields
  exist, any bead of the form "the bots do not buy enough X" is unfalsifiable,
  and the bead says so. Helps measure.

### The epic's own children

`mvm-z83.5`, `.12`, `.13`, `.15`, `.19`, `.20`, `.21`, `.9`, `.14`, `.11`,
`.16`, `.17`, `.18`, `.6`, `.7`, `.8`. These are the design's work and are not
re-argued here. Two notes:

- `mvm-z83.16` is the one that decides whether this epic has produced anything
  a player could feel. Nothing else on the list matters until it closes.
- `mvm-z83.18`, randomness as a parameter, is not only a differential-test
  concern. `PickBuildArea`'s tier jitter is `GetRandomFloat(0.8, 1.75)`, and
  `mvm-72s` measured the engineer walking between spots three thousand units
  apart while re-picking every one to two seconds. A seed that can be pinned
  turns "the churn is a symptom of spots that refuse him" from a reading into a
  reproducible run. That is worth writing into the bead.

## The capabilities the design does not have

Three, in the order they would pay.

### 1. A nav-mesh model in Go

**What it is.** Parse the map's `.nav` file and the map config under
`configs/defenderbots/map/` offline, and expose to Go the areas: centre,
extents, connections, height, and the named spots the config holds. No server,
no engine, a file parser and a graph.

**What it converts.** These become unit tests over real Decoy, Rottenburg,
Bigrock and Mannworks geometry rather than waves:

- `mvm-fgs`. The cause is a 120-unit snap picking the floor below a 70-unit
  rock. That is `BuildStandPoint` against known area centres. A test asserts
  which area the snap returns for the Bigrock exit spot at -178 3921 318, and
  the fix is verified without ever running the climb.
- `mvm-0am`. "Is this spot beside a drop" is a query over neighbouring area
  heights. A test asserts no configured teleporter exit on any shipped map has
  a fall within N units. That is a check over all map configs at once, which is
  the right shape given the bead says three maps now have it.
- `mvm-778`. Same query, applied to routes rather than spots.
- `mvm-wb0` and `mvm-ipf`. 1014 885 274 is a coordinate. What the nest picker
  offers that routes through it is answerable from the mesh.
- `mvm-dop` and `mvm-nza`. `ScoreNestArea` becomes testable on real areas,
  which is the only way to know a scoring change did what was intended without
  spending a wave on it.

**What it costs and what it does not do.** It is a `.nav` parser, a format with
a version history, plus a decision about how much of `CTFNavArea` to model. It
does not simulate movement, so it cannot tell you a bot will wedge. It tells you
a spot or a score is wrong, which is what five of the six beads above actually
are.

### 2. Recorded-trace replay with an assertion layer

**What it is.** The plugin already writes a bot sample every half second with
the action stack, the position and whether the bot was firing;
`testbed/internal/wave/idle.go` documents the format and `IdleShare` already
reads it. The capability is to make that stream a generated table (one Go
declaration emitting the plugin writer and the Go reader, the same as the wave
record), and to grow a set of assertions run over every recorded run without
being asked.

`IdleReport` is the first such assertion and it works: validated against
`sm_redbots_debug_empty_stack`, 72 of 72 samples on the injected engineer
against zero for everyone else. The capability is the second, third and fourth.

**What it converts.**

- `mvm-78m`, bots never leave spawn. The fault is literally "a prepared
  defender whose positions stay inside the spawn polygon". That is an assertion
  over the existing sample stream. Today it is a report from Area 52 and
  Thriller; with the assertion it is a pass or fail on any recorded run of any
  map, including the Valve maps where the bead says it happens occasionally and
  nobody has caught it.
- `mvm-wb0`, the same wedge coordinate four times in four waves. "N defenders
  occupied the same 32-unit cube for more than T seconds" is one pass over the
  positions. That is how you find the next Mannworks wedge before a player
  reports it.
- `mvm-6rt`. The sweep the bead asks for, class by loadout by manager mode,
  is a matrix of runs each asserting `IdleReport` is empty. The assertion
  exists; what is missing is that it is a report a human reads rather than a
  test that fails the run.
- `mvm-zx0`. "Samples on a zero length path per engineer" is the exact metric
  the bead names as the one to read, and it is a field in the sample.

**What it is not.** Not a simulation. It replays what happened, it does not
predict what would happen. Every one of these still needs a real run to produce
the trace. What changes is that the run is judged by a test rather than by
reading a table, and that old traces can be re-judged when a new assertion is
written, which is how `mvm-6rt`'s near-miss was found in the first place.

### 3. Fault injectors as table rows, and the run record that names them

**What it is.** `mvm-0lo` lists five injectors, hand written, hand named, one
unbuilt. `mvm-zx0` is open only because its injector has never been armed.
`mvm-81n` lists "results carry no record of the arm, the build or the
preconditions" as still open. These are the same gap. The capability is one Go
table of injectors beside the feature table, emitting the convars on the plugin
side and the arming and recording on the testbed side, so that a result file
states which injector was armed at what value and no A/B can silently compare a
fix against itself.

**What it converts.** `mvm-0lo` names the consequence exactly: "until it
existed every A/B compared the fix against itself, which is why a day of runs
came out identical and why I kept reading noise as a result". `mvm-zx0` is
one arming away from an answer. `mvm-6rt` needs one injector built. This is
the cheapest of the three capabilities and it is squarely inside what
`internal/tables` already does.

### What is deliberately not proposed

A Go-side simulation of a wave. Nothing in the open list would be settled by
one. Every bead that looks like it wants a simulator, `mvm-72s`, `mvm-ds3`,
`mvm-631`, is really asking about combat outcomes, and a simulator accurate
enough to answer those is a second game. `mvm-72s` already spent a session
proving that a plausible reading of four attempts is a story, and a simulator
would produce those readings faster rather than better.

Nor a deterministic full-run seed. `mvm-z83.18`'s parameterised randomness
covers the decision functions, which is where it can be honoured. The server,
the population and the physics are not ours to seed, and claiming a
reproducible run would be claiming more than the machinery can deliver.
