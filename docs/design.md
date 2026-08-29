# Authoring the bot AI in Go

A draft. Nothing here is measured yet, and the first bead is the one that says
whether the rest is possible at all.

## The complaint

Three things go wrong often enough to be the reason for this document.

A name inserted in the wrong place renames a convar. `features.sp` holds an enum
and a parallel array of strings, and the compiler counts the entries and says
nothing about their order. `ammo_failover` sat at `FEATURE_WATCH_IDLE_BOTS` for a
release, so `sm_redbots_feature_watch_lurking_snipers` drove the idle watchdog,
an A/B armed the wrong feature, and the number it produced was read as a
measurement. The comment above the array now asks the reader to be careful. That
is not a fix, it is a note about a fix nobody wrote.

The decision code cannot be run without a game server. Threat scoring, upgrade
order, nest spot choice and patient choice are arithmetic over numbers the engine
hands us, but they live inside actions that only exist inside a running mission,
so the only way to ask whether a change to them is right is to play a wave. A
wave takes minutes and answers with noise.

The measurement is not trusted. mvm-666 found arm order biasing damage and wave
counts, not only crashes, and mvm-81n lists five ways a run could look ordinary
and measure nothing. Both are fixed. What is left is that we have no noise floor,
so we cannot say whether a difference is a result.

None of the three is a SourcePawn problem. They are a problem of one fact being
written down twice, and of the interesting code being unreachable from a test.

## What Go is for, and what it is not for

Not a rewrite of the plugin. The mass of this repository is SM natives, DHooks,
CBaseNPC actions, gamedata offsets and entity access. A transpiler cannot carry
any of it, and the one that exists for this, SourceGo, is a beta that supports a
subset and asks you to drop into raw SourcePawn through `__sp__()` when it runs
out. Aiming a transpiler at the whole plugin means owning a compiler and still
writing the hard parts by hand.

Go owns three things instead.

**The tables.** Features, their convars, their defaults, the telemetry field
names, the action names the report groups by, the upgrade priorities, the
per-class loadouts. Each of these is one fact written in Go once, generating the
SourcePawn enum, the name array and the `CreateConVar` calls on one side, and the
testbed's parser and report columns on the other. The `features.sp` class of bug
stops existing, because the enum and the names come out of the same slice.

**The decision functions.** The pure ones. A threat score from distances, classes
and health. An upgrade order from money and what is already bought. A nest spot
from a list of candidates. These take numbers and return a number or a choice,
they have no engine in them, and they are where the regressions are. Written in
Go, unit tested and fuzzed in Go, generated into SourcePawn.

**The experiment.** The testbed is already Go and already refuses bad runs. It
gains the arm definitions, from the same feature table, plus a noise floor and a
stopping rule.

Everything that touches the engine stays hand written SourcePawn. Generated code
is called by it and never calls back into it: a generated function takes a struct
of plain values and returns a plain value. That rule is what keeps the generator
small.

## The part that has to be proved first

A generated function is worth nothing if we only believe it matches the Go it
came from. The check is a differential test: the same golden inputs through the
Go function and through the generated SourcePawn, asserting the same output.

That needs the generated SourcePawn to run with no game server. SourcePawn ships
`spshell`, a standalone VM, and running plugin code under it for CI is a thing
people have asked for. Whether our generated functions actually load and run
there, with no `sourcemod.inc` and no natives, is not something to assume. It is
the first bead, and if the answer is no, the design changes: the fallback is a
test plugin that runs the golden inputs on a dedicated server at startup and
prints a verdict, which is slower and still worth having, but a worse deal.

## The Go subset

Restricted on purpose, and enforced. Not "the subset SourceGo happens to support"
but "the subset we write", checked by a pass that refuses everything else with a
line number. Roughly: functions, the fixed size numeric types, `float`, bool,
fixed length arrays, structs of those, `if`, `for`, `switch`, and calls to other
functions in the same package. No slices that grow, no maps, no interfaces, no
closures, no allocation, no strings beyond fixed buffers. Anything a bot decision
needs that this cannot express is a sign the function is not pure and belongs on
the hand written side.

