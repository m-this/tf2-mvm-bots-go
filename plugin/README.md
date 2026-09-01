# MvM Defender TFBots
TFBots that can play Mann vs Machine.
<p>This is a constant work-in-progress. Expect many things to change down the line.

> [!IMPORTANT]
> There have been a few reports about the bots not working as intended with external mods such as [sigsegv-mvm](https://github.com/rafradek/sigsegv-mvm). If you need more help regarding compatibility issues with your installation, please contact me in my _Discord_ development server. External mods are not natively supported!

# Requirements
- [[TF2] TF2Attributes](https://github.com/FlaminSarge/tf2attributes)
- [[TF2] Econ Data](https://github.com/nosoop/SM-TFEconData)
- [TF2 Utils](https://github.com/nosoop/SM-TFUtils)
- [CBaseNPC](https://github.com/TF2-DMB/CBaseNPC)
- [Actions](https://forums.alliedmods.net/showthread.php?t=336374)
- [REST in Pawn](https://github.com/ErikMinekus/sm-ripext)
## Compilation Only
- [stocklib_officerspy](https://github.com/OfficerSpy/SM_Stock_OfficerSpy)
# Testing
`testbed/` runs a server with nobody on it and writes down what the bots did
with every wave: cleared or lost, how long, how many robots and defenders died,
how many backstabs, what the engineers lost. It also samples every bot and every
building every five seconds, so "the engineer was stood in a house" and "that
dispenser fed nobody" are numbers rather than reports. The runner and the reports
are Go and live in the sibling checkout since `mvm-x2c`: `cmd/testbed` plays a
mission and `report` reads a run and compares two, both from `../tf2-mvm-bots-go`.

See `testbed/README.md` for running it, `docs/testbed-metrics.md` for what the
numbers mean, and `docs/how-bots-break.md` before debugging anything: the faults
in this mod have one shape, and it is worth knowing it before guessing.
`docs/spawn-nav-recovery.md` documents the community-map fallback for defenders
that cannot leave RED spawn.
`docs/engineer-and-medic.md` covers the two classes that generate most of the
reports, including two fixes that looked obviously right and lost their A/B.
`docs/soldier-and-demoman.md` covers the two lowest-scoring seats, where the
answer turned out to be self-inflicted blast rather than aim.

# Notes
- Initial AI code is a port over from [[TF2] MvM AFK Bot](https://github.com/Pelipoika/TF2_Idlebot) AI code.
- Internal PluginBot based on C++ code from [PathFollower](https://github.com/Pelipoika/PathFollower).
- Certain functions have functionality based on Valve's own code from the game.
- Tank Weapon Score System based on C++ code from mvm defender bots behavior from [sigsegv MvM](https://github.com/sigsegv-mvm/sigsegv-mvm).
- Custom Sniper spots for various maps by Us_le.
- Spy checking and the stickybomb trap follow the approach in [RCBot2](https://github.com/chrizonix/RCBot2) by Cheeseh: paranoia that grows from where a Spy was last seen, a suspect picked out as the teammate who was not there a moment ago, and a bomb-by-bomb trap laid across a spread rather than onto one point. The code is ours; the ideas are worth the credit.
- Engineer nest spots are read from the map's own `bot_hint_sentrygun` and `bot_hint_engineer_nest` entities where a map carries them.
- Spawn navigation recovery by [kelly-cs](https://github.com/kelly-cs), which is what lets the bots play community maps whose nav mesh leaves them stuck in spawn.
