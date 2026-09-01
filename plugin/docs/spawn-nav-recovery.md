# Community-map navigation recovery

Defender bots recover automatically when a community map puts RED spawn on a
disconnected or unusable NAV island. After shopping, a bot with no route, no
meaningful movement for six seconds, or twelve seconds spent near spawn moves
to walkable NAV near the final objective. Shopping players, humans and bots
that have left the recovery radius are ignored.

The default recovery area includes 512 units around every RED spawn brush.
These settings take effect immediately:

```text
sm_redbots_spawn_nav_recovery 1
sm_redbots_spawn_nav_recovery_radius 512
sm_redbots_spawn_nav_recovery_time 12
```

The radius accepts `0` to `4096`; the deadline accepts `1` to `120` seconds.
`sm_dump_spawn_nav` reports each bot's eligibility, spawn distance, timers and
selected objective anchor. `sm_recover_spawn_bots` runs recovery immediately.
Both require generic admin access.

Recovery uses the map's capture trigger, `func_capturezone`, control point or
bomb-target NAV distance, in that order. It is map-independent, but still
requires usable NAV near one of those objectives. Successful recovery logs
`SpawnNavRecovery`.

The initial playtest covered Area 52 and Thriller. Official Valve maps did not
show the stuck-spawn behavior.
