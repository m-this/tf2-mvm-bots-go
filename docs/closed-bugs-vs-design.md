# The closed bugs against the Go design

Read the closed record first and the verdict is uncomfortable. Of the 35 closed
non-design beads in `../tf2-mvm-bots`, 14 of the 31 that are real defects or
investigations are category (c) engine, entity, nav and geometry, or category (d)
action lifecycle, and the design does nothing for any of them. It says so itself,
in "What this does not fix", and the record says that sentence covers the largest
half of the work. Not one closed bug is the `features.sp` bug the design opens
with. That bug is real and mvm-z83.2 has already made it impossible, but it
happened once, before the record starts, and nothing since has looked like it.
The design is aimed at the failure the maintainer can name, and the record is
full of failures he cannot: a bot standing still with nothing on his action
stack, a native that answers wrongly instead of failing, and a measurement taken
on a machine that was paging.

That is the finding. The rest of this file is the evidence and what to add.

## Every closed bead

### mvm-bj8, sniper bots stand in spawn on the stock loadout, P0

`SetMission` sat behind `TF2Econ_GetItemClassName`, which cannot succeed for a
stock primary because `TF_ITEMDEF_DEFAULT` is not an item definition, so a stock
sniper never got his lurk. `source/redbots3/loadouts.sp`.
Category (c), the semantics of an engine call. DOES NOTHING. Worth naming the
tail: four fixes were written and each was refuted on the reporter's server,
including one that dropped sniper damage from 16430 to 479. None of the four went
near the missing call, and no Go unit test of a pure function would have either.

### mvm-2uj, replaced bots never shop, P0

There was no bug. The reporter had purchase-in-chat off, so the buys happened
and nothing printed them. Category (e), measurement. DOES NOTHING. The whole
P0 was built on an absent log line.

### mvm-hnb, the engineer freezes and never builds, P0

A bot wedged on map geometry, fixed by moving him to the nearest walkable nav
area after three stucks, in the watchdog. Mannworks went from 0 waves and no
sentries to 4 of 4 and 42 sentries. The close reason says the real cause,
whatever keeps walking him into that spot, is still unknown. Category (c).
DOES NOTHING.

### mvm-tnt, the server dies in CBaseNPC's serial refresher, P0

Not CBaseNPC. `testbed/build.sh` gave staged prebuilt extensions a new mtime
every build, so the testbed copied a live mapped `.so` over itself. Fixed with
`cp -p`. Category (e), the test bed corrupting what it measures. DOES NOTHING:
the design's reproducibility work, mvm-z83.12, is about the spshell toolchain,
not about how the plugin is staged into a running server.

### mvm-gwq, bots do not buy resistance upgrades, P0

Premise disproved. 38 shopping trips over six Decoy waves: every trip saw a
damage type, so resistances were priced at 210 and not 25, and the wave read was
right including on wave 1. Category (e) with a (b) core, since the pricing is
arithmetic. CAUGHT EARLIER, weakly and honestly: the price-from-wave-and-damage
function is exactly the shape mvm-z83.6 and mvm-z83.5 target, and a golden input
table would have answered "does the list price resistances correctly" in a second
instead of six waves.

### mvm-cf3, the wave start stutters, P0

Closed by the wedge recovery: worst frame 1833.7 ms to 141.7 ms on Mannhattan.
Category (c). DOES NOTHING.

### mvm-7kr, engineers freeze in spawn between waves, P0

`g_bShoppedThisBreak` was cleared on `mvm_wave_begin`, which ends a break rather
than starts one, so a bot that survived a wave carried "has shopped" through the
next break and `GetDesiredBotAction` had nothing to offer an engineer or a medic.
`source/redbots3/events.sp` and `nextbot_behavior.sp`. Category (d).
DOES NOTHING as the design stands, and this is the single most important miss in
the file. See the additions below.

### mvm-489, a sniper sent to the front strafes without firing, P1

The round-running branch of `GetDesiredBotAction` answers a rifle sniper with
`Plugin_Continue`, on the assumption that `Timer_PlayerSpawn` already gave him
his sniping behaviour. It had not, so he had nothing for the whole wave.
Category (d). DOES NOTHING.

### mvm-ijg, the sentry action throws when the bot has already left, P1

`TF2Util_GetPlayerObjectCount` throws for a client not in game, and a thrown
native takes the callback with it, so the rest of `OnEnd` never ran.
`source/redbots3/util.sp`. Category (c). DOES NOTHING today, though see the
binding annotation proposal.

