# Custom bot loadouts, mod side

Draft. The launcher side is `docs/custom-loadouts.md` in tf2-archipelago, and
it holds most of the work. This file is what is left over here.

## The ask

From the Discord thread, 2026-08-25 (Cowser):

> instead of Scorch Shot for the Phlog build, it's the Gas Passer. Or instead of
> Kunai, it's Big Earner.

Plus three behaviour suggestions in the same thread:

> - Engineers always prioritize replacing the Dispenser when it is broken, and
>   can move spots based on the Bomb's location, and will reset to the base
>   location between waves
> - Pyros with Gas Passer prioritize Explode on Ignite
> - Snipers prioritize Explosive Headshot

## The weapon half needs no change here

`GetServerLoadoutWeapon` (`source/redbots3/player_pref.sp:133`) reads the slot
straight out of `configs/defenderbots/loadout.cfg` and hands the number to
`PrepareCustomLoadout` (`source/redbots3/loadouts.sp:257`). Nothing checks it
against the `WEAPONS_*` arrays. Those arrays are the pool
`GetRandomWeaponForClass` draws from, and only that.

So the launcher can write any item definition index today and the bot will
hold it. The whole feature is a Go menu.

Two things that already work and should not be broken by the churn:

- The seat half of the file is implemented. `JumpToServerLoadoutSeat`
  (`player_pref.sp:105`) reads `seats/N/class` and falls back to the class
  block when the seat does not name this bot. The comment on `Render` in
  `launcher/internal/botloadout/botloadout.go` still says the mod has not
  caught up. It has. Fix that comment.
- `PrepareCustomLoadout` sets `CTFBot_MISSION_SNIPER` off the primary's item
  class name, so a custom Sniper primary still gets sniper AI. Any new weapon
  the launcher offers goes through the same path.

## What actually needs work here

Weapons the launcher can already name and the AI does not drive:

| Weapon | What the bot does not do |
| --- | --- |
| Beggar's Bazooka (730) | Charge three rockets, then fire. Already a preset. |
| Huntsman (56) | Draw and hold. Untested; may fall through to hitscan aim. |
| Gas Passer (1180) | Throw it at a group. Sits unused. |
| Phlogistinator (594) | Spend Mmmph. |
| Cow Mangler (441) | Charged shot against a tank. |
| Banners (129, 226, 354) | Deploy when the meter is full. |
| Jarate / Mad Milk (58, 222) | Throw. Meter reading exists (`nextbot_behavior.sp:2171`); the throw does not. |

That table is the reason the launcher wants a support list. Offering the Gas
Passer in a menu and then having the bot never throw it is worse than not
offering it, because it reads as a broken server rather than a missing feature.

Proposal: `docs/weapon-support.md`, one row per non-stock item definition index
the mod can hand out, with a state of `driven`, `held` or `untested`. Generated
by hand, kept honest by the testbed. The launcher reads it as data, or copies
it; either way the mod repo owns it, because the mod is where the answer lives.

Start it with what is already known rather than auditing all thirty arrays:
every index named in a launcher preset, plus the seven above.

## The three behaviour suggestions

**Engineer, dispenser first when it is broken.** Not missing, mis-ranked. The
rebuild is gated on the sentry being safe: `engineeridle.sp:519` only suspends
into `CTFBotMvMEngineerBuildDispenser` when `m_ctSentrySafe` is in the future.
An engineer under pressure with a dead dispenser therefore builds nothing. The
change is a rank, and a rank is an A/B: run it through the testbed and read
`docs/testbed-metrics.md` for which numbers move. `docs/engineer-and-medic.md`
already carries two engineer fixes that looked obviously right and lost.

**Engineer, move by the bomb and reset between waves.** Both exist.
`EngineerNestRelocation_*` (`engineeridle.sp:1081`) evaluates during a wave and
`events.sp:42` and `events.sp:93` reset it. What may not exist is *reset to the
base spot*, as opposed to reset the evaluation. Check `_ResetAll` against
`bot_hint_engineer_nest` before treating this as new work.

**Sniper, prioritise Explosive Headshot.** Done. `behavior/upgrade.sp:794`
ranks `explosive sniper shot` at 330, above reload speed, and the comment
explains why charge rate was demoted. Tell Cowser it is in.

**Pyro, Gas Passer explode on ignite.** This one is real and it is two pieces,
in this order:

1. Throw the gas. No behaviour throws it today. Nearest model is the sticky
   trap (`behavior/stickytrap.sp`): pick a spread the wave walks through,
   commit to it, do not re-aim every frame.
2. Buy the upgrade. `behavior/upgrade.sp` ranks by attribute name, so this is
   a priority entry keyed on the Gas Passer's item definition index, the same
   shape as the Machina case at `upgrade.sp:664`.

Do not do 2 without 1. A bot that buys explode-on-ignite and never throws the
gas has spent money on nothing, which is a worse outcome than the bot ignoring
the weapon.

## Testing

Everything above is a testbed run, not a judgement. `testbed/run.sh` plays a
mission and `go run ./testbed/report` compares two. The Gas Passer work needs a
mission with tight approaches; the dispenser rank needs one where the engineer
is under pressure, which is what made it a report in the first place.

Read `docs/how-bots-break.md` before touching the behaviour tree. The faults in
this mod have one shape.

## Order of work

1. Fix the stale `Render` comment in the launcher. One line, and it is
   currently telling the next reader something false.
2. `docs/weapon-support.md`, seeded from the launcher presets.
3. Gas Passer throw, then its upgrade priority.
4. Dispenser rank A/B.
5. Check whether nest reset returns to the base spot.

The launcher can ship its menu after step 2 and before any of the rest.
