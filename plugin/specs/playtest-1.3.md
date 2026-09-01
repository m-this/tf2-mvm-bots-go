# Playtest report, 1.3

Two days of play on Decoy and elsewhere, mostly stock loadouts. Every item below
is the report, what the code actually did about it, and what was done. Line
numbers are from the tree this was written against.

The report is one player's. Where it names a cause rather than a symptom, the
symptom is what is recorded here and the cause is checked against the code.
Two of the items turned out to be reports of code that had never run at all.

Most of it is fixed. None of it is play-tested. What is left is at the bottom.

## Engineer

The engineer is the class the report spends the most words on, and it is the
class the mode is decided by. Everything in this section is one problem seen
from five sides: the nest is chosen for the bomb and for nothing else.

### 1. The pre-round nest sits on the hatch. Done

A nest now has a floor as well as a ceiling: `IsNestRangeSane` refuses ground
closer to the bomb than a third of the sentry's range, in both pickers, with a
last-resort tier so a map that offers nothing else still gets a nest. Valve
keeps its own engineer robots 1300 units off the bomb for the same reason.

The line-of-fire test is no longer one trace to one point. `NestSightScore`
asks the nav mesh how many pieces of the approach the area can see, sampled at
two dozen spread across the ground within a sentry's range of the bomb.
Visibility between areas is computed when the mesh is built, so it is a lookup
rather than a trace, and a mesh built without it scores every candidate zero
and changes nothing. That is what "blocked by the walls" was.

Scoring the team rather than the bomb is not done.

Reported: sentries firing into walls, dispensers nowhere near the team, and an
engineer that dies at the start without having contributed anything.

`PickBuildAreaPreRound` in `util.sp:1878` anchors on the hatch, keeps areas on
the bomb path within `NestDistanceLimit`, and prefers the ones with a line to
the hatch. `NEST_HATCH_CLEARANCE` is 180 units, which is the only thing keeping
a sentry off the hatch itself. `sm_redbots_manager_engineer_nest_depth` defaults
to 0.4, so the whole search is the back four tenths of the path.

That was a deliberate change and it overshot. Anchoring on the robots' spawn
door was worse, but the answer is not the other end of the path. The ground
worth holding is where the team fights, which is neither.

What it needs:

- A forward floor as well as a ceiling. An area closer to the hatch than some
  fraction of the path is not a nest, it is a last stand. Same convar, a second
  bound.
- A line of fire test that is not the hatch. `IsEntirelyVisible(hatch)` says
  nothing about whether the sentry sees the ground the robots walk over. Sample
  the bomb path itself: how many path points within sentry range the area can
  see. That is the number the report is complaining about, and it is what
  "blocked by the walls" means.
- Score the team, not just the bomb. A nest the other bots pass through is a
  dispenser that heals somebody. Distance to the rest of the lineup's held
  ground belongs in `ScoreNestArea` (`util.sp:1648`).

### 2. No map has nest data. Wrong, for four maps of seven

`grep -l EngineerNest configs/defenderbots/map/*.cfg` returns nothing, so the
configured spots are empty on every map and the nav reasoning is not a fallback
but the whole thing. That part of the report holds.

What does not hold is the assumption under it, that this data has to be written
by hand. Four of the seven official maps carry it already. Decoy has thirteen
`bot_hint_sentrygun`, thirteen `bot_hint_engineer_nest` and thirteen
`bot_hint_teleporter_exit`; Bigrock, Rottenburg and Mannhattan carry between
fourteen and twenty-seven of each. They are BLU's, placed for the engineer
robots, and they point the wrong way. A sentry shoots in a circle, so the only
thing the facing costs is the first second, and what the entity really says is
that a level three fits here and has a line down the lane.

`PickMapHintNestArea` reads them at runtime, between the configured spots and
the nav reasoning, and scores them like anything else, so a spot deep in the
robots' half loses on its own merits. Reading them rather than copying them
into the configs means community maps built on the same prefabs are covered
without anybody authoring anything.

Coaltown, Mannworks and Ghost Town carry none of it. They predate engineer
robots. Those three still want hand written spots, and a hand written spot
still outranks everything else: somebody stood there.

### 3. No teleporters, and possibly for the better. Gated

Reported first as a complaint, then withdrawn: the other bots take the
teleporter even when the bomb is at spawn, which pulls the team backwards.

