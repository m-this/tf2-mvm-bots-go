# Recover a managed defender that leaves RED

## Report

On `mvm_null_b9c`, one managed defender no longer counted on RED during an MvM
wave. The manager then reported that RED held five of six and requested one
replacement every second. Every request failed because the misplaced defender
still occupied the final client slot.

Purging the managed defenders recovered the game immediately: the existing fill
path recreated the lineup and every replacement joined RED.

## Cause

The manager marks each defender client after it joins RED. Its `player_team`
handler previously ignored every fake client, including an already-marked
defender that moved away from RED. That left two facts which could not converge:

- RED was one defender short, so the imbalance timer requested a replacement;
- the misplaced defender still occupied a client slot, so a full server could
  not create that replacement.

## Invariant and recovery

An identified managed defender belongs on RED for the rest of its connection.
If a non-disconnect team event moves it from RED to another team, remove that
client and its buildings. The existing imbalance timer then owns the recovery:
it sees the empty RED seat and creates one replacement through the normal join,
loadout and currency path.

The recovery is deliberately narrower than the successful manual purge. It
does not remove healthy defenders, ordinary BLU robots or humans, and it ignores
the team event emitted by an intentional disconnect.

## Verification

The Go test exercises the recovery and each exclusion directly. A live
verification still has to force one identified defender from RED during a
managed wave and confirm that its replacement joins RED and requests stop once
the team is full.
