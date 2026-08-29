# Parity, as a work-list

What "feature parity with the plugin, generated from Go, and better" contains and
what it costs. Read `design.md` first, then `closed-bugs-vs-design.md` and
`open-bugs-vs-design.md`. This file is the inventory those three assume.

Every part of `../tf2-mvm-bots/source/` is classified into exactly one bucket.
The bucket totals, the hand-written floor, the port order and the delete list are
at the end. Where a number is a guess it says so.

## How this was measured

Two passes, and they are separate on purpose.

The mechanical pass is grep and a brace-matching script over all 47 `.sp` files.
It produces numbers that can be re-run: lines per file, lines containing an engine
call, function spans, and how many engine calls each function contains. Nothing in
it is a judgement.

The reading pass is four passes over the files themselves, assigning every line to
a bucket. That is a judgement, and the two passes disagree by about 8 percent on
the hand-written total (13,777 read against 14,942 measured at whole-function
granularity). The gap is prose and tuning constants that sit inside a function
that touches the engine: the reading pass calls them TABLE, the script counts the
whole function. Both numbers are given below rather than reconciled, because the
disagreement is the honest error bar.

Line counts were taken while another session was editing the tree, so they move by
a few lines. The behavior directory read as 27 files and 8,352 lines, not the 30
files and 8,395 lines the brief quotes. Treat every figure as accurate to about
one percent, not to the line.

## The buckets

- **GENERATED-DECISION**: pure arithmetic or choice, no engine calls. Plain
  values in, a number or a choice out. Moves to Go under the `mvm-z83.4` subset.
- **GENERATED-SHAPE**: boilerplate a template emits. Action lifecycle
  (`OnStart`/`Update`/`OnEnd`/`OnSuspend`/`OnResume`, constructors, registration),
  and for the non-behaviour files the mechanical declaration blocks: SDKCall prep,
  DHook registration, offset lookup, convar creation, menu item construction.
- **GENERATED-TABLE**: a fact written twice today. Features, spots, loadouts,
  upgrade chains, weapon tuning, cosmetics, wave and telemetry record fields, and
  the tuning constants that other files' constants depend on.
- **HAND-WRITTEN**: touches the engine. SDKCall, DHook, offsets, entity property
  access, nav mesh, traces, event hooks, natives with side effects.
- **DELETE**: dead, superseded, or a workaround the new design removes.

A line lands in one bucket only. Blank lines and comments go to the bucket of the
code they sit in, which is why the TABLE column is large: several files are more
rationale than code, and that rationale is provenance a table format has to carry.

## Behaviour files: 27 files, 8,352 lines, 26 actions

One action per file except `medicheal.sp`, which no longer holds one.

| file | lines | decision | shape | table | hand | delete | responsible for |
|---|---:|---:|---:|---:|---:|---:|---|
| engineeridle.sp | 1215 | 27 | 44 | 67 | 988 | 89 | the engineer's whole standing state: nest, haul, repair, wrangle, suspend into the build actions |
| upgrade.sp | 1201 | 84 | 16 | 378 | 723 | 0 | ranking every purchasable upgrade and buying down the list |
| attacktank.sp | 633 | 50 | 19 | 309 | 255 | 0 | picking a tank, the right weapon for it, and a standoff range |
| engineerbuildteleporter.sp | 622 | 38 | 49 | 112 | 397 | 26 | the teleporter pair, with crouch-jump climbing onto authored ledges |
| engineerbuilddispenser.sp | 431 | 18 | 14 | 80 | 319 | 0 | choosing a dispenser spot and getting the game to accept the placement |
| spycheck.sp | 409 | 30 | 41 | 52 | 286 | 0 | team-shared Spy paranoia and the "who appeared who was not there" tell |
| getammo.sp | 387 | 70 | 45 | 43 | 229 | 0 | walking to the nearest reachable ammo, and the shared candidate scan |
| engineerbuildsentrygun.sp | 341 | 20 | 25 | 74 | 222 | 0 | getting a sentry placed and facing outward |
| movetofront.sp | 305 | 37 | 16 | 49 | 203 | 0 | walking to the robot spawn between rounds and marking ready |
| attack.sp | 258 | 59 | 19 | 21 | 159 | 0 | picking an enemy, closing to weapon range, strafing while shooting |
| gethealth.sp | 238 | 60 | 21 | 10 | 147 | 0 | the same as getammo, for health, with a ratio-scaled radius |
| stickytrap.sp | 235 | 15 | 9 | 64 | 147 | 0 | scattering stickies around the bomb before a fight |
| engineerbuilddisposable.sp | 230 | 15 | 16 | 30 | 169 | 0 | disposable minis on a ring around the real sentry |
| gotoupgrade.sp | 174 | 5 | 15 | 37 | 117 | 0 | walking to an upgrade station |
| markgiant.sp | 171 | 10 | 15 | 3 | 132 | 11 | Fan O'War Scout applying mark-for-death to a giant |
| spylurk.sp | 164 | 29 | 22 | 6 | 107 | 0 | stalking, circle-strafing behind a target, stabbing |
| evadebuster.sp | 158 | 15 | 20 | 26 | 97 | 0 | running to the nav area furthest from a sentry buster |
| collectmoney.sp | 156 | 29 | 23 | 17 | 87 | 0 | walking to the best uncollected credit pack |
| guardpoint.sp | 156 | 28 | 16 | 3 | 109 | 0 | holding ground near a capturable point |
| campbomb.sp | 146 | 20 | 23 | 0 | 103 | 0 | guarding the dropped bomb instead of chasing |
| spysapplayer.sp | 133 | 33 | 36 | 0 | 64 | 0 | putting a sapper on an enemy player |
| medicrevive.sp | 128 | 18 | 20 | 0 | 90 | 0 | walking to a revive marker and beaming it |
| attackforuber.sp | 128 | 17 | 30 | 0 | 81 | 0 | melee to build charge, including the taunt-kill attempt |
| spysap.sp | 106 | 6 | 52 | 0 | 48 | 0 | sapping buildings |
| destroyteleporter.sp | 95 | 9 | 23 | 0 | 63 | 0 | breaking an enemy teleporter |
| medicheal.sp | 74 | 0 | 4 | 0 | 70 | 0 | not a behaviour: a post-mortem on the deleted heal action plus a dump command |
| collectnearmoney.sp | 58 | 12 | 25 | 0 | 21 | 0 | picking up a nearby pack when no threat is visible |
| **total** | **8352** | **754** | **658** | **1381** | **5433** | **126** | |