The teleporter code refuses to build without `TeleporterEntrance` and
`TeleporterExit` in the map config, and no map has either, so nothing has been
built yet. The withdrawal is about Valve's bots, not ours.

That changes the shape of the feature, and the ride was worth gating on its
own. `ShouldUseTeleporter` returned true in both of its branches, so the caller
that exists to stop a bot looking for a teleporter never stopped anything. The
entrance is at spawn, so riding one means walking backwards to reach it: bots
left the hatch, rode forward, and arrived roughly where they started, having
given the wave the seconds it took. It now refuses unless the fight is at least
1500 units further up the path than the bot is, which is the case where the
walk is bought back.

Building one is still not done and should stay that way until somebody answers
whether it is worth building at all. No new map data until then.

### 4. Nothing is done about sentry busters. Done

Reported: the bots do not react, do not run, do not haul the sentry away.

`CTFBotEvadeBuster_IsPossible` (`behavior/evadebuster.sp:76`) is false until
`g_iDetonatingPlayer` is set, and that is set in `tf2_defenderbots.sp:755` when
the buster starts its detonation taunt. By then there are about two seconds
left. The escape then walks to the first nav area more than 500 units away,
which is inside the blast, and an engineer within 500 units of its sentry walks
towards the sentry to pick it up, which is towards the bomb.

It was worse than that. Nothing ever suspended for `CTFBotEvadeBuster`, so the
whole action was dead code and none of it had ever run.

There are two answers now, at two distances. Far out, the engineer picks the
sentry up and walks it to ground `PickBusterRetreatArea` chose for being a
blast further from the buster, using the carrying machinery the nest advance
already had, and walks it back when the buster is gone. Close in, everybody
runs, engineers included, to the furthest ground within 1500 units rather than
the first area past a threshold. The evade is suspended for from the tactical
monitor, above health and ammo, because a bot walking to a health pack through
the blast arrives dead.

A mini sentry is not carried. It costs 100 metal and two seconds, so a
Gunslinger engineer lets the buster have it.

### 5. The wrangler is never pulled. Done

`TF_ITEMDEF_WRANGLER` is defined at `util.sp:1597` and used nowhere. A wrangled
sentry is immune-ish and shoots where the engineer looks, which is most of what
a wrangler engineer is for, and it is also the buster answer that does not cost
a rebuild.

It was half used, not unused: `CTFBotMvMEngineerIdle` already pulled it for a
threat beyond the sentry's range, which is the reach half of the weapon. The
shield half was missing, and the shield is what the report is asking for.

`UpdateSentryUnderFire` watches the sentry's health for a drop, which is the
only thing that says it is being shot at without a damage hook, and the same
block now deploys the wrangler for three seconds after the last health it lost.
Two thirds of the damage aimed at the sentry stops there. It costs the seconds
of aim, and a sentry that survives the giant is worth more than that.

### 6. Engineers buy nothing for their primary or secondary. Done

Reported as "never buy metal capacity", which is not quite it. Metal is bought:
`metal regen` is 220 and `maxammo metal increased` is 210 under
`ClassUpgradePriority` (`behavior/upgrade.sp:485`).

The real hole is next to it. `CollectUpgrades` (`behavior/upgrade.sp:176`) gives
an engineer melee, building and PDA slots and never primary or secondary. So:

- A Widowmaker engineer buys no damage and no metal on hit, which is the whole
  weapon. Every shot he fires is metal the sentry does not get.
- A Rescue Ranger engineer buys nothing for it either, and the rule written for
  it at `behavior/upgrade.sp:471` is dead code that never runs.
- A Wrangler engineer buys nothing for the secondary.

Both slots are collected now, and ranked so the gun never outbids the nest: an
engineer's primary and secondary upgrades sit under every sentry line, because
the general table would otherwise put "damage bonus" at 260 and buy shotgun
damage ahead of the metal that keeps the sentry firing.

The weapons that are the reason to carry the loadout say so for themselves. The
Widowmaker buys damage, which is what pays its metal back. The Short Circuit
buys metal regeneration. The Frontier Justice buys the clip that holds its
banked crits. And an engineer whose gun is paid for out of the metal supply at
all now puts metal capacity and regeneration second only to the sentry's own
fire rate, wherever the game happens to attach those upgrades.

