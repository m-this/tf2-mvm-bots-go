# The map spots, and where they came from

Six maps were flown in game and every spot stood on, on 20 August 2026. This is
the record of what was captured and what was said about it, because a
coordinate in a config file cannot say why it is there.

`sm_dump_spot <block> [aim]` prints the line to paste. Standing on the spot is
the accurate way; `aim` traces the crosshair to the world, which is how a map
gets marked from above without landing on every spot.

The raw capture stays in `testbed/results/` and is not tracked: `spots.log` is
what the command printed and `chat.log` is what was said while printing it.
This file is the part worth keeping.


Things the screenshots say that the config format cannot currently express.

## Rottenburg: spots that depend on the wave type — DONE

Implemented. `NestTankOnly` and `NestNoTank` are map config blocks, and `IsTankWave()` in
util.sp reads the wave's class icons off `m_iszMannVsMachineWaveClassNames` so the answer is
available between waves, before any tank_boss exists. Original note kept below.



Two of the marked sentry spots are conditional:

- one nest is marked "tank wave only"
- one nest is marked "don't build here on tank wave"

The config format has no per-spot condition, so both are unrepresentable today. Recording
them as ordinary EngineerNest entries would be wrong in one direction or the other on every
wave.

What it would take:

- a per-spot key in the map config, e.g. "wave" "tank" / "wave" "notank", defaulting to any
- knowing whether the coming wave has a tank BEFORE it starts

The second is the harder half. The mod only ever detects a tank by finding a live `tank_boss`
entity (`dbtm.sp:10`, `demoman_stickies.sp:118`, `attacktank.sp:153`, `nextbot_behavior.sp:954`),
which is too late: the engineer picks its nest and builds during the between-waves period, when
no tank exists yet. The wave class icons on `tf_objective_resource` are the candidate source,
since they are populated before the wave starts and include the tank icon.

## Rottenburg: the teleporter exit is a relationship, not a point

The blue annotation reads "Opposite upgrade station" rather than marking a spot. That is a rule
about where the exit goes relative to another entity, not a coordinate, and the config format
only stores coordinates.

## Mannhattan: nests belong to a gate

The courtyard nests cover Gate A or Gate B. Which one is right depends on where the robots
spawn that wave, which nothing in the mod currently senses. Recorded per side in
spots-annotated.tsv so the fact survives until there is something that can use it.

## Every spot, with what was said about it