Two things stand out and neither is the decision column.

The shape is completely uniform. Every file opens with the same
`ActionsManager.Create` plus callback assignment, every `OnStart` calls
`SetMinLookAheadDistance(GetDesiredPathLookAheadRange(actor))`, and 7 of the 8
smallest files end `Update` with the identical repath-then-`m_pPath.Update` tail.
658 lines, no variance. `spysap.sp` is 49 percent shape and
`collectnearmoney.sp` is 43 percent. This is what `mvm-z83.11` is for and it is
the least risky line count in the whole plugin.

The table column is dominated by three files. `upgrade.sp` holds 335 lines of
`StrEqual(attribute, "...") return N` across `LoadoutUpgradePriority`,
`ClassUpgradePriority` and `GeneralUpgradePriority`, keyed on strings out of
`mvm_upgrades.txt`. `attacktank.sp` holds 296 lines of nine per-class weapon score
tables with the same `switch (slot)` fallback repeated eight times verbatim. The
engineer build files hold 250 lines of tuning constants with their rationale, and
`DISPOSABLE_BUILD_REACH`/`DISPOSABLE_TRY_POINTS` are copies of the sentry file's
values with the press-settle fix missing from the copy.

## Support files: 13 files, 3,348 lines

| file | lines | decision | shape | table | hand | delete | responsible for |
|---|---:|---:|---:|---:|---:|---:|---|
| loadouts.sp | 502 | 0 | 173 | 34 | 295 | 0 | per-class weapon pools and applying a custom loadout |
| cosmetics.sp | 502 | 25 | 65 | 14 | 398 | 0 | hats and unusual effects, pools built from the item schema |
| dhooks.sp | 363 | 0 | 119 | 0 | 241 | 3 | detour and hook registration, and the ten callback bodies |
| debug_faults.sp | 319 | 0 | 62 | 0 | 247 | 10 | fault injection: wedge, empty stack, refuse ammo paths, unreachable goal |
| sdkcalls.sp | 295 | 0 | 278 | 0 | 16 | 1 | SDKCall preparation and 16 thin invocation wrappers |
| features.sp | 236 | 45 | 13 | 178 | 0 | 0 | the 22 named on/off switches and their convars |
| demoman_stickies.sp | 216 | 60 | 0 | 44 | 112 | 0 | whether to detonate, and whether to hold the sticky launcher |
| tf_upgrades.sp | 197 | 10 | 44 | 79 | 64 | 0 | the memory layout of CMannVsMachineUpgradeManager |
| blu_assist.sp | 188 | 54 | 0 | 45 | 75 | 14 | scaling robot health down when RED is short-handed |
| weapon_tuning.sp | 179 | 0 | 0 | 162 | 17 | 0 | per-item-definition engagement ranges |
| offsets.sp | 148 | 0 | 13 | 23 | 112 | 0 | gamedata offset resolution and ten accessors |
| medic_uber.sp | 146 | 51 | 0 | 37 | 55 | 3 | when to fire the charge, per medigun type, and the shield |
| archipelago.sp | 57 | 0 | 35 | 0 | 22 | 0 | optional-native bridge for Archipelago cash bundles |
| **total** | **3348** | **245** | **802** | **616** | **1654** | **31** | |