## Medic

### 7. Uber is popped at death's door. Done

Reported: medics hold uber until they are nearly dead, which wastes a Kritzkrieg
entirely.

Nothing in this repository deploys uber. `m_bChargeRelease` appears nowhere. The
medic runs Valve's `CTFBotMedicHeal` and only its edges are patched
(`nextbot_behavior.sp:545`): the fetch-flag override, the resist medigun swap,
and `CTFBotAttackUber` for building charge. Valve's rule is a panic rule, which
is right for a stock medigun and wrong for everything else.

`medic_uber.sp` asks each medigun its own question, because each one is carried
for a different reason. The Kritzkrieg pops when the patient is shooting and
there is a giant or three robots to shoot at. The Quick-Fix pops on damage
taken, at seventy percent, because it is a heal rate and not an
invulnerability. The Vaccinator pops on any damage at all, because a quarter of
a meter is cheap. Stock keeps the panic rule and moves it off the floor to
half health, so the charge is spent on the fight rather than on the retreat.

None of it suppresses Valve's rule, which it cannot: the deploy belongs to the
game and a plugin can only press the button earlier. Earlier is the whole fix.

The enum this reads was also wrong. `MEDIGUN_UBER` was the name on the value
the game uses for the Kritzkrieg; it is `MEDIGUN_CRITBOOST` now, which is what
Valve calls it.

### 8. Bots repeat the same upgrade line. Done

Reported on Medic and Engineer, which are the two classes that buy four tiers of
one attribute in a row. The plugin sends nothing to chat outside debug paths, so
this is the game announcing each purchase, once per tier, and the bots buying
every 0.1 to 1.25 seconds is what turns it into a flood.

Both the repetition and the "purchase spam clogging chat" item are the same
thing, and the game already had the answer in it. `MVM_Upgrade` takes a count
and the code was passing 1. It now passes every step the bot can afford, up to
four, which is one announcement instead of four and a bot that finishes
shopping sooner.

Nothing about what gets bought changes. The list is a strict priority and the
top of it stays the top until it maxes out, so the steps bought in one go are
the ones the next four intervals would have bought anyway.

## Demoman

### 9. Stickies still do nothing. Done

It is not a preference, it is the absence of one: `EquipBestWeaponForThreat`
listed the Demoman with the classes that only ever want their primary, with no
branch at all.

Two halves, and both are in now. `demoman_stickies.sp` is the weapon used
directly: the launcher comes out for giants, for the bomb carrier and for a
crowd, and the bombs go off when two robots or one giant stand in the blast, or
when one is stuck to a tank, which is a case none of the player counting sees.
Against a tank the launcher scores 110, above the melee it used to lose to.

`behavior/stickytrap.sp` is the trap, and the state machine is Cheeseh's from
RCBot2's `deployStickies`. With nothing to fight, the Demoman puts eight bombs
across a spread on the ground the robots are walking to, which for a defender
is wherever the bomb is, one bomb at a time with a gap between them so the
launcher keeps up and the bot has time to turn. It gives up on a deadline, it
gives up the moment something shoots at it, and it will not lay a trap it is
standing in.

## Scout

### 10. Scouts never jump. Done

`IN_JUMP` does not appear anywhere in the source. Not in combat, not while
collecting money, not to cross a gap. So the report is exact: a scout is a flat
target at all times.

`UpdateScoutCombatJump` runs from the tactical monitor: on the ground, moving,
within 900 units of a threat it has seen recently, not stunned, on an irregular
half to one and a fifth second cooldown, because a jump on a fixed beat is as
easy to lead as no jump at all.

Only a Scout. Every other class is slower in the air than on the ground, and a
Heavy who leaves it has traded his aim for a hop.

## All classes

### 11. Slow to react to spies. Done

The honest version of this is not omniscience, and RCBot2 had already worked
out what the honest version looks like. `behavior/spycheck.sp` follows it.

Paranoia first. A team that has seen a Spy knows one exists and does not know
where he went, so the sighting is remembered as a position and a time, and a
circle around it grows at a walking pace and stops mattering after twenty
seconds. A bot inside the circle checks; a bot outside gets on with the wave.
A sighting is a Spy with no disguise on in plain view, a Spy whose cloak has
been broken, or a Spy who has just killed somebody, which is `player_death`
and the only piece of it that costs nothing to notice.

