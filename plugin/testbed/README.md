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

The runner and the reports are Go, and they run from the repository root. This
directory keeps what is not code, which is `build.sh`, the compose files, the
popfiles and the map configs, and the runner reaches all of it from there.

The image installs community missions and not Valve's: `-mission` names one of
the popfiles under `scripts/population/`, and leaving it out plays the map's own.

```sh
go run ./cmd/testbed -arm plain:                  # two waves of Decoy
go run ./cmd/testbed -mission mvm_decoy_exp_dissolution -arm plain:
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

### Two beds at once

A second checkout, a worktree for instance, gets a bed of its own rather than
recreating the first one's container out from under it:

```sh
TESTBED_PROJECT=mvmbots-puppet TESTBED_PORT=27137 go run ./cmd/testbed -arm plain:
TESTBED_PORT=27137 go run ./cmd/rc status
```

It does not download the game again. The second container mounts the first
bed's volume read-only and builds its own tree over it: a symlink to every file
of the game, and a real copy of `addons`, `cfg` and `logs`, so the plugin and
the `server.cfg` each bed installs are its own. It runs the server without the
image's steamcmd update, since a linked tree is nothing steamcmd can update; the
first bed keeps updating the game for both. Each bed has its own lock.

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

## Standing in for a player

Several faults need somebody on RED and cannot otherwise be measured here: a
medic that ignores a call, a bot switcher nobody drives, an upgrade station
nobody stands at. A puppet is a body the runner seats and drives, so the fault
gets a run instead of waiting for somebody to be free.

```sh
go run ./cmd/testbed -map mvm_decoy \
  -team scout,soldier,heavyweapons,engineer,medic -defenders 5 \
  -puppets 1 -puppet-calls \
  -arm on:sm_redbots_feature_medic_answers_call=1 \
  -arm off:sm_redbots_feature_medic_answers_call=0
```

A decision that branches per class is not reached by one lineup: the default
holds no sniper, spy or pyro, so an A/B on anything class-specific plays two.
`-teams` takes several, space separated, and files each under its own tag:

```
go run ./cmd/testbed -arm on:... -arm off:... \
  -teams "scout,soldier,demoman,heavyweapons,engineer,medic scout,sniper,spy,pyro,engineer,medic"
```

The lineup has to hold a medic, or there is no beam to measure and the run
answers a question nobody asked.

**A puppet takes a defender's seat.** RED is six and the host already holds one,
so `-defenders` and `-team` come down by one for each puppet. Forget it and the
attempt is refused by name rather than measured short: `checkPuppets` reads the
roster and says how many arrived.

**The mod has to agree it is a player.** `IsTFBotPlayer` is `IsFakeClient`, and
every body a plugin can seat is a fake client, so without help a puppet is one
more bot and `medic_answers_call` is a no-op. `sm_redbots_feature_bot_test_by_nextbot`
answers the question by the nextbot instead: a `tf_bot_add` defender and a
popfile robot have one, a `CreateFakeClient` body does not. The runner turns it
on with the puppets, before the arm cvars, so an arm can still turn it off.

**Scout by default.** A Heavy puppet wins the medic's ranking on its body alone,
so a beam on it would say nothing about the call. A Scout is the smallest body
on the team and only takes the beam by being a player or by calling, which is
the two halves of the ask and nothing else. `-puppet-class` changes it.

**The call rides on the poll.** `-puppet-calls` presses `voicemenu 0 0`, the same
command a player's key sends, once every twenty seconds while a wave is running.
The answer time is ten seconds, so half of each gap has no call live: a beam that
sits on the puppet throughout is the player rule, and one that arrives after a
press is the call.

What the run says is in the telemetry, in the medic's `beam went to` line: the
puppet is named after itself, so a beam that moved names it and one that did not
names whichever bot the medic had. `mvmbots_puppet_status` says the same thing
while the wave is still running.

The cvars are `mvmbots_puppet_count`, `mvmbots_puppet_name` and
`mvmbots_puppet_class`; the commands are `mvmbots_puppet_call [n]` and
`mvmbots_puppet_status`. `mvmbots_roster` counts puppets separately from the
host, the humans and the mod's own bots, because a puppet read as a defender is
a run that thinks it measured six bots and measured five.

**The break a losing team takes.** A player's team sits in the ready-up after a
loss, and that is where the launcher's Bot Team tab saves a new lineup. The host
readies the instant a round ends, so that window never existed here and the
fault reported in it (mvm-tcc) could not be played. `-ready-delay` holds the
host and the puppets in the ready-up for that long after every round, and
`-relineup-after-loss` types a lineup into the first such break the way the
launcher does, the four convars then `sm_redbots_reseat`:

```sh
go run ./cmd/testbed -map mvm_rottenburg -waves 2 \
  -ready-delay 60s -relineup-after-loss "pyro,soldier,demoman,heavyweapons,engineer,medic" \
  -arm plain:
```

The runner says what RED holds at every poll after the retype, and the next
`wave_begin` line says how many bots the wave started with. The watcher allows
RED to stand empty for the delay plus two polls before it calls the run off, so
a team that never comes back is reported by name rather than waited on.

A puppet reproduces what a player does, not what a player feels. It has no
input timing, no interpolation and no packet loss, so "the medic feels
unresponsive" is still a play-test question.

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

The file carries two more records the server did not write. A **run record**
says what was played and on what machine, so a file found a week later is
self-describing. An **attempt record** says what the runner made of it:
finished, crashed, empty or refused, the crash kind, the watcher's reason, and
whether a crash happened again on its replay. Without that second one, a run
that had crashed twice read back as a clean one and the measurement was taken
again.

For a question narrower than the report prints, ask for the field rather than
writing a script over the file:

```sh
go run ./report results/x-on-1.jsonl -json                 # the report's numbers, without the words
go run ./report results/x-on-1.jsonl -field robot_kills    # the series, count and quartiles
go run ./report results/x-on-1.jsonl -field action -where class=engineer
```

The names come from the generated record, which is the same table the plugin's
`FormatEx` is generated from, so a field that is not written is a refusal rather
than a column of zeros.

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
testbed/seed-native.sh                          # once: copies the game out of the container
go run ./cmd/testbed -native -waves 6 -arm x:   # the native path
testbed/symbolise-core.sh core.1234             # a backtrace with names in it
```

`-native` is a mode of the runner and not a second program. It was a shell
script of its own, with none of the guards the runner exists for: no check of
the loaded plugin against the source, no interleaved arms, no machine record, no
watcher, no bed lock. It wrote results files the reports could not tell apart
from a real run's, which made the one measurement that would settle this the one
measurement nothing could vouch for. The runner refuses a native run for the
same reasons it refuses a container one.

It runs `srcds_linux` directly rather than through `srcds_run`, so a crash stays
crashed and leaves a core instead of being restarted underneath the measurement.
The runner raises the core limit for the server it starts, and `-crashes` names
the core and prints the `symbolise-core.sh` line for it.

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
TESTBED_NATIVE_ROOT=~/path/to/tf2ap/tf-dedicated go run ./cmd/testbed -native -waves 12 -arm x:
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

The runner notices a server that stops answering rcon and stops with a message
rather than waiting out the timeout, because from outside a crashing server and
a slow one look the same: no new results either way.

The first thing the test-bed ever found was a crash in the branch it was built
to measure. That is what it is for. To chase one:

```sh
go run ./cmd/testbed -crashes                # the last hour, classified
go run ./cmd/testbed -crashes -since 20m     # or a window of your own
```

That reads the container log scoped to the window and says which of the faults
it was, because they are not variations on one thing: a watchdog kill means
something was slow, a `SIGSEGV` means something is corrupt, and a `SIGBUS` means
a file was rewritten under the running server. It also names the newest core and
prints the `symbolise-core.sh` line for it, which is the difference between a
backtrace and two sessions of guessing.

Do not read the log by hand. `docker logs` with no `--since` keeps the whole
life of the container across restarts, so a count taken over that window charges
this attempt with crashes from hours ago; a bead was filed on exactly that
inference and closed as one. And `srcds_run` prints its "add -debug" restart
line every thirty seconds for the whole of an install, so a `grep -c` over it
reads dozens of restarts as dozens of crashes.

A crash that happened in one arm and not the other is not evidence yet. The
runner replays a crashed attempt once on the same arm, and only a crash that
happens again is charged to the arm; the rest are the bed's and are reported as
`bed crashes`.

## Watching a run

A run in flight writes what it is doing to `$TMPDIR/<bed>-run.json` at every
poll, so a second shell can read it without a log path, `pgrep` or `docker`:

```sh
go run ./cmd/testbed -status        # what the bed is playing, and until when
go run ./cmd/testbed -follow        # each wave result as it lands, with a running tally
go run ./cmd/testbed -wait          # block until it ends, exit with its verdict
go run ./cmd/testbed -bed list      # every bed on this machine and who holds it
```

All four play nothing and take no lock, and all four follow a bed rather than a
process, so they can be pointed at a run somebody else started. `-json` on any
of them is for a reader that is not a person.

`-follow` is worth the habit on a long mission: a six-wave Mannhattan takes
forty minutes to say anything otherwise, and a run whose first two waves already
say what the change did is a run that can be stopped at attempt two.

Never bring a bed up or down with `docker compose` by hand. `compose.yml`
defaults its project name to the first bed, so a compose line typed without
`TESTBED_PROJECT` recreates somebody else's server, and `pkill -f` on a test-bed
pattern kills their runner. `-bed up` and `-bed down` exist so the project name
cannot be left off.

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
| `cmd/checkspots`        | which dispenser spot each authored nest would take     |
| `build.sh`              | compiles the mod on the host, into `build/package`     |
| `seed-volume.sh`        | copies an existing game install into the bed's volume  |
| `install-game.sh`       | downloads the game from Steam, when there is none to copy |
| `seed-native.sh`        | copies the game out of the volume for `-native`        |
| `compose.yml`           | one service, loopback only                             |
| `entrypoint.sh`         | installs into the game volume, writes `server.cfg`     |
| `stats/mvmbots_stats.sp`| the plugin that counts                                 |
| `stats/mvmbots_host.sp` | the fake client that holds a seat and readies up       |
| `loadouts/`             | a loadout to run instead of the shipped one, via `TESTBED_LOADOUT` |
| `replays/`              | somebody's `server.cfg` for `-replay`; ignored by git   |
| `cmd/rc`                | one console command over rcon                          |
| `symbolise-core.sh`     | turns a core into a backtrace with names in it         |
| `versions.env`          | every pinned version                                   |

`build/` and `results/` are working directories and are not committed.

## Running an A/B

`go run ./cmd/testbed` replaces the shell for anything being measured.
It builds, restarts the server onto what it built, plays each arm, and prints
the comparison.

```
go run ./cmd/testbed \
  -map mvm_decoy -mission mvm_decoy_exp_dissolution \
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
