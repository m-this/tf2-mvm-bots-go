# The soldier and the demoman

The two seats that fight with an arcing, exploding projectile, and the two
lowest-scoring seats on the team. Every number here came out of
`testbed/report`.

## The baseline

```
class      damage per wave   (30 waves, two maps)
heavy         4787
scout         3278
engineer      2903
pyro          2335
demoman       1608
soldier       1004
```

## What it is not

**It is not accuracy.** Six waves on Decoy:

```
projectiles   soldier 465 fired, 187 hit (40%);  demoman 302 fired, 130 hit (43%)
```

Forty percent with a projectile that arcs, against robots that are walking, is
not a bot that cannot shoot.

**It is not that they stand too close.** The Demoman's median distance to the
nearest robot is 1044 units, which is further out than the 600 his own attack
action is trying to close him to.

**It is not one half of the demoman's kit.** Pipes and stickies both land:
`pipes 4164, stickies 3922` over six waves. The stickybomb detonation rule
works.

## What it is

```
hurt themselves    soldier 3988   demoman 2380   (six waves)
soldier damage     rockets 16721
killed themselves  soldier 4
```

**A quarter of the Soldier's output goes into his own feet**, and it kills him
four times in six waves. The Demoman gives up about a sixth the same way.

**And trying to stop it made everything worse.** `explosive_min_range` gave the
Soldier his shotgun inside 220 units and stopped him aiming at feet inside 350.
Six waves on Decoy against six without:

```
                        ON      OFF
soldier damage       10886    16890
soldier accuracy       40%      60%
soldier self-kills       3        6
demoman damage        8269    11265
defender deaths         47       28
```

The hit rate is the explanation. **The ground does not move and a robot does.**
A rocket at the floor lands and splashes whatever the robot did next; one aimed
at a chest arrives where the chest was. The splash he catches is the price of
the shots that land at all, and it is a price worth paying: he traded a third
of his damage for three fewer self-kills, and the team died more anyway.

So the self-damage is real, it is large, and it is not a defect. Two of the
three things below are gone.

The original reasoning, kept because it is the reasoning anyone will have:

1. **No minimum range on an explosive.** `EquipBestWeaponForThreat` never asked
   how near the threat was, so a Soldier fired rockets into whatever walked up
   to him. He carries a shotgun that does not explode; the Demoman carries a
   bottle.
2. **Feet-aiming had no lower bound.** Shooting the ground makes a rocket
   unreflectable and catches a crowd, and both are worth having at a distance.
   Up close the ground under the robot is the ground under the Soldier — and
   the rule fires unconditionally on robot Pyros, which are the class that
   closes to his face. The shot written to dodge a reflection was the one
   making the splash.
3. **Blast resistance was priced by the wave.** Resistances are ranked by what
   the coming robots carry, which is the right question for damage somebody
   else deals. These two explode themselves in every wave whatever the robots
   are made of, so the resistance has a floor for them now.

**Blast resistance did not pay either.** `blast_resist_self` put a floor under
the resistance for these two, on the grounds that their own weapons are in
every wave. Six waves on Decoy each way:

```
                        ON      OFF
soldier self-damage   2147     3272     <- the resistance works
soldier self-kills       1        3
soldier damage       13485    14187
waves cleared            3        3
defender deaths         50       40
```

The mechanism does what it says and buys nothing with it. Credits spent there
come out of upgrades that produce damage, and on this harness a five percent
swing over six waves is inside the noise. Deleted.

## What worked: letting him shoot

`demo_hold_fire` capped his pipes at 900 units. Twelve waves on two maps, each
way:

```
                  demoman dmg   shots   accuracy   team deaths   cleared
ON  (cap 900)           20073     450        43%           146         6
OFF (cap 1400)          22285     492        40%           103         7
```

Better on every axis except the hit rate, which is the axis that does not
matter: **the shots he was not allowed to take are worth more than the ones he
lands.** +11% damage, 30% fewer team deaths, one more wave.

It had won an A/B before, bundled with `attack_strafe` and never split. Split
now, `attack_strafe` was doing the work. The cap is gone.

The lesson generalises past this file: a pair that wins together says nothing
about either half. Split it.

## What worked: letting the pipes go while he is still turning

Explosives carry a second fire gate that no hitscan weapon does. `IsHeadSteady`
requires the aim to have stopped moving, and it is invalidated by any eye
movement above a small per-frame rate, so a bot following a walking robot rarely
qualifies.