Then the tell, which is the good part and is entirely Cheeseh's. Looking at
the face is worthless because the face is the disguise. What gives a Spy away
is that he was not there a moment ago: the bot lists the teammates it can see,
watches for one who appears who was not on the list, walks over and hits him.
Being wrong costs nothing, because friendly fire is off. Being right takes the
disguise off. And it stops early if the suspect fires his own weapon, which is
the alibi a disguised Spy cannot produce.

Kept from the first attempt: a Spy at knife range behind the bot for four
tenths of a second gets handed to the vision interface directly. That one is
not paranoia, it is somebody standing too close.

A Spy who keeps his distance and picks his moment still gets his stab.

### 12. Bot seating for specific classes does not work. Open, needs a repro

Carried over from day one, and not diagnosed here. The class preference flags
(`player_pref.sp:84`), the lineup modes (`menu.sp:996`) and
`sm_redbots_manager_team_composition` are three ways to decide the same thing
and they may not agree. This needs a repro from the reporter before any code
changes: which of the three they used, and what they got.

## Not this plugin

### 13. Cash bundles vanish on a lost wave. Open, in the other repository

Reported: buying upgrades with cash bundle money and then losing the wave leaves
the currency negative, with the upgrades kept.

The refund on a lost wave restores the currency the game thinks the player had,
which does not include anything a bundle granted. Nothing in this repository
grants currency; the bundles come from the Archipelago side. Fix belongs there,
and it is a real bug: track granted currency separately and re-grant it after
the wave-loss refund, or grant it as an upgrade-station credit the refund
already accounts for.

## What landed

Everything above marked done, plus two things the report did not ask for and
the code turned out to need.

The sentry buster evasion was dead code. `CTFBotEvadeBuster` existed, nothing
ever suspended for it, so none of it had ever run. It runs now, from the
tactical monitor, and it was rewritten while it was being connected: it starts
on a live buster rather than a detonating one, and it runs to the furthest
ground it can find rather than the first area past an arbitrary 500 units,
which was inside the blast. The engineer half is separate and lives in
`CTFBotMvMEngineerIdle`, which already knows how to carry a building.

Rockets at the ground was dead code too. `ShouldAimRocketsAtFeet` was written
into the aiming path behind `IDLEBOT_AIMING`, which is not compiled, so the
last release shipped it doing nothing. The live aiming is Valve's and it aims
at the middle of the robot, which is right for one robot and wrong for the line
of them walking a choke. `CTFBotMainAction_SelectTargetPoint` asks it now.

New files: `medic_uber.sp`, `demoman_stickies.sp`, `behavior/spycheck.sp`,
`behavior/stickytrap.sp`.

Two of them follow RCBot2, credited in the README and at the top of each file.
The engineer's under-fire detection turned out to be the same health-drop test
RCBot2's `lookAfterBuildings` uses, arrived at separately.

There is a test-bed now, in `testbed/`. It runs a server with nobody on it and
writes one line of JSON per wave: cleared or lost, how long, robots killed,
defenders killed, backstabs, sentries lost, busters detonated. `report.py`
compares two runs per wave, which is the only way to answer whether any of the
above helped. The server stack is tf2-archipelago's, which already built this
mod in Docker; what is new is that nobody plays and everything is counted.

Verified by compiling against SourceMod 1.12.7164 with the dependency includes.
Warnings went from 58 to 54; every one that is left was already there.

Nothing here has been played. The numbers in it are arguments, not
measurements: the nest range floor, the buster distances, the crowd that is
worth a sticky cluster, how long a Spy has to stand behind a bot. They are the
first thing to change if the next play-test says so.

## What is left

1. Run the test-bed. Everything above is an argument until a wave is counted.
2. Nest scoring that weighs where the rest of the team is, not just the bomb (item 1).
3. Hand written nest spots for Coaltown, Mannworks and Ghost Town, which carry no hints (item 2).
4. Whether an engineer should build a teleporter at all, before any map data is authored (item 3).
5. The launchers that want playing differently: Quickiebomb, Scottish Resistance (item 9).
6. A repro for the class seating (item 12).
7. The cash bundle refund, in tf2-archipelago (item 13).