Refusing is the whole value. A transpiler that quietly does something reasonable
with a construct it half supports is how you get a generated plugin that compiles
and is wrong.

## Order of work

One function end to end before anything else. Threat priority is the candidate:
it is already a feature switch, it is arithmetic, and it has a measurement
attached. Prove the loop, tables through generation through differential test
through a testbed run, on that one function. Then widen a class at a time.

The build gate at the end: `testbed/build.sh` regenerates and fails if a
generated file changed, so a hand edit to generated SourcePawn cannot be
committed.

## What this does not fix

Bots stuck on geometry, nav mesh failures, engineers wedged in props. Those are
the open P1s and none of them is a decision function. This makes the decisions
testable and the experiments trustworthy. It does not make the bots walk.

## Why not fork SourceGo

The goal is right: hand write only the SourcePawn that has to be SourcePawn.
The measurements say that floor is low. In 24.7k lines there are 121 SDKCall
sites, 34 DHook sites and about 70 raw address or entity-data reads. Under 250
places genuinely touch the engine in a way a generator cannot. Everything else
is arithmetic, property reads and action plumbing.

Forking SourceGo is the expensive route to that floor.

It is 532 KB across 94 Go files with one test file of 190 bytes, last pushed in
September 2022. It was abandoned mid rewrite and both halves are still in the
tree: `srcgo/ast_to_sp.go` and `srcgo/ast_transform.go` at 66 KB, and
`rewrite/srcgo/pass1..pass10` with `rewrite/sptools/` at about 210 KB. Taking
the fork means choosing which half is real and owning both until you have.

The 210 KB half is a SourcePawn front end: a tokenizer, a preprocessor, two
parsers, one still named `old-parser.go`, and a typechecker. We do not need a
SourcePawn front end. `spcomp` is the front end, and it is the one whose opinion
counts. That is dead weight from the first commit.

The 36 files under `sourcemod/`, plus `tf2.go` and `sdktools.go`, are the
SourceMod API transcribed into Go by hand. Four years stale, and they cover none
of what this plugin leans on: tf2utils, tf2attributes, tf_econ_data, CBaseNPC,
actions.ext. Transcribing those is the single largest cost of the fork, and it
is a cost that recurs every time an include changes.

It does not need to be a cost at all. All 67714 lines of those includes are
already in this repository under `testbed/build/`, and a `native` declaration is
a one line signature. Bindings are generated from the includes, and they cannot
go stale.

And SourceGo has no methodmaps and no inheritance. The structure of this AI is
162 lines of `BehaviorAction` subclassing with OnStart, OnUpdate, OnEnd,
OnSuspend and OnResume. That is the part that matters most and the part it
cannot express.

What is worth taking, and it is MIT so take it with attribution: the pass list.
`pass1_illegal_code` is the refusing checker this draft already wants.
`pass3_merge_rettypes` turns multiple returns into by-reference parameters,
`pass9_mutate_ranges` handles range loops, `pass10_mutate_no_ret_calls` handles
discarded results. Those are hard won mappings, about 30 KB of them, and reading
them is cheaper than rediscovering them.

## Three generators, not one transpiler

Splitting it this way keeps each piece small enough to be tested.

**Bindings, from the includes.** Parse the `native` and `methodmap` declarations
under `testbed/build/` and emit Go declarations. Mechanical, no language design,
regenerated when a dependency moves.

**Actions, from a Go type.** A Go type implementing an action interface emits the
`BehaviorAction` subclass, the constructor and the callback wiring. This is
templating, not translation, and it is where most of the plugin's line count
lives.

**Bodies, from a restricted Go subset.** The part that is a real compiler, kept
as small as the subset allows, refusing everything else with a line number, and
checked against the Go it came from by golden inputs under spshell.

## The Go standard this is held to

From `starter-kits`: generated code is gitignored and the build verifies that
generation succeeds and the output compiles. One `make check` is the gate and CI
runs Make targets rather than raw commands, so the gate runs the same locally.

This repository has no Makefile today, only `testbed/build.sh`. It gets one.
