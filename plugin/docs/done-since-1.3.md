# Done since 1.3

One line each. The spec has the detail and the reasoning.

- Engineers buy for their primary and secondary, ranked under the sentry.
- Sentry busters: the engineer hauls the sentry out of the way, everybody else
  runs. The evade action existed and had never once run.
- The wrangler comes out for the shield, not only for the reach.
- Nests have a minimum distance from the bomb, and are scored on how much of
  the approach they can actually see rather than one trace to one point.
- Nest spots are read from the map's own `bot_hint_sentrygun` entities.
- Medics deploy by medigun type instead of by panic.
- Upgrades are bought several tiers at a time, which is one chat line each.
- Scouts jump.
- Demomen use the sticky launcher, lay a trap with it, and detonate it.
- Spy checking: paranoia from the last sighting, suspect the teammate who was
  not there a moment ago.
- Rockets aim at the ground when the splash pays. That was written for a code
  path that is not compiled.
- Bots stop riding a teleporter backwards to reach its entrance.
- `testbed/` measures a run: waves cleared, how long, who died to what. It
  plays a mission with nobody on the server.
- Six maps carry hand written nest, dispenser and teleporter exit data, walked
  in game rather than guessed. See item 3.
- Engineers place dispensers on authored spots, and skip one another engineer
  already occupies.
- Engineers re-score their nest when a wave ends and haul to a better one.
- Rottenburg's two conditional nests work: `NestTankOnly` and `NestNoTank`,
  decided by the wave's class icons rather than by looking for a live tank.
- `PickBuildArea` skips `BLOCKED` areas. Only `PickBuildAreaPreRound` did.
- `GetBombInfo` tested `BLUE_SPAWN_ROOM` twice where it meant `RED_SPAWN_ROOM`,
  in two places, so RED spawn counted toward the length of the bomb path.
- Engineers build teleporters. The entrance comes off the nav mesh's own route
  out of spawn instead of a straight line through the walls, the exit stands
  beside the nest instead of on the sentry, and an all-bot team holds the ready
  long enough for him to do it. See item 5.
- The medic picks his patient from the whole team rather than from whoever is
  within nine metres, which is what parked him at the engineer's nest for the
  length of a wave. See item 14.
- The bots wear hats rather than tournament medals. The pool was drawn from the
  schema's misc slot, which is mostly UGC and ozfortress season badges; it is
  filed by equip region now, because the head slot is the game's old single-hat
  one and no modern item reports it.
- `AddBotsFromChosenTeamComposition` counts who is already on RED. It added the
  whole lineup, which is right only while nothing else fills the team before the
  wave, and tf2-archipelago now does.
- The dispenser build stopped ending itself three seconds in on a flag only the
  idle action refreshes. Mannworks went from 13% dispenser uptime to 39%.
- An engineer with no sentry builds one instead of considering a move, which is
  what left him standing on Bigrock for most of a wave.
- The sentry stands in front of the engineer like the other two buildings, and
  every part of building one has a clock on it.
- `testbed/sweep.sh` plays every installed map and `testbed/sweepreport` reads a
  whole sweep. See `specs/sweep-2026-08-22.md`.
- The loadout file can name a seat of `sm_redbots_manager_team_composition` and
  not only a class, so seat 1 holds the wrangler and seat 2 need not.
- The crash was the test-bed's own entrypoint: `cp -r` over a running server's
  mapped extension every thirty seconds, so the next instruction out of that
  extension was a SIGBUS. `cp -ru` fixes it. `specs/` has the full account, and
  the way to get a backtrace out of a stripped 32-bit server is worth keeping:

  ```
  gdb -batch -ex 'set auto-solib-add off' -ex run -ex 'info proc mappings' \
      -ex bt --args ./srcds_linux <the srcds_run arguments>
  ```