### mvm-bha, scale BLU down when few humans play, P1

Three levers, one kept. The first measurement said health and speed did nothing
because both were written inside the spawn frame where the popfile overwrites
them, and health also needs the max health additive bonus. Category (c), engine
write ordering, with an (e) tail. DOES NOTHING.

### mvm-74t, a shopping bot never mirrors the player's ready, P1

`CTFBotTacticalMonitor_Update` returned early for a bot in the upgrade zone, and
that return sat above the readiness call, so the comment claiming it ran "every
frame, wherever it came from" was false. `nextbot_behavior.sp:762`.
Category (d). DOES NOTHING. Note that the invariant was written down, in prose,
directly above the code that broke it.

### mvm-tz9, Decoy ships with no sniper spots, P1

Six `SniperSpot` origins were commented out for a test in 5aedaff and shipped in
v2.15.0, so every nightly ran Decoy with zero sniper spots.
`configs/defenderbots/map/mvm_decoy.cfg`. Category (a), two facts drifting: the
config and what every run assumed the config held. DOES NOTHING, because the
design's tables stop at features and telemetry and never reach the map configs.
The bead itself says the line that would have caught it was already being
printed by `Config_LoadMap` and nobody read it.

### mvm-3it, bots spy-check the human player, P1

The sighting rule in `Event_PlayerDeath` was "a Spy killed somebody on another
team", which a defending human Spy satisfies every time he stabs a robot, and
the suspect finder filtered no humans. `source/redbots3/behavior/spycheck.sp`.
Category (b): both halves are pure predicates over team, class and fake-client
status. CAUGHT EARLIER. `IsSpySighting(killerTeam, killerClass, victimTeam)`
takes plain values and returns a bool, it is inside the mvm-z83.4 subset, and a
golden table with a friendly Spy row fails on the shipped rule.

### mvm-6dc and mvm-e5r, mid-mission class changes and custom loadouts, P1

Features, not defects. Not counted in the verdict tally.

### mvm-ckw, Rescue Ranger engineers never refill the sentry, P1

A sentry at full health with an empty magazine still answered yes to
`SentryNeedsMetal`, so `CanRepairFromRange` let the engineer fire bolts into
something already whole, forever. Category (b), a pure decision computing the
wrong answer. CAUGHT EARLIER, and this is the clearest hit in the file: the
inputs are health, magazine and metal, all plain numbers, and the golden input
harness of mvm-z83.14 with a full-health-empty-magazine row fails immediately.

### mvm-8ws, engineer disposable mini-sentry, P1

An A/B that came out clean: deaths per wave off [0,3,5,7,9,10] against on
[11,12,14,15,15,17]. The close reason admits the arms ran under different
machine load. Category (e). CAUGHT EARLIER by mvm-z83.8: a noise floor is
exactly what turns "the spreads do not touch" from an eyeball into a result.

### mvm-1b7, the engineer carries a building forever, P1

`CTFBotMvMEngineerIdle_Update` only places within 70 units of the nest centre and
there was no clock, so an unreachable centre left him holding it. Fixed with
`CARRY_GIVE_UP_TIME` at 25 seconds, from reading the code, never reproduced.
Category (d). DOES NOTHING.

### mvm-0l5, the Gas Passer crashed the server, P1

Two causes. `HasAmmo` is always true for a weapon firing off a charge meter, so
the Pyro equipped a jar he could not throw every tick. Then the residual
watchdog crashes were nav mesh searches with no bound, reached from the
per-frame path refresh, capped by `PATHS_PER_FRAME` at two a frame; 41 runs and
231 waves since with no crash. Category (c) twice over. DOES NOTHING. The bead
names the pattern that produced four separate crashes: a nav mesh call inside
something that reads like a cheap predicate.

### mvm-ibb, the test-bed needs a machine with memory free, P1

Around 200 MB free and 20 MB/s of swap-in made the watchdog fire, and a crash
rate was read as a property of the code. `testbed/run.sh` now refuses below
1500 MB. Category (e). DOES NOTHING: a statistical noise floor does not detect
paging, it absorbs it.

### mvm-ph0, none of it is play-tested, P1

Meta. Not counted.

### mvm-297, nest relocation trips the watchdog, P1