The three files the epic points at as the irreducible core are the most
generatable files in the tree, and this is the most important single finding in
the inventory.

| file | declaration and setup | genuinely hand-written body |
|---|---:|---:|
| sdkcalls.sp | 212 (24 handle declarations, `InitSDKCalls` 25-212) | 82, of which **16 are actual `SDKCall(...)` invocations** |
| dhooks.sp | 119 (declarations, `InitDHooks`, `RegisterDetour`/`RegisterHook`) | 241 (the ten callbacks) |
| offsets.sp | 85 (`InitOffsets`, `SetOffset`, `GetOffset`) | 57 (the ten accessors) |

806 lines total, 416 of which derive entirely from a tuple of (name, call type,
conf type, parameter list, return type) and could be emitted from the same
gamedata file `spcomp` already reads. Every one of the 16 SDKCall setup blocks
types its signature name a second time inside a `LogError`; `offsets.sp` types its
ten keys a third time inside a `TESTING_ONLY` dump. That is the `features.sp` bug
in three more places, and it has not bitten yet only because a wrong error string
is not a wrong convar.

`weapon_tuning.sp` is 91 percent table: a 15-weapon lookup written as control flow,
with one `GetEntProp` at the top. `cosmetics.sp` is the opposite and says so in its
own header: 40 percent prose carrying five separate incident write-ups, and 4 lines
a generator could own.

## Core files: 7 files, 12,988 lines

| file | lines | decision | shape | table | hand | delete | responsible for |
|---|---:|---:|---:|---:|---:|---:|---|
| nextbot_behavior.sp | 3364 | 520 | 400 | 120 | 1720 | 604 | action dispatch, threat selection, monitor overrides, pathing, three watchdogs |
| tf2_defenderbots.sp | 3359 | 520 | 700 | 240 | 1791 | 108 | plugin entry: roster, composition, convars, commands, `OnPlayerRunCmd`, map configs |
| util.sp | 3286 | 430 | 210 | 260 | 2266 | 120 | shared queries, econ item creation, bomb info, and the whole nest-scoring system |
| menu.sp | 1005 | 60 | 500 | 380 | 65 | 0 | vote, setup and preference menus |
| botaim.sp | 900 | 300 | 120 | 90 | 390 | 0 | aim simulation: head tracking, fire and reload timers, weapon classification |
| player_pref.sp | 580 | 90 | 130 | 180 | 160 | 20 | KeyValues-backed class and weapon preferences, server loadout seats |
| events.sp | 494 | 40 | 90 | 20 | 298 | 46 | game event hooks and the timers they arm |
| **total** | **12988** | **1960** | **2150** | **1290** | **6690** | **898** | |

`menu.sp` is 93 percent generatable and touches the engine on 2 lines. Lines
153-502 are 350 lines of hand-expanded per-class per-slot weapon loops over arrays
that already exist in `loadouts.sp`. It is a table indexed by (class, slot) written
as an if-tree, and it is the largest single block of duplicated data in the plugin.

The decision functions the epic names, with spans:

| function | file:span | lines | bucket |
|---|---|---:|---|
| `GetDesiredBotAction` | nextbot_behavior.sp:2213-2350 | 138 | DECISION |
| `ShouldTakeUpPosition` | nextbot_behavior.sp:2371-2382 | 12 | DECISION |
| `CanGuardTheHatch` | nextbot_behavior.sp:2389-2400 | 12 | DECISION |
| `GetUpgradePostAction` | nextbot_behavior.sp:2402-2422 | 21 | DECISION |
| `ThreatPriority` | nextbot_behavior.sp:799-836 | 38 | DECISION |
| `SelectMoreDangerousThreat` | nextbot_behavior.sp:838-935 | 98 | HAND, 27 lines of DECISION inside |
| `BiggestBody` | nextbot_behavior.sp:1890-1994 | 105 | DECISION |
| `CTFBotTacticalMonitor_Update` | nextbot_behavior.sp:1634-1735 | 102 | HAND, writes a CountdownTimer at `action + 0x70` |
| `ComputeHealthAndAmmoVectors` | behavior/getammo.sp:226-287 | 62 | HAND |
| `ScoreNestArea` | util.sp:2160-2185 | 26 | DECISION |
| `BestNestArea` | util.sp:2233-2257 | 25 | SHAPE (argmax over an ArrayList) |
| `NestCrowdingPenalty` | util.sp:2198-2226 | 29 | HAND |
| `BuildStandPoint` | util.sp:630-661 | 32 | HAND, 16 lines of trig inside |
| `SentryNeedsMetal` | behavior/engineeridle.sp:772-784 | 13 | HAND, pure once four reads are hoisted |
| `CanRepairFromRange` | behavior/engineeridle.sp:844-880 | 37 | HAND (a trace) |
| `NearestFreeExitSpot` | behavior/engineerbuildteleporter.sp:559-581 | 23 | HAND |
| `NearestConfiguredSpot` | behavior/engineerbuildteleporter.sp:604-622 | 19 | DECISION, and shared with the dispenser file |