```
# map           block                    origin           note
mvm_bigrock     EngineerNest             -283 3781 316    leftmost red circle, sentry
mvm_bigrock     EngineerNest             -286 3783 316    duplicate of the line above, 3 units away
mvm_bigrock     DispenserSpot-discarded  38 4584 248      early aim, superseded by -4 4580 246
mvm_bigrock     DispenserSpot-discarded  72 4589 257      early aim, superseded by -4 4580 246
mvm_bigrock     DispenserSpot            -4 4580 246      rightmost green circle, the only dispenser here (confirmed in chat)
mvm_bigrock     EngineerNest             -423 4579 260    rightmost red circle, on the rocks
mvm_bigrock     EngineerNest             -552 4266 250    middle red circle
mvm_bigrock     DispenserSpot            -1086 4386 133   second dispenser, far from the first
mvm_bigrock     TeleporterExit           -178 3921 318    first blue circle
mvm_bigrock     DispenserSpot-skipped    (not recorded)   bottom green on the crates: a sniper spot, skipped on request
mvm_coaltown    EngineerNest             467 1887 496     first coaltown nest, unlabelled
mvm_coaltown    DispenserSpot            453 1092 494     first coaltown dispenser
mvm_coaltown    DispenserSpot            525 1542 273     second coaltown dispenser, 220u lower than the first
mvm_coaltown    DispenserSpot            -328 964 422     third coaltown dispenser (3 greens on the capture, i miscounted)
mvm_coaltown    EngineerNest             -193 938 406     second coaltown nest, 140u from the 3rd dispenser
mvm_coaltown    EngineerNest             329 960 402      third coaltown nest
mvm_coaltown    TeleporterExit           -6 463 672       central building roof; ~200u above the nests, nav reachability unverified
mvm_decoy       TeleporterExit           -361 -159 573    blue circle, roof at bottom left of the capture
mvm_decoy       EngineerNest             292 598 544      first decoy nest
mvm_decoy       EngineerNest             214 537 364      second decoy nest, 180u below the first
mvm_decoy       EngineerNest             -218 737 360     third decoy nest
mvm_decoy       DispenserSpot            -205 1162 365    first decoy dispenser
mvm_decoy       DispenserSpot            320 439 358      second decoy dispenser
mvm_decoy       DispenserSpot            -452 634 352     third decoy dispenser
mvm_mannworks   DispenserSpot            -287 1480 284    first mannworks dispenser
mvm_mannworks   EngineerNest             -178 1131 249    first mannworks nest, ~370u from the 1st dispenser
mvm_mannworks   EngineerNest             -236 1249 252    second mannworks nest, only 131u from the first - the two clustered reds?
mvm_mannworks   EngineerNest             223 1141 240     third mannworks nest, ~460u east of the pair
mvm_mannworks   EngineerNest             1014 885 256     fourth mannworks nest - capture marked 3 reds, confirm
mvm_mannworks   EngineerNest             1014 885 256     CORRECTION: intentional. multiple valid spots in this area; which is best depends on the robot route that wave
mvm_mannworks   DispenserSpot            999 551 256      second mannworks dispenser, pairs with the 1014 885 256 nest
mvm_mannworks   DispenserSpot            -141 725 258     third mannworks dispenser
mvm_mannworks   TeleporterExit           417 663 384      the Inside blue circle: upper floor inside the central house (confirmed)
mvm_mannhattan  DispenserSpot            926 -1777 -64    first mannhattan dispenser, side not yet stated
mvm_mannhattan  DispenserSpot            -723 -629 -68    second mannhattan dispenser, ~2000u from the first
mvm_mannhattan  TeleporterExit           -61 -1351 204    first mannhattan tele exit, 270u above the dispensers
mvm_mannhattan  EngineerNest             214 -1319 204    first mannhattan nest, 277u from the tele exit at same height
mvm_mannhattan  EngineerNest             -362 -1393 203   second mannhattan nest, 580u west of the first, same level
mvm_mannhattan  EngineerNest             -152 -3319 -63   interior capture: warehouse floor nest (chat: now doing the interior)
mvm_mannhattan  EngineerNest             271 -2910 -63    interior capture: second warehouse nest, 580u from the first
mvm_mannhattan  DispenserSpot            -414 -3222 -240  interior capture: dispenser on the lower floor, 177u below the nests (confirmed intentional)
mvm_mannhattan  TeleporterExit           167 -3224 -235   interior capture: tele exit on the same lower floor as the interior dispenser
mvm_mannhattan  TeleporterExit           -131 -2058 -54   third mannhattan tele exit, between the courtyard and the interior
mvm_rottenburg  DispenserSpot            1459 -1775 -582  the green by the boulder
mvm_rottenburg  EngineerNest             1987 -1314 -549  WRONG SPOT, user said so in chat; superseded by the next EngineerNest dump
mvm_rottenburg  EngineerNest             1958 -1289 -543  the plain red up by the rocks (unconditional) - the corrected one
mvm_rottenburg  NestNoTank               1476 -1074 -539  behind the fence: do NOT build here on a tank wave (confirmed)
mvm_rottenburg  NestTankOnly             1855 -720 -415   the wooden platform: tank waves only (confirmed)
mvm_rottenburg  TeleporterExit           1723 -199 -407   the tele exit, opposite the upgrade station (confirmed)
```

## Mannworks: the wave starts inside

Told in play-testing. Every test-bed run on this map clears one wave and stops, on
both builds, so the mission does not tell two builds apart.

An earlier version of this note said none of the four nests was inside and that the
map needed an interior one. That was wrong: it was read off the coordinates by
somebody who has not seen the building.

The interior spots are the ones walked last: the nest at `1014 885 256`, the
dispenser at `999 551 256` next to it, and the teleporter exit at `417 663 384` on
the upper floor. The earlier spots are the open ground to the north. So the map data
already covers the inside, and the reason the mission does not tell two builds apart
is something else.

## Mannhattan: which teleporter exit is where

The highest, `-61 -1351 204`, is outside in the gate courtyard. The lowest,
`167 -3224 -235`, is inside on the warehouse's lower floor, under the sentry ground
and beside the dispenser at `-414 -3222 -240`. The third sits between the two.

That matters because an exit is only worth building if it puts somebody where the
fight is. The inside one delivers under the nest; the outside one delivers to the
gates.