A symbolised core put the killed frame in `NavAreaBuildPath` with
`maxPathLength=0`, reached from the mod's own per-frame refresh inside
`PlayerRunCmd`. Fixed by the path budget. Category (c). DOES NOTHING.

### mvm-smt, one ask adds twice, P1

`AddBotsFromTeamComposition` returned a bool that is true only when it filled all
N, so a partially named team fell through and the lineup mode was asked for N
again on top. RED came up 10 instead of 7. Category (b), arithmetic over counts.
CAUGHT EARLIER: `remaining(asked, filled)` is one line of pure Go and a table
test with a partial fill fails on the bool version.

### mvm-bng, bot seating for specific classes, P1

`IsBotClassBlacklisted` returned false for any class named by any composition,
which is right for a lineup somebody typed and wrong for a default the mod
guessed at. Category (b), a pure predicate over blacklist, composition and
where the composition came from. CAUGHT EARLIER, same mechanism as mvm-3it.

### mvm-oh7, one engineer builds two dispensers, P1

A one-frame race: `VS_PressFireButton` puts the building down on the next tick
and `GetObjectOfType` was asked in the same frame, so the answer was always none
and the toolbox re-armed. `behavior/engineerbuilddispenser.sp`. 18 duplicate
events to 0. Category (d). DOES NOTHING; a generated pure function cannot see a
tick boundary. Note also the (e) half: the first instrument, the game's own
object list, could not show the thing being looked for.

### mvm-pvt, sniper bots on the stock loadout stand in spawn, P1

The same stale shopping flag as mvm-7kr, because a rifle sniper is one of the
classes `ShouldTakeUpPosition` refuses. Category (d). DOES NOTHING. This is the
other reverted fix: a fallback letting an unhinted sniper fight was written and
measured and thrown away, 4521 damage lurking against 570 attacking. A sniper
had never been in a testbed lineup, 3417 engineer samples and not one sniper,
which is why it survived.

### mvm-e4g, the medic had nothing to do between waves, P1

`ShouldTakeUpPosition` refused four classes on the reasoning that each has
somewhere else to be, and that is false for a medic between rounds, who heals
nobody. Category (b) in shape, since it is a pure class predicate, but the fault
was a wrong belief about the game and not a wrong number. DOES NOTHING: a Go
unit test written by the same author encodes the same belief and passes.

### mvm-427, the test-bed calls an install-time restart a crash, P2

Filed on a wrong inference and closed as such: the install line prints every
thirty seconds, so it separated nothing, and `run.sh` was already right.
Category (e). DOES NOTHING.

### mvm-s99, a taunting bot holds the wave, P2

`FakeClientCommand(client, "tournament_player_readystate")` is held by a taunt,
and a looping taunt never ends, so the flag never flips. Category (c).
DOES NOTHING. Still not reproduced against a looping taunt.

### mvm-1pq, the teleporter exit is built inside the blast that kills the sentry, P2

`TELEPORTER_EXIT_RADIUS` is 150 in `behavior/engineerbuildteleporter.sp` and
`BUSTER_BLAST_RANGE` is 400 in `util.sp`, so the exit was always well inside the
radius the same mod uses to make bots run away. The narrower root cause was that
Decoy names one `TeleporterExit`, `NearestFreeExitSpot` hands it to the first
engineer, and the second fell to the nest-anchored fallback. Category (a) and
(b) together. CAUGHT EARLIER: spot choice is on the design's own list of pure
decisions, and a Go test asserting the chosen exit is beyond
`BUSTER_BLAST_RANGE` of the sentry fails on the fallback path.

### mvm-y9d, engineers do not replace a destroyed dispenser, P2

Mostly not a defect: median 20 seconds without a dispenser over 1441 stretches.
The tail is real and the cause is that the dispenser branch only fires when
`m_ctSentrySafe` is in the future. Category (b). CAUGHT EARLIER in principle,
since that gate is a pure predicate, though only if somebody asks the question.

### mvm-654, a refused purchase ended the whole shopping trip, P2

Ten of 45 trips ended on a refusal, one walking away with 700 credits. The
refusal is remembered per trip now. Category (b), and it is upgrade-order code,
which the design names. CAUGHT EARLIER.

### mvm-a8g, nothing deploys the Medic's Projectile Shield, P2

Built and measured, and two of three arms sit inside the baseline band, so it
ships off. Category (e). CAUGHT EARLIER by the noise floor. The bead also
contains the best instrumentation idea in the whole record: the shield press is
logged 26 to 35 times a run, "so a null result cannot be mistaken for a behaviour
that never ran".