The whole of action selection, the function family that produced five shipped
freezes, is 183 lines. `internal/actionsel` already expresses it in 358 lines of Go
plus 474 of test.

## Totals

| bucket | lines | share |
|---|---:|---:|
| GENERATED-DECISION | 2,959 | 12.0% |
| GENERATED-SHAPE | 3,610 | 14.6% |
| GENERATED-TABLE | 3,287 | 13.3% |
| HAND-WRITTEN | 13,777 | 55.8% |
| DELETE | 1,055 | 4.3% |
| **total** | **24,688** | |

Generatable, adding the first three: **9,856 lines, 40 percent of the plugin.**

## The true hand-written floor

The epic says "121 SDKCall sites, 34 DHook sites and about 70 raw address reads.
Under 250 places genuinely touch the engine in a way a generator cannot." The site
counts are right. The conclusion drawn from them is not.

**A site is not a line, and a line is not a function.** The three counting levels,
measured:

| level | count | source |
|---|---:|---|
| SDKCall / DHook / address-read sites | 228 | 121 in `sdkcalls.sp`, 37 in `dhooks.sp`, ~70 address and entity-data reads |
| lines containing any engine call | 1,389 | grep over 47 files, one line counted once |
| functions containing at least one engine call | 436 of 756 | brace-matching script |
| lines inside those 436 functions | 14,942 | same script |

The 250-site figure counts only three call kinds. It omits `GetEntProp` and its
family, nav mesh queries, traces, `TF2Util_`/`TF2Attrib_`/`TF2Econ_` natives, event
hooks and timers, all of which are equally impossible to generate and all of which
are more common. Counting every kind gives 1,389 lines, 5.6 times the site figure
before any surrounding code is counted at all.

What the floor actually is depends on how far you are willing to split a function
into a pure core and an engine shell. The distribution says where the work is:

| engine calls in the function | functions | their lines |
|---|---:|---:|
| 0 | 320 | 4,485 |
| 1 | 163 | 2,632 |
| 2 | 86 | 2,257 |
| 3 or more | 187 | 10,053 |

- **320 functions, 4,485 lines, are already pure.** They can move to Go with no
  split at all. This is the free half of the DECISION and SHAPE buckets.
- **249 functions, 4,889 lines, have one or two engine calls.** Hoist the call to
  the caller and the rest is pure. Each is a small, individually reviewable edit.
- **187 functions, 10,053 lines, have three or more.** These are the plugin. They
  stay SourcePawn.

So:

| floor | lines | what it assumes |
|---|---:|---|
| implied by the epic | ~1,200 | 250 sites at 4.7 lines each, the measured density of `sdkcalls.sp` |
| absolute lower bound | ~2,300 | every one of the 436 functions reduced to a marshalling shell: 1,389 engine lines plus ~2 lines of signature and brace each |
| **realistic floor** | **~10,800** | split only the 249 functions with one or two calls; the 187 dense ones stay whole, plus ~750 lines of shell for the splits |
| floor if nothing is split | 14,942 | a function that touches the engine stays SourcePawn entirely |
| floor from the reading pass | 13,777 | line-by-line judgement, sits between the last two as expected |

**The earlier estimate is low by roughly nine times.** The realistic floor is about
10,800 lines, 44 percent of the plugin, against an implied 1,200. The absolute
lower bound of 2,300 is not reachable in practice: getting there means 436
individual function splits, each one a hand edit to code with no test, in a plugin
whose bug record is dominated by exactly this kind of change.

This does not refute the design. It resizes it. 40 percent of the plugin is
generatable and that is a real number. But "hand write only the SourcePawn that has
to be SourcePawn" describes a 10,800-line SourcePawn plugin, not a 1,200-line one,
and the two imply completely different amounts of work.

## Which actions are cheapest, and which are dangerous

**Cheapest.** Ranked by shape share, absence of engine density, and absence of any
closed bug touching them.

| action | lines | shape | hand | why it is cheap |
|---|---:|---:|---:|---|
| collectnearmoney | 58 | 25 | 21 | smallest action in the tree, 43% pure template, no bug history |
| destroyteleporter | 95 | 23 | 63 | one action, one target query, no state carried between updates |
| spysap | 106 | 52 | 48 | 49% shape; three of its five callbacks only toggle one flag |
| medicrevive | 128 | 20 | 90 | self-contained, one nav search, no cross-file state |
| attackforuber | 128 | 30 | 81 | self-contained, no shared globals |
| campbomb | 146 | 23 | 103 | reads shared bomb info but writes nothing others read |
| evadebuster | 158 | 20 | 97 | the decision is one argmax over area centres |
| collectmoney | 156 | 23 | 87 | the cost function is two floats in, one float out |

