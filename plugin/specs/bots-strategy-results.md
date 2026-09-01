# What the strategy work measured

Six runs of `mvm_decoy`, three per build, alternating, on a machine with nothing
else running. Lineup forced to `scout, soldier, demoman, heavyweapons, engineer,
medic` on both sides, so the run measures the code and not the team.

`base` is 1.5.5-tf2ap.10. `current` is this branch.

| run | waves | cleared | deaths | kills |
| --- | --- | --- | --- | --- |
| current-1 | 5 | 5 | 27 | 380 |
| current-2 | 6 | 5 | 54 | 447 |
| current-3 | 6 | 5 | 65 | 517 |
| base-1 | 6 | 5 | 38 | 452 |
| base-2 | 6 | 5 | 46 | 420 |
| base-3 | 6 | 5 | 44 | 410 |

## The answer

Parity. Every run on both builds cleared exactly five waves. Defender deaths
average 49 on this branch against 43 on the old code, and the branch's spread is
wider: 27 to 65 against 38 to 46.

So the research produced code that reads well, holds its own, and does not beat
the old behaviour on this mission. That is worth writing down plainly rather
than picking the run that flatters it. An earlier ten-run batch said the same
thing, 4.6 waves against 4.2.

## What was worth it anyway

The test-bed earned its keep by finding three defects that had nothing to do
with strategy:

- the Gas Passer crashed the server, bisected to the weapon and then to the
  `HasAmmo` test that is always true for a charge meter. Item 12
- `NestBuildPosition` read the centre of a null nav area, because the null check
  sat after the call that needed it. Mine, from the same session
- the upgrade teardown was gated on a nest moving, so with relocation off it
  never ran and engineers started waves behind a level 1

And two facts about the harness that invalidate older numbers:

- a machine that is paging cannot measure this. 1.5.5-tf2ap.10 failed five runs
  in a row at 200 MB free and then ran six clean at 1.5 GB. Item 13
- `fps_max 0` had the server spinning a whole core producing frames past a
  66-tick simulation

## What is still not measured

Mannworks floors at one wave on both builds, so it separates nothing. Mannhattan
carries the nest zones and the two-Pyro lineup and has never been run at all.
Decoy, which is what the numbers above are, is the map where this work should
matter least: its lineup was already the standard six and its nest spots were
already sensible.
