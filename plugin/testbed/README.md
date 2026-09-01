# Test-bed

A Team Fortress 2 server that plays Mann vs Machine with nobody on it, and
writes down what the bots did with every wave.

The mod is judged by play, and play is an opinion until something is counted.
This counts the few things that are not opinions: whether the wave was cleared,
how long it took, how many robots died, how many defenders died, how many of
them died to a knife in the back, and what the engineers lost.

It builds the mod from the working tree, not from a tag. The point is to
measure the change you just made.

The server stack comes from `tf2-archipelago`, which already had a working
Docker build of this mod. What is new here is that nothing is played by a
person and everything is written down.

## Running it

The runner and the reports are Go, and since `mvm-x2c` they live in the sibling
checkout: run them from `../tf2-mvm-bots-go`. This repository keeps what is not
code, which is `build.sh`, the compose files, the popfiles and the map configs,
and the runner reaches all of it from there.

```sh
cd ../tf2-mvm-bots-go

go run ./cmd/testbed -arm plain:                  # two waves of Decoy
go run ./cmd/testbed -mission mvm_decoy_advanced -arm plain:
go run ./cmd/testbed -waves 12 -timeout 60m -arm plain:
go run ./cmd/testbed -maps "mvm_decoy mvm_coaltown" -arm plain:
```

Then compare two runs:

```sh
go run ./report ../tf2-mvm-bots/testbed/results/after.jsonl \
	../tf2-mvm-bots/testbed/results/before.jsonl
```

Needs Docker and Python 3.

The first run needs Team Fortress 2, which is about fourteen gigabytes. On a
machine that already has a server on it, the runner finds
`tf2-archipelago_tf2game` and copies it rather than downloading the game again;
`TESTBED_SEED_FROM=some_volume` names a different one. It is a copy and not a
shared mount on purpose: the test-bed installs its own plugins over `addons/`,
and doing that to the volume a live server is reading ruins the evening for
whoever is playing on it. With nothing to copy from, the game downloads.

The server is left running when the script finishes, because the second run of
the day should not do any of that again. `--down` stops it.

It listens on **27025**, not 27015, so it can share a machine with a server
that is already running. Loopback only: it has no password, no Steam session
and a known rcon password, and it exists to be shouted at by a script.

### Alongside the worklab server

worklab already runs the `tf2-archipelago` stack, deployed by
`ansible-lab/worklab/roles/tf2-archipelago`, and that is where the fourteen
gigabytes come from. The test-bed is deliberately a separate compose project,
on a separate port, with a copy of the game rather than a share of it, so an
apply of that role and a run of this cannot reach each other. Nothing here is
managed by Ansible and nothing here should be: a test-bed that has to be
deployed is a test-bed nobody runs.

The one thing they do share is the machine, so a run competes with whatever the
laptop is doing. Wave durations measured while it compiles something else are
not comparable with wave durations measured while it is idle.

## How a wave starts with nobody playing

This took several wrong answers to get right, so here is the working one.

**A fake client has to hold a seat.** The mod adds its bots in response to a
human pressing F4: its ready listener passes its own bots straight through, and
Mann vs Machine will not begin a wave with nobody ready. An empty server sits in
the pre-round forever. `mvmbots_host.sp` connects one fake client, puts it on
RED, gives it a class and readies it. It has no AI and does nothing else.

**Ready has to be pressed twice.** In `READY_BOTS` mode the first press answers
"Press ready again to start the bots" and does nothing else; the second, within
three seconds, is what spawns them. The mod also rate limits a client to one
command every 0.3 seconds, so the host presses, waits a second, and presses
again.

**Hibernation has to go.** An empty server stops simulating, so no timer runs
and nothing ever adds a bot. The convar is `tf_allow_server_hibernation`, not
the generic `sv_hibernate_when_empty`, which does not exist in Team Fortress 2
and can be set all day without doing anything.

**And one ready player has to be enough**, which is
`tf_mvm_min_players_to_start 1`, with `sm_redbots_manager_min_players -1` to
turn off the mod's own gate, which counts RED before the bots exist.

With all four, the chain runs by itself: host connects, double-readies, the mod
spawns six bots, the bots shop and ready themselves, and the wave begins.

The host is a body in a spawn room and not a seventh bot. The mod counts humans
and its own bots when it decides how many to add, and the host is neither, so
RED ends up with six real bots plus the host. Every `wave_begin` line records
how many of RED were bots, so a results file can always say what it measured.