Lifting it, twelve waves on two maps:

```
                       demoman dmg (shots, acc)    soldier dmg (shots, acc)
ON  (fire tracking)      19754 (409, 46%)            9602 (296, 34%)
OFF (wait for steady)    17905 (384, 43%)           14023 (313, 48%)
```

**The two weapons want opposite answers.** The Demoman gained on both maps
independently and his hit rate went *up*. The Soldier lost a third of his
damage with his hit rate falling 48% to 34%.

A pipe arcs, bursts wide and is usually thrown at a group, so one released
mid-turn still lands somewhere worth landing. A rocket is fast and flat:
released mid-turn it arrives where the head was pointing and hits nothing.

So the gate is lifted for the grenade and sticky launchers and kept for
rockets. Not a feature switch — the weapon decides.

One hypothesis died here and is worth recording: this was expected to raise
their *rate* of fire, and it did not. Shots went 384 to 409 for the Demoman and
313 to 296 for the Soldier. `IsHeadSteady` was never the throttle. A Demoman
firing forty five pipes in a wave from a launcher that reloads in six tenths of
a second is still unexplained.

## What worked for the soldier: the stock launcher

The loadout handed him the Beggar's Bazooka. Ten waves across Decoy and
Bigrock, against the stock rocket launcher:

```
                    stock          Beggar's
Decoy  damage       17071            11411
Bigrock damage      15211            14977
combined            32282            26388     +22%

hit rate              61%              46%
damage per rocket   62 / 68          43 / 50
self-damage        1171/1517        2698/1909
```

**A bot cannot aim off a spread.** A person loads three, releases, and walks
the spread onto the target; a bot aims at a point and eats every deviation.
Damage per rocket is 36% higher with stock on both maps and the hit-rate gap is
the same on both, which is what a mechanical cause looks like as opposed to map
luck.

Note Bigrock's totals nearly tie, because the Beggar's burst fires more rockets
there (302 against 225) and volume almost covers for accuracy. Decoy's +50% on
its own would have overstated this; the two maps together give +22%.

The general lesson for `configs/defenderbots/loadout.cfg`: **a weapon chosen
from a human guide can be actively wrong for a bot.** The comment on that entry
said exactly why it was picked, and the reasoning was sound for a person.

## Still open

The Soldier's rocket launcher has no entry in `weapon_tuning.sp` and falls
through to a 1250-unit desired range, while every comparable explosive in the
table sits at 600–650 (Loose Cannon 650, Iron Bomber 600, Beggar's 600). A
rocket takes over a second to cross 1250 units and the blast covers 146 of
them. `soldier_closes_in` (750) is written and not yet measured.

Whether it matters depends on the item below, because a desired range only
means something if he can act on it.

## What did work, and what it cost

`attack_path_nudge`: when the mesh refuses a path, step toward the target
instead of standing still. Twelve waves across two maps, each way:

```
                  ON      OFF
demoman damage   20592   16778    +23%
team damage     221807  222822    flat
defender deaths    130      95    +37%
waves cleared        7       8
```

Both effects replicated on both maps, so this is a trade rather than noise. It
is the only change measured that raised the two classes' damage, and it did not
raise the team's.

**A fighter is not a medic.** The identical fix inside
`PluginBot_SimulateFrame` took the medic from 4% of a wave with his beam
connected to 30%, because a path that fails on the way to a teammate is a bot
standing still for nothing. Arriving next to a robot is not the same as
arriving next to a friend, and ground the mesh will not path is often ground
worth not standing on.

So the nudge stays where it wins and is gone from `attack.sp` and
`campbomb.sp`. The failure *counting* stays everywhere: `sm_dump_front` reports
dead-end paths per bot, which is how the 1044-versus-600 gap was found in the
first place.

## The path bug reaches both of them

`ComputeToTarget` returns a bool and every one of the mod's twenty-one call
sites discarded it. An empty path walks the bot nowhere while the behaviour
above believes it is travelling. It was fixed in `PluginBot_SimulateFrame`
first and nowhere else, which left `attack.sp` — the action every fighting
class spends 40–56% of its samples in — still holding bots at whatever range
the mesh happened to refuse at.

That is the likeliest reason the Demoman sits at 1044 units while asking to be
at 600. `attack_path_nudge` covers it and is not yet measured.
