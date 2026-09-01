# What the testbed measures

`go run ./testbed/cmd/testbed` plays a mission with nobody in it and writes one JSON object
per line to `testbed/results/`. `go run ./testbed/report <file>` turns that into
prose; a second file argument compares two runs.

Facts go in the file, verdicts go in the report. Changing your mind about what
counts as a useless dispenser should cost a recompile, not another run.

## The lines

| `event` | One per | What it answers |
|---|---|---|
| `wave_end` | wave | did the team hold, who did the damage, what killed them |
| `wave_begin` | wave | what the between-rounds time bought |
| `engineer` | engineer, twice a wave | what he had standing and where |
| `perf` | wave | frame times, and the worst frame with a timestamp |
| `stall` | frame over 250 ms | when the server hitched |
| `bot` | bot, every 5 s | where he was and what he was doing |
| `building` | building, every 5 s | where it was and whether it was worth its metal |

`bot` and `building` sample in **both** round states. Half of what has gone
wrong went wrong between waves — the walk to the front, the shopping trip, the
toolbox still set to the last building — so sampling only during a wave samples
the half that was never the problem.

## Reading the bot samples

```
what the bots were doing (312 samples)
  Wesley       medic     DefenderMedicHeal 71%, DefenderGotoUpgrade 18%, none 11%
                         beam on somebody 44% of the time
                         secondary 88%/primary 12%, hurt 9% of the time
```

- **the action share** is where the time went. A bot stuck in a house shows as
  one action at 90% with nothing to show for it.
- **beam on somebody** is the medic's actual output. Following a patient and
  healing him are different things and this is the only number that separates
  them.
- **the weapon slot share** answers "is the Heavy using his minigun or his
  shotgun" without anyone watching him.

## Reading the building samples

```
  Bob sentry         level 3, saw a robot 61% of samples (1.8 at a time), 1.2 teammates in range
  Bob dispenser      level 3, saw a robot  4% of samples (0.1 at a time), 2.4 teammates in range
  Bob mini sentry    level 1, saw a robot  8% of samples (0.2 at a time), 0.1 teammates in range
```

- **saw a robot** is a sentry's worth. It traces to every live robot in range,
  so it is line of sight and not just distance. A sentry that never saw one is
  in the wrong place however healthy it is.
- **teammates in range** is a dispenser's worth. That is the question the
  hand-walked `DispenserSpot` entries in `configs/defenderbots/map/` exist to
  answer, and nothing checked it before.
- **two rows for one owner and one building type** means a duplicate, which is
  a real bug that happened and survived four waves.

## Where they stood, and how close the fight was

```
how close the nearest robot was
  heavy     median 745, inside his own blast radius 4% of samples
  medic     median 1896, inside his own blast radius 2% of samples

where they stood
  Wesley     medic     stood still 79% of the time, longest 105s at 707 -2559 512, 7396 units walked per wave
  Bob        engineer  stood still 41% of the time, longest 85s at -179 889 416, 33687 units walked per wave
                       wrench out 13 samples, 15% of them out of reach of his own buildings
```

Both are computed in the report from `at` and `nearest_enemy`, which the plugin
has always written and nothing read. No extra sampling cost.

- **median nearest robot** is the front line, as a number. "The soldier is too
  close" and "the medic is nowhere near the fight" are both claims about this
  and were argued about for two sessions without it. A class whose median is
  double everybody else's is not in the fight.
- **inside his own blast radius** is 146 units, a rocket's own splash. This is
  the mechanism behind the self-damage column, one step upstream.
- **stood still** is the share of five-second gaps in which the bot moved under
  100 units. Every class fighting normally lands between 15% and 45%. Anything
  near 80% is a bot that has stopped, and `longest` and the coordinates say
  where to go and look.
- **units walked per wave** is the same fact without a threshold in it, and it
  separates faster than the share does: 30000 or more is a bot working the map,
  and the parked medic was doing 4866.
- **wrench out of reach** is an engineer holding a wrench more than 100 units
  from anything he owns. He cannot repair from there. It does not prove he is
  swinging, only that swinging would achieve nothing.

Positions are worth printing rather than summarising. Two of these went straight
to a named place on the map that somebody could walk to.

## Between waves

Every other table here filters on `t > 0`, which is a wave running. The break is
the other half of a mission: the shopping, the nest, the teleporter, the walk to
the front.

The number is the share of break samples with no `Defender` action anywhere on
the action stack. `MainAction < TacticalMonitor < ScenarioMonitor` is a bot with
nothing of this mod's on it, and the game has no answer for a defender between
waves, so he stands where the wave left him.

It found the freeze reported as "the engi bots just freeze in spawn": the
engineers were at 86% and 100% of break samples with no behaviour, and the walk
figure beside it said 777 units for a whole break.

Read it with `TESTBED_BOT_MANAGER_MODE=2`. That is AUTO_BOTS, which is what
tf2-archipelago runs, and the freeze does not show in READY_BOTS.

## How much one wave varies from the next

Every total in the report is a sum over waves, and a sum hides its spread. The
report prints the per-wave median with its quartiles and range beside it, and the
rule is simple: **a difference between two arms that does not clear the quartiles
of either is a story, not a result.**

This was learned the expensive way. A lineup change was read as a forty four per
cent longer hold, eight waves against eight, and the same change built a second
way came back at the baseline. The arm that looked like a discovery had waves of
88 and 282 seconds in it:

```
    held for         median   255   quartiles 178 to 274   range 88 to 282
```

Two arms of eight can see a wave lost sixteen times out of sixteen. They cannot
see a change in how long it is held.

### The band for Bavarian Botbash wave 1

Twenty four attempts on the unchanged build, `mvm_rottenburg_advanced1` wave 1,
six bots and the host, AUTO_BOTS. None of them cleared it.