### mvm-1oj, map data, P2

A survey task. Not counted. Twenty-two config files are still unwalked.

### mvm-b3j, the test-bed says a crash happened but not what kind, P3

`run.sh` counted crashes without distinguishing a watchdog kill from a segfault,
and the two lead opposite ways. Category (e). DOES NOTHING.

### The five closed design beads

mvm-z83.1 proved spshell runs a pure generated function and matches Go bit for
bit. mvm-z83.2 made the `features.sp` bug impossible and proved the proof by
swapping two names. mvm-z83.3 did the same for 112 telemetry fields. mvm-z83.4
is the refusing subset checker, 66 tests, and it was validated against real code:
`ThreatPriority`, `SelectMoreDangerousThreat` and `ScoreNestArea` all fit.
mvm-z83.10 generates bindings from the includes and found a genuine trap, that
SourcePawn passes arrays by reference so a non-const `float vec[3]` must emit as
a pointer. This work is sound. The question is not whether it works, it is
whether it is pointed at the bugs.

## The aggregate

Categories, over the 35 closed non-design beads:

- (a) two facts written twice and drifting: 2 (mvm-tz9, mvm-1pq)
- (b) a pure decision computing the wrong number: 7 (mvm-ckw, mvm-smt, mvm-bng,
  mvm-654, mvm-e4g, mvm-y9d, mvm-3it)
- (c) engine, entity, nav, geometry: 8 (mvm-bj8, mvm-hnb, mvm-cf3, mvm-ijg,
  mvm-bha, mvm-0l5, mvm-297, mvm-s99)
- (d) action lifecycle or state machine: 6 (mvm-7kr, mvm-489, mvm-74t, mvm-1b7,
  mvm-oh7, mvm-pvt)
- (e) measurement or test-bed error: 8 (mvm-2uj, mvm-tnt, mvm-ibb, mvm-427,
  mvm-b3j, mvm-gwq, mvm-8ws, mvm-a8g)
- (f) features and survey tasks, no defect: 4 (mvm-6dc, mvm-e5r, mvm-ph0,
  mvm-1oj)

Verdict, over the 31 that are defects or investigations:

- PREVENTED: 0
- CAUGHT EARLIER: 10 (mvm-ckw, mvm-smt, mvm-bng, mvm-654, mvm-1pq, mvm-y9d,
  mvm-3it, mvm-gwq, mvm-8ws, mvm-a8g)
- DOES NOTHING: 21

Prevented is zero on purpose. The design makes a class of bug impossible, the
`features.sp` name drift, and that class has not recurred in the record. Every
other bug is a wrong belief about the game or a wrong sequence in time, and
writing the same wrong belief in Go does not fix it. Ten of the caught cases
depend on somebody thinking to write the failing row, which is a real gain over
playing a wave to find out, but it is a gain in speed, not in certainty.

Three of the ten caught cases are the noise floor and the golden-input harness
rather than the transpiler: mvm-8ws, mvm-a8g and mvm-gwq. The cheapest parts of
the design carry a third of its value.

## What the design should add

### 1. Action selection is a pure decision, and it is the one the design left out

The design lists threat score, upgrade order, nest spot and patient choice.
The record says the function that broke most is none of those. mvm-7kr,
mvm-pvt, mvm-e4g and mvm-489 are all one function answering wrongly:
`GetDesiredBotAction` and `ShouldTakeUpPosition`, deciding which behaviour a bot
gets from his class, the round phase, and a handful of flags. That is plain
values in and a choice out, it is inside the mvm-z83.4 subset, and it produced
two P0s and three P1s in a week.

Move it, and add the property the unit tests missed: exhaustiveness. Enumerate
class times phase times flags and assert that no input maps to "nothing". The
shipped code had four holes, an engineer in a break with the shopped flag set, a
medic in a break, a rifle sniper in a break, and a rifle sniper in a wave with no
mission, and every one of them shipped as a bot standing still. A total table is
mechanical and would have found all four before Peppy did.

### 2. A behaviour-coverage gate in the testbed, generated from the same table

mvm-7kr already built the instrument: a between-waves table giving the share of
samples with no Defender action anywhere on the stack, 86% and 100% before, 0%
after. It exists, it is read by hand, and nothing fails a run on it. Make it a
gate driven by the same Go table as item 1: for each seat, the allowed share of
empty stacks per phase, and a run that exceeds it fails. mvm-489, mvm-hnb and
mvm-pvt all show as a stack that is empty or frozen, so a gate covers part of
category (c) as well, which the design says it cannot touch.