## What comes out

One JSON object per line, appended as the waves happen. A crashed run still
leaves everything it measured.

```json
{"event":"wave_end","map":"mvm_decoy","wave":3,"result":"cleared","duration":184.2,
 "robot_kills":214,"giant_kills":6,"tank_kills":1,"sentry_kills":63,
 "defender_deaths":9,"backstabs":2,"buster_detonations":1,
 "sentries_lost":2,"dispensers_lost":1}
```

Which of those to read depends on what changed:

| the change             | the number that should move                 |
| ---------------------- | ------------------------------------------- |
| sentry buster reaction | `sentries_lost`, `buster_detonations`       |
| spy checking           | `backstabs`                                 |
| engineer nests         | `sentry_kills` up, `sentries_lost` down     |
| uber deployment        | `defender_deaths`                           |
| stickies, scout jumps  | `robot_kills`, `duration`                   |
| anything at all        | `result` and `duration`                     |

## Every map, and A against B

One map says whether a change works on that map. Most of what an engineer does
is a property of geometry, so it takes all of them to tell a map-shaped bug from
a mod-shaped one.

```sh
go run ./cmd/testbed -maps "..." -waves 6 -arm plain:
go run ./cmd/testbed -maps "..." -waves 4 -tag night -arm plain:
go run ./sweepreport ../tf2-mvm-bots/testbed/results/sweep-night
```

The sweep report adds two tables the per-run report has no way to produce: what
every engineer had standing at the start of each wave and for how much of it,
and what each class did with its seat measured against the waves that class
actually played.

A feature is a named switch (`source/redbots3/features.sp`), which means the
same build can play both sides of an argument:

```sh
go run ./cmd/testbed -maps "mvm_coaltown mvm_decoy" \
  -arm on:sm_redbots_feature_demo_sticky_first=1 -arm off:sm_redbots_feature_demo_sticky_first=0
go run ./sweepreport ../tf2-mvm-bots/results/ab-demo_sticky_first/on \
                             results/ab-demo_sticky_first/off
```

It plays off then on per map, rather than every off run followed by every on
run, so the halves of a pair are minutes apart instead of hours. Every results
file records the features that were on, so a file says which arm it is without
anybody having to remember.

Six waves an arm is a small sample and the bots are not deterministic. A
difference of one cleared wave is noise; only a large move in damage per wave is
worth reading as anything.

## Native Linux, which is where the crashes are

Players report a native Linux server crashing far more often than the same mod
under Docker or on Windows, and the container bed cannot see that: it restarts
srcds by itself, so a crash there reads as a hiccup while the same crash
natively ends the session.

```sh
testbed/seed-native.sh                 # once: copies the game out of the container
testbed/run-native.sh --waves 6        # the native path, still shell
testbed/symbolise-core.sh core.1234    # a backtrace with names in it
```

It runs `srcds_linux` directly rather than through `srcds_run`, so a crash stays
crashed and leaves a core instead of being restarted underneath the measurement.
Cores are enabled by the script, land beside the binary, and are symbolised and
removed at the end of a run unless `--keep-core`.

It writes the same `server.cfg` as the container, by sourcing `entrypoint.sh`
rather than by keeping a second copy: two copies of that file would drift, and a
difference between the two beds is the one thing this is for.

The game is a copy, about fifteen gigabytes, at `~/tf2-native` by default
(`TESTBED_NATIVE_ROOT`). A copy and never a share, for the same reason the
container copies: this installs plugins over `addons/`.

`TESTBED_NATIVE_ROOT` is also how to point this at the install that is actually
crashing rather than at a copy of the container's. tf2-archipelago's launcher
keeps its server at `<install root>/tf-dedicated`, so:

```sh
TESTBED_NATIVE_ROOT=~/path/to/tf2ap/tf-dedicated testbed/run-native.sh --waves 12
```

That runs the mission against the same tree, the same plugins and the same
machine that produces the crash, which is the half of this that cannot be done
from here.

Two things the host may not have. A 32-bit C++ runtime, which `seed-native.sh`
takes from the image into the game's own `bin/` so the tree stays self-contained
rather than needing `libstdc++6:i386` installed. And core dumps: the script
raises the soft limit itself, but `/proc/sys/kernel/core_pattern` has to be a
plain name or a path you can write, not a pipe to a crash handler.

## When the server crashes

The runner notices a server that stops answering rcon and stops
with a message rather than waiting out the timeout, because from outside a
crashing server and a slow one look the same: no new results either way.