| | median | quartiles | range |
| --- | --- | --- | --- |
| held for | 160s | 154 to 192 | 84 to 286 |
| defenders died | 24 | 22 to 31 | 14 to 47 |
| robots killed | 58 | 52 to 64 | 25 to 98 |

An arm whose median falls inside those quartiles has shown nothing. That is the
band every change to this mission is read against, and building it is what made
the earlier readings on this wave interpretable at all: an arm that was called a
forty four per cent longer hold has a median of 254 seconds, which is outside the
band, and a second arm of the same change built another way came back at 179,
which is inside it. One of the two is wrong and neither is a result.

## Repairs

```
buildings took    3140, engineer put back 1890 (60%)
```

Sampled twice a second, because a sentry that loses two hundred health and gets
it back inside one five-second telemetry interval is invisible at five seconds.
Construction and upgrades are skipped: both raise health for reasons that have
nothing to do with the wrench.

There is no repair event to hook — the game fires nothing when a wrench
connects, and metal spent covers building and upgrading too — so this is health
differences and nothing cleverer. An engineer who never swings and one who
repairs perfectly produce identical uptime and identical `sentries lost`. This
is the column that separates them.

## Count the event, do not sample the state

`CanRepairFromRange` can only run when a sentry is damaged *and* its engineer is
standing two hundred units or more away. A four wave reproduction with a Rescue
Ranger in his hands hit that state **zero times in 137 samples**: the sentry was
below full health twice, and never while he was far enough back. On a test bed a
sentry is healthy or it is dead.

That is not bad luck, it is arithmetic. A five second sampler cannot see a state
that lasts three seconds and happens twice an hour.

So the mod counts the event instead and exports the counter
(`Defenderbots_RangeRepairStalls`). A counter accumulates, and **sampling a
counter loses nothing however rare the thing is** — one stall in an hour still
reads as a one at the end of the run.

```
Bob  engineer  3 times he fired at his sentry for three seconds and it gained nothing
```

Use this shape for anything rare: a stall, a refusal, a retry, a fallback taken.
Sample states that persist; count events that do not.

## Self-damage

`hurt themselves` and `killed themselves` on the wave line, per class. A soldier
firing rockets at a tank he is standing against looks, on any scoreboard, exactly
like a soldier who fought hard and got shot: damage up, kills up, deaths up.
This is the column that tells them apart, and it is comparable between builds.

Printed only when non-zero.

## Cost

Six bots and five buildings at five seconds is roughly 130 lines a minute. The
results directory is gitignored. A `bot` line carries the whole behaviour stack,
so keep `STATS_LINE_LENGTH` ahead of it.

### `t` and `clock`

`t` is seconds into the wave, and it is zero for every sample taken between
waves because there is no wave to be seconds into. Use `clock`, the server's
game time, to tell two samples apart. Reading a file back on `t` alone made one
dispenser sampled fourteen times look like fourteen dispensers, which is a bug
this file exists to find and briefly invented instead.

## Gotchas paid for already

- **Never rebuild while a run measures.** `testbed/build.sh` restages the
  tree, and the container copies the stage into the game tree every thirty
  seconds. For the compiled extensions that truncates a file the server has
  mapped and kills it. For a `.smx` it swaps the build under the run, so the two
  halves of an A/B are no longer the same mod. Wait for the run, or use a second
  checkout.
- **`cp` gives an identical file a new timestamp.** The installer uses `cp -ru`,
  so an unchanged tree is never rewritten. `build.sh` staged the prebuilt
  extensions with a plain `cp`, so every build made them look new. It reads as a
  segfault deep inside whichever extension the installer rewrote. The core says
  `SEGV_MAPERR` at an instruction whose operand carries a relocation: the text
  page reverted to its on-disk bytes. `cp -p` keeps the upstream timestamps,
  which never change.
- **Check the lever reached the server.** A run where a convar silently did not
  apply looks exactly like a lever that does nothing. One rcon query before the
  waves start settles it.
- **`ActionsManager.Iterator` throws on a client that is not a NextBot**, and a
  thrown native takes the whole callback with it. `mvmbots_host` parks an
  ordinary fake client on RED to hold a seat, so the first client of every pass
  killed the pass and the file never appeared. Guarded by `HasBehaviour`.
- **Sample off the frame hook, not a timer.** A timer without
  `TIMER_FLAG_NO_MAPCHANGE` dies with the map, and one created in
  `OnPluginStart` is created once. That produced a results file with a sample
  count of zero for every engineer on every map — a measurement that says
  nothing and looks like a measurement.
- **`docker logs` retains history across container restarts.** One crash in the
  morning read as a crash in every run for the rest of the day until the check
  grew a `--since`.
- **A run measured under paging measures the machine.** A session went from six
  clean waves to four failed runs in a row with no code change: 200 MB free, 5 GB
  swapped, swap-in at 20 MB/s. To a watchdog that measures frame time, a page
  fault is an infinite loop. The runner refuses to start below 1500 MB available
  now; `TESTBED_MIN_FREE_MB=0` overrides it.
- **A popfile the game refuses leaves the map's own mission running.** Three
  sessions of "Bavarian Botbash wave 3" were measured against Rottenburg's
  default intermediate mission. The runner reads `tf_mvm_popfile` back now. The
  real names are in the VPK: `grep -ao 'mvm_<map>[a-z0-9_]*' tf/tf2_misc_dir.vpk`.
- **`File.WriteLine` formats through a 2048 byte buffer.** The wave result
  outgrew it and came out at exactly 2047 characters, which is not JSON, so
  every reader skipped it and a four wave run read as one wave. `WriteString`
  has no such buffer.
- **Do not edit a shell script while it is running.** `sh` resumes at a byte
  offset; the tail of the old file becomes a new command.