Start with `collectnearmoney` and `spysap` for the template, and `collectmoney`
for the first generated body: `MoneyPackCost(distance, timeLeft) -> float` followed
by an argmin is the cleanest pure function in the whole behaviour directory.

`mvm-z83.11` names `movetofront` or `gethealth` as the first template target. The
inventory disagrees. `movetofront` is on the path of `mvm-7kr` and `mvm-pvt`, it
carries a stuck counter and a `NudgeTowardsGoal` fallback that manually shoves the
bot, and it lies about readiness when it gives up short of the front. `gethealth`
shares a byte-identical `ShouldAttack` body with `getammo` and a duplicated
ratio-to-range block, so porting it forces a decision about the duplicate on day
one. Both are worse first choices than `collectnearmoney`.

**Dangerous.** Grounded in the closed record, which the design review already
scored: engineer idle, upgrade and the engineer build actions produced the most
bugs.

| action | lines | closed and open beads against it | why it is dangerous |
|---|---:|---|---|
| engineeridle | 1215 | mvm-1b7, mvm-hnb, mvm-7kr, mvm-cf3, mvm-tin, mvm-6rt, mvm-zx0, mvm-dh8, mvm-wb0 | 81% hand-written; one 575-line `Update`; four independent give-up clocks; a 0.1 s repeating timer walking one client per tick; the file that produced both P0 freezes |
| upgrade | 1201 | mvm-gwq, mvm-654, mvm-2uj, mvm-abf, mvm-mqy, mvm-y8b | 378 lines of string-keyed priority table blocked behind `mvm-z83.17`; a spend cap whose own comment says nobody knows whether the game or the ranking is broken; a refusal memo per trip |
| engineerbuildteleporter | 622 | mvm-1pq, mvm-0am, mvm-fgs, mvm-1yo, mvm-dx2 | 64% hand-written; five separate geometry bugs; the exit ring at line 431 has no drop check and 32% of Decoy's ring sits beside a fall |
| engineerbuilddispenser | 431 | mvm-oh7, mvm-y9d | 74% hand-written and mostly workaround: a press debounce, a 45 s give-up clock, a refused-spot retry list compensating for a path query that lies |
| engineerbuildsentrygun | 341 | mvm-hnb, mvm-ipf | two timers chasing each other because the global stuck watchdog restarts the action and re-arms its own deadline |
| attacktank | 633 | mvm-ds3 | the tank does not occupy a player slot, so the whole threat path has to change shape before this action means anything |

The pattern the record shows and the inventory confirms: **danger is a tick
boundary, not a line count.** `mvm-oh7` is one frame between pressing fire and
asking whether a building exists. `mvm-74t` is a return statement sitting above a
call. `mvm-7kr` is a flag cleared on the wrong event. A generated pure function
cannot see any of them, and porting the arithmetic out of these files leaves every
one of those bugs exactly where it is. That is not an argument against porting
them. It is an argument for porting them last, when the mechanism is boring.

## The ordering that minimises the half-migrated state

The half-migrated plugin is the worst state and it will exist for months. The way
to make it least bad is not to shorten it. It is to make it **describable in one
sentence that never changes.**

Migrating file by file fails that test. After six weeks the rule is a list of 47
file names that changed last Tuesday, and the question "is this file mine to edit?"
needs a lookup. Migrating by kind passes it: at every moment, the rule is one line.

**Rule 1: migrate a whole kind before starting the next.** Finish all tables, then
all shape, then decisions. The sentence is "every table lives in Go; everything
else is SourcePawn", then "every table and every action skeleton lives in Go", and
so on. Three sentences over the life of the project, each true for months.

**Rule 2: the answer to "who owns this file" is its path, not memory.** Generated
output lands in one directory, gitignored, and `mvm-z83.9` makes a hand edit to it
fail the build. A contributor never has to know the migration's state; they only
have to know whether the file they opened is under the generated path.

**Rule 3: a fact is never live in both places, not even for a day.** This makes
`mvm-z83.16` the gate on everything, and it is already the epic's own judgement:
until the generated `features.sp` is in the plugin, the Go table is pure cost with
a second copy of the truth beside it. Adoption is not the last step of a table, it
is the step that finishes it.

**Rule 4: never a long-lived branch.** Every step ships. The plugin compiles and
plays at every commit, and `make check` regenerates and runs `spcomp` over the
output. A migration branch that cannot be played is a migration nobody can measure.

The order, then:

1. **Finish and adopt the tables that already exist** (`mvm-z83.16`). Features and
   the wave record round-trip today and have never been through `spcomp`. Nothing
   else in the epic matters until a player could feel it. Cost: small. Risk: a
   noisy first diff, and the two known normalisations are already documented.