Add the missing lineup coverage while doing it. mvm-pvt survived because a
sniper had never been in a testbed lineup across 3417 samples. The lineup matrix
belongs in a Go table next to the feature table, with a test asserting every
class the mod can seat has been played.

### 3. Cross-file constants belong in the same table treatment as features

mvm-1pq is `TELEPORTER_EXIT_RADIUS` 150 in one file against `BUSTER_BLAST_RANGE`
400 in another, and no compiler was ever going to notice. `CARRY_GIVE_UP_TIME`
from mvm-1b7, `PATHS_PER_FRAME` from mvm-297 and `STUCK_RADIUS` from mvm-bj8 are
the same shape: tuning numbers that other numbers depend on. One Go table
emitting the defines, with the relations asserted in Go, is the mvm-z83.2
generator applied to a second fact family and costs almost nothing.

### 4. Map configs are a table too, and nobody is checking them

mvm-tz9 shipped Decoy with zero sniper spots for several releases because a test
comment was left in a config file. mvm-1pq's real cause was that Decoy declares
one `TeleporterExit` and there are two engineers. mvm-1oj says twenty-two config
files have never been walked at all. `Config_LoadMap` already prints the counts.
Declare the expected per-map block counts in Go, generate the check, and fail
the build on a `TEMPORARILY` or `TODO` marker or on a count of zero where the map
is supposed to have spots. This is the largest gap between what the design calls
a table and what the record calls a drifting fact.

### 5. Bindings should carry the semantics, not just the signature

mvm-z83.10 generates 1175 native declarations and it already found one semantic
trap on its own, the by-reference array. Three closed bugs are nothing but
native semantics: `TF2Econ_GetItemClassName` fails for `TF_ITEMDEF_DEFAULT`
(mvm-bj8, P0), `HasAmmo` is always true for a meter weapon (mvm-0l5, P1),
`TF2Util_GetPlayerObjectCount` throws for a client not in game (mvm-ijg, P1).
Add an annotation file beside the generated bindings: per-native preconditions
and a cost marker, then lint the plugin's call sites against it. mvm-0l5 names
the rule to encode first, that `IsPathToVectorPossible` is a full
`NavAreaBuildPath` and reads like a cheap predicate. That single annotation
covers a pattern the bead says produced four separate crashes.

### 6. The noise floor has to include the machine, and the arms have to interleave

mvm-z83.8 as drafted is a statistical floor over wave outcomes. The record says
the variance that ruined measurements was not statistical. mvm-ibb is paging,
mvm-tnt is `build.sh` copying a live mapped `.so` because of an mtime, mvm-8ws
says the arms ran under different machine load and mvm-a8g says one arm shared
the machine with another server. Record `MemAvailable`, load and the extension
checksums per run, refuse a comparison whose arms differ, and interleave the
arms rather than running one after the other. Without that the floor measures
the machine.

### 7. Every feature reports whether it fired

mvm-2uj was a P0 built entirely on an absent chat line from a setting that is off
by default. mvm-tz9 was a zero that was printed and never read. mvm-a8g solved
it the right way by logging the shield press 26 to 35 times a run so a null
result could not be mistaken for a behaviour that never ran. Generate that from
the feature table in mvm-z83.2: one fired-counter per feature, emitted into the
wave record from mvm-z83.3, and a testbed rule that an A/B whose armed feature
fired zero times is not a result. This is a small addition to work already done
and it retires the class of bug the design's own opening paragraph describes,
an A/B that armed the wrong feature and produced a number that was read as a
measurement.

### 8. Write the disproving measurement before the fix

mvm-bj8 took four fixes, all refuted by the reporter, one of which cost 16430
damage against 479. mvm-pvt's fix was written, measured and thrown away.
mvm-bha shipped three levers and removed two in v2.23.0. mvm-gwq and mvm-427
were both filed on premises that measurement destroyed. That is the regression
churn the maintainer is complaining about, and it is not a language problem.
The design's contribution to it is the golden-input harness, mvm-z83.5 and
mvm-z83.14, used before a fix rather than after: state the input that is
supposed to be broken, show it produces the wrong output, then change the code.
mvm-bj8's four attempts never once produced an input where `SetMission` was
missing, because nobody wrote one down.