The first thing the test-bed ever found was a crash in the branch it was built
to measure. That is what it is for. To chase one:

```sh
docker compose -f testbed/compose.yml logs srcds | grep -iE 'core dumped|Segmentation'
```

For a backtrace rather than a guess, run the server by hand with `-debug`,
which writes `tf/debug.log` inside the game volume:

```sh
docker compose -f testbed/compose.yml run --rm srcds \
  bash -c 'cd $STEAMAPPDIR && ./srcds_run -game tf -console -debug \
           -port 27025 -usercon +maxplayers 32 +map mvm_decoy'
```

## How much to believe it

Not much, from one run.

A wave is not deterministic. The bots draw their loadouts, the mod picks their
classes, and a giant that walks left instead of right decides a wave. Two
seconds of difference in one wave's duration is noise. So is one cleared wave.

What is worth something is the clear rate over a dozen waves, and a number that
moves the same way across several runs of the same mission. Run the baseline
twice before believing the first comparison, and if the two baselines disagree
with each other by as much as the change did, the change has not been measured
yet.

The numbers cannot see everything either. A wave cleared by six bots standing
on the hatch is a cleared wave and so is a wave they held at the choke, and only
one of those is the bots playing well. This says whether a change helped. It
does not say whether the bots look right, and somebody still has to watch them.

## Files

| file                    | what it is                                            |
| ----------------------- | ----------------------------------------------------- |
| `cmd/testbed`           | brings the server up, runs the arms, reads results     |
| `report/`               | turns a results file into a table, and compares two    |
| `sweepreport/`          | reads a whole sweep, or one A/B arm against the other  |
| `checkspots.py`         | which dispenser spot each authored nest would take     |
| `build.sh`              | compiles the mod on the host, into `build/package`     |
| `seed-volume.sh`        | copies an existing game install into the test-bed's    |
| `compose.yml`           | one service, loopback only                             |
| `entrypoint.sh`         | installs into the game volume, writes `server.cfg`     |
| `stats/mvmbots_stats.sp`| the plugin that counts                                 |
| `stats/mvmbots_host.sp` | the fake client that holds a seat and readies up       |
| `loadouts/`             | a loadout to run instead of the shipped one, via `TESTBED_LOADOUT` |
| `rcon.py`               | Source RCON client, from tf2-archipelago               |
| `versions.env`          | every pinned version                                   |

`build/` and `results/` are working directories and are not committed.

## Running an A/B

`go run ./cmd/testbed` replaces the shell for anything being measured.
It builds, restarts the server onto what it built, plays each arm, and prints
the comparison.

```
go run ./cmd/testbed \
  -map mvm_decoy -mission mvm_decoy_advanced \
  -arm on:sm_redbots_feature_watch_idle_bots=1 \
  -arm off:sm_redbots_feature_watch_idle_bots=0 \
  -attempts 3 -waves 2 -tag idle
```

The last `-arm` is the control, and every other arm is compared against it.

It refuses a run rather than producing one nobody should believe:

- **One at a time.** A lock file holds the test-bed and a second runner is told
  who has it. Two shell runs waiting on the same "is a run going" check both
  started, three times in one session, and each looked ordinary afterwards.
- **The loaded plugin must be the built one.** The version in the source is
  compared with the version the server reports. A `--no-build` run once measured
  a two hour old mod and reported the fix doing nothing.
- **The mission must be the one asked for**, and the map is loaded before the
  mission is named. A changelevel resets `tf_mvm_popfile`, and naming a mission
  on a long-running map leaves some of them loaded with none of their robots.
- **RED must hold defenders.** A wave with twenty two robots and nobody to fight
  them still writes a file, and every number in it is zero.

While a wave plays it is watched rather than only waited for, because a run that
has gone wrong looks like a slow one for the first minute and like a finished one
at the end. It stops and names the reason when:

- **no robot has been on BLU for five polls.** The mission is loaded and not
  playing. Nine Mannhunt attempts ended this way and were read as losses.
- **nothing has been written for six polls.** The statistics plugin writes every
  five seconds, so that is the plugin gone, not a quiet wave.
- **RED holds no defenders.** Whatever the rest of the wave measures, it is not
  the lineup asked for.
- **the server stops answering rcon.** That is a crash in what is being
  measured, and it is counted as one.

A precondition that fails stops the whole run. They fail the same way every
time, and grinding through the remaining attempts costs an hour to learn nothing.