2. **The rest of the tables, largest duplication first.** `menu.sp` 153-502 and
   `loadouts.sp` collapse together, because the menu is an if-tree over the
   loadout arrays: 380 + 34 lines of data behind 523 lines of dispatch.
   `weapon_tuning.sp` next at 91 percent table. Then the three declaration files:
   `sdkcalls.sp` at 212 setup lines from a gamedata tuple, `dhooks.sp` at 119,
   `offsets.sp` at 85. Then `attacktank.sp`'s nine weapon tables. `upgrade.sp`'s
   335 lines wait for `mvm-z83.17`, since the subset has no strings.
   These are the safest lines in the plugin: a table that round-trips byte for byte
   cannot change behaviour, and `mvm-z83.2` and `mvm-z83.3` have already proved the
   proof mechanism on 22 features and 112 fields.

3. **Action selection** (`mvm-z83.22`), and it is already built. 183 SourcePawn
   lines, five shipped freezes, and the Go side enumerates 1,425,408 reachable
   combinations. What is missing is the differential test tying `Shipped` to the
   file, which is `mvm-z83.5` under spshell. Do that before anything else moves,
   because it is the only port whose correctness argument is a proof rather than a
   comparison.

4. **Delete the three watchdogs**, immediately after step 3 and not before. They
   exist because nothing could prove the selection table total. Once it can, they
   are 670 lines of symptom. Deleting them is the first time the epic makes the
   plugin smaller instead of larger, and it is the moment the half-migrated state
   starts paying rather than costing.

5. **The action template** (`mvm-z83.11`), on the cheap actions, generated beside
   the hand-written one and diffed until the diff is empty. 658 lines of behaviour
   shape with no variance in it.

6. **The pure decisions, cheapest first.** `collectmoney`'s cost function,
   `evadebuster`'s argmax, `attack`'s perpendicular step, `spylurk`'s
   behind-the-target geometry, `spycheck`'s paranoia range. Then threat priority
   (`mvm-z83.6`) as a record rather than an entity index, which removes `mvm-ds3`
   as a side effect. Then nest scoring and build placement (`mvm-z83.33`), where
   `internal/navmesh` can check the answer offline.

7. **The engineer files last.** By then the template is boring, the tables are
   gone, and the differential test exists. Port them when a mistake is cheap.

The one ordering mistake worth naming: do not start with the transpiler on a hard
function to prove it can be done. The record already scored that bet. Over 31
closed defects, the generator prevented zero and would have caught ten, and three
of those ten are the noise floor and the golden harness rather than the generator.

## What should be deleted rather than ported

1,055 lines are counted as DELETE above. The specific items, with spans.

**The three stall watchdogs, 670 lines.** All in `nextbot_behavior.sp`. They are
the plugin's answer to a hole in action selection, one case at a time, and
exhaustiveness makes all three unnecessary.

| watchdog | spans | lines |
|---|---|---:|
| `FEATURE_WATCH_IDLE_BOTS` stuck and idle watchdog | 1304-1347, 1400-1539, 1540-1633, call site 1679, `features.sp` 40 and 159-163 | ~285 |
| `FEATURE_WATCH_LURKING_SNIPERS` and the sniper stall rescue | 1333-1372, 1374-1398, 1428-1456, `events.sp` 147, `features.sp` 41 and 165-177 | ~112, ~83 net of overlap |
| the spawn-exit watchdog, a third one not on the list | 150-232, 233-259, 390-537, 538-581 | ~302 |

The spawn-exit watchdog is the find here. It has the same shape as the two named
ones, it is bigger than either, and it carries two debug commands
(`Command_DumpSpawnNav`, `Command_RecoverSpawnBots`, 44 lines) whose only purpose
is exercising it. One caveat: `RecoverDefenderFromDisconnectedSpawn` at 473-494 is
called from `GetDesiredBotAction` at 2216, so it needs a replacement rather than a
plain deletion.

**Other watchdogs and workarounds, ~190 lines.**

- `events.sp` 181-226, the staggered behaviour-reset ticker: a hand-rolled
  one-bot-per-tick spreader to avoid a frame spike, superseded by the
  `PATHS_PER_FRAME` budget already in `nextbot_behavior.sp` 107-145. 46 lines.
- `engineeridle.sp` 108-132 and its five call sites, `ReportEngineerStall`: a stall
  reporter whose own comment says "one of its early returns owns the wave and there
  is no way to tell which from outside". That is exactly what exhaustive selection
  answers. ~40 lines.
- `engineeridle.sp` 786-841, the range-repair stall counter, plus
  `tf2_defenderbots.sp` 494 and 551-561 (`Native_RangeRepairStalls`). Its comment
  says "watched, not acted on" and nothing in this plugin consumes it. ~50 lines,
  and it is a delete only if no external stats plugin depends on the native.
- `engineerbuildteleporter.sp` 322-337 and 195-204, `SayClimb` and the
  `[teleclimb] not asked` block: bisect instrumentation for a single diagnosed bug,
  `mvm-fgs`. 26 lines.
- `markgiant.sp` 75-85: `ForgetAllKnownEntities()` then `AddKnownEntity(target)` to
  force aim, carrying the author's own `//TODO: aim directly on target instead of
  doing this dumb shit`. Wiping a bot's entire vision memory to steer one shot will
  fight every other action's threat selection. 11 lines.
- `debug_faults.sp` 47-49 and 253-262, `redbots_debug_old_wedge_recovery`:
  pre-2.21.3 behaviour kept alive as a switch for one A/B that is finished. 10
  lines. Separately, line 61 arms a 0.1 s repeating timer unconditionally at init
  and the callback then checks the convar, so it runs ten times a second on every
  server forever to do nothing.
- `nextbot_behavior.sp` 1458-1461: `bool lurkingNowhere = false;` assigned once,
  never reassigned, and OR-ed into `wantsToBeElsewhere`. A dead term left from the
  earlier sniper attempt. 4 lines.
- `nextbot_behavior.sp` 2352-2370: two consecutive comment blocks on
  `ShouldTakeUpPosition`, the second contradicting the first about whether the
  medic is excluded. One of them is stale and it is not obvious which. 19 lines.
- `medic_uber.sp` 133: an unconditional `LogMessage` on every shield deploy, which
  writes to a live server's log for the length of every wave. 3 lines.
- `blu_assist.sp`: 28 lines of header describing three levers when only one exists,
  `BluAssistBendAttrib` carrying an unused `scale` parameter for the removed speed
  bend, and 102-117 explaining a multiplication no caller performs. ~14 lines of
  drift, and the drift is worse than the code because it describes a feature that
  is not there.
- `dhooks.sp` 231, `if (g_bSpyKilled) g_bSpyKilled = false;`, and the
  `LoadUpgradesFile` detour at 17-23 that its own comment flags as a fallback for
  an address nobody could find. 3 lines. Guess on the second one: it may still be
  load-bearing on some game version.

**Delete by consolidation, ~1,000 lines.** These are not dead. They are the same
code written many times, and porting each copy is the mistake.

- `util.sp` 1183-1690: nine variations of "loop clients, filter, keep the min or
  max distance" (`FindEnemyNearestToMe`, `GetBestTargetForSpy`,
  `GetNearestSappableObject`, `GetNearestEnemyTeleporter`, `GetNearestEnemyCount`,
  `GetNearestSappablePlayer`, `GetFarthestSappablePlayer`,
  `GetEnemyPlayerNearestToPosition`, `GetNearestSappablePlayerHealingSomeone`).
  ~500 lines that collapse to one parameterised scan with a pure predicate. This is
  also where `mvm-ds3` lives: every one of them loops player slots, and `tank_boss`
  holds none.
- `menu.sp` 153-502: 350 lines of if-tree over arrays that already exist.
- `loadouts.sp` 258-315 and 399-491: ~150 lines of two-level string dispatch over
  the same table, collapsing to an index.
- `util.sp` 2547-2911: `PickBuildArea`, `PickBuildAreaPreRound` and
  `ShouldRelocateNest` all end in the same five-tier `BestNestArea` cascade at
  2677-2681 and 2835-2838. One tier-list helper replaces three copies.
- The `m_ct*Ask` plus `m_b*Possible` memo cache, written three times in
  `getammo.sp`, `gethealth.sp` and `collectmoney.sp`. One shared helper.
- `ShouldAttack`'s spy revolver logic, byte-identical in `getammo.sp` and
  `gethealth.sp`. `IsValidAmmo` and `IsValidHealth` differ only by classname list.
- The build-ring machinery duplicated between `engineerbuildsentrygun.sp` and
  `engineerbuilddisposable.sp`, with the disposable copy missing the press-settle
  fix. Guess: the disposable action has the same latent double-build as `mvm-oh7`.

**Not supported by measurement.** One reader flagged the trailing `stock` block in
`util.sp` 2912-3286 as ~120 dead lines. A grep of the seven named candidates
(`DoesAnyPlayerUseThisName`, `ReadInt`, `DereferencePointer`,
`TEMP_GetPlayerMaxHealth`, `IsServerFull`, `VMX_VectorNormalize`,
`NudgeTowardsGoal`) shows every one has at least one caller besides its definition.
The guess is refuted for those seven. `stock` does suppress the unused-symbol
warning, so something in there may be dead, but nothing here shows it and the 120
lines should not be counted until a per-symbol pass says so.

## And better

Parity is the floor. These are the things the generated plugin can do that the
current one cannot, each tied to something already in the record rather than to
ambition.

**1. Exhaustiveness over class times phase, and the deletion it unlocks.** Proven.
`internal/actionsel` walks 1,425,408 reachable combinations, names each hole as its
exact combination, and `TestExhaustivenessCatchesAPunchedHole` proves the mechanism
cannot rot into a test that passes for the wrong reason. It found `mvm-vnn`, a
Scout with nothing to collect and nothing to shoot, without playing a wave. Four
closed bugs (`mvm-7kr`, `mvm-pvt`, `mvm-e4g`, `mvm-489`) are the same shape. The
plugin cannot have this property at any price: a `switch` with a missing case is
not an error in SourcePawn, and the only way to find the hole is for a player to
watch a bot stand still. The consequence is the 670 lines of watchdog above, which
exist only because the hole could not be proved absent.

**2. Geometry checked offline, across every map at once.** Proven.
`internal/navmesh` parses seven MvM meshes to the last byte, 7,628 areas, zero
unresolved connections, and refuses a file it does not finish. It found three bugs
nobody had reported: `mvm-1yo`, where the teleporter exit ring at
`engineerbuildteleporter.sp:431` has no drop check and 32 percent of Decoy's ring
sits beside a fall with 28 of those lethal; `mvm-0dn`, where Rottenburg SniperSpot
4 is 213 units from any nav surface and has been shipping as a spot that silently
does nothing; and `mvm-dx2`, four declared spots in a ground-level hole. The plugin
cannot do this because it can only see the map it is running, one at a time, at
runtime, while a wave is playing.

**3. Cross-map invariants at build time.** `mvm-tz9` shipped Decoy with zero
sniper spots for several releases because a test comment was left in a config file,
and `Config_LoadMap` was already printing the count nobody read. `mvm-1oj` says 22
config files have never been walked. 27 configs exist and all 27 parse in Go today.
A build that fails on a count of zero, or on a `TODO` marker, or on a spot with no
ground under it, is a property the plugin has no place to express: it has no build
step that can see more than one map.

**4. A refactor that can be proved behaviour-preserving.** Proven by `mvm-z83.1`: a
pure function compiles and runs under spshell with no game server, no
`sourcemod.inc` and no natives, and its float32 result matches the Go bit for bit,
compared as raw bits rather than through `printfloat`. The test was proved able to
fail. Today no change to a decision function in this plugin can be shown safe by
anything except playing a wave, and `mvm-bj8` is what that costs: four fixes
written, four refuted on the reporter's server, one of them dropping sniper damage
from 16,430 to 479.

**5. A fact cannot be written twice.** Proven twice over: 22 features and 112
telemetry fields round-trip byte for byte, and `TestFeatureProofCatchesASwappedName`
proves the proof. This inventory measures the remaining surface at 3,287 TABLE
lines, and it names three places nobody had listed: the 416 declaration lines in
`sdkcalls.sp`, `dhooks.sp` and `offsets.sp` that restate their gamedata key inside
an error string, the 350 lines of `menu.sp` restating `loadouts.sp`, and the action
name typed in `ActionsManager.Create` and again as a string literal in
`GetCountOfBotsWithNamedAction` (`campbomb.sp:143`, `destroyteleporter.sp:90`).

**6. A question answered in a second instead of six waves.** `mvm-gwq` spent 38
shopping trips over six Decoy waves to establish that the resistance price was 210
and not 25. That is arithmetic over money, wave number and damage type. `mvm-ckw`
is a sentry at full health with an empty magazine still answering yes to
`SentryNeedsMetal`: health, magazine and metal, three integers. `mvm-1pq` is
`TELEPORTER_EXIT_RADIUS` 150 against `BUSTER_BLAST_RANGE` 400 in another file, which
one assertion in Go catches and no compiler was ever going to.

**7. Refusal as a mechanism.** `gosubset` refuses a construct it does not support,
with a line number, rather than emitting something plausible. The nav parser refuses
a file it does not finish reading, which is what catches a layout misread that stays
in bounds. Both are "fail loudly rather than limp on", and the plugin has no
equivalent: `mvm-bj8` is `TF2Econ_GetItemClassName` failing quietly for a stock
primary, `mvm-0l5` is `HasAmmo` returning true for a meter weapon, `mvm-ijg` is
`TF2Util_GetPlayerObjectCount` throwing and taking the rest of `OnEnd` with it.
`mvm-z83.30`, native bindings carrying preconditions, is where that lands.

**8. The plugin can be measured about itself.** Nothing in the current toolchain
can say that 187 of 756 functions hold 41 percent of the line count, or that 320
functions are already pure, or that three files are 806 lines of which 416 are
mechanical. Those numbers are what makes an estimate arguable instead of asserted,
and they are the reason this document can say the hand-written floor is nine times
the epic's figure rather than agreeing with it.

## What parity does not buy

The closed record scores 31 defects: prevented 0, caught earlier 10, does nothing
21. Of the 14 open P1s outside this epic, the design as drafted helps 2, and only
to measure. Everything in the DELETE and TABLE columns above is real and safe, and
none of it makes a bot walk out of a corner on Mannworks.

Three of the ten caught cases are the noise floor and the golden harness rather
than the generator. Those are the cheapest parts of this epic and they carry a
third of its measured value. The ordering above puts them first for that reason,
not for a reason about generators.
