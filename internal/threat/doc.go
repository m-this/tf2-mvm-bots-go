/*
Package threat is what a robot is worth shooting first, as a function of a
record rather than of an entity index.

It is a port of ThreatPriority in source/redbots3/nextbot_behavior.sp, and a
port changes nothing. Priority is the shipped chain branch for branch, including
the two places it answers PriorityNone for something a player would call a
threat.

The record is the point, and it is the acceptance criterion on mvm-z83.6. The
shipped function takes an entity index and asks the engine six questions about
it, and every threat scan in the plugin walks player slots to find one. A tank
occupies no player slot, so those scans never see it, which is mvm-ds3. A
decision that takes a filled record cannot have that bug: whoever fills the
record decides what a threat is, and the decision decides what it is worth.

Unlike internal/actionsel, none of the questions here costs anything. They are
six reads with no side effect and no randomness, so the record can be filled
eagerly and the edge needs no lazy walk. That is why mvm-z83.40 does not apply
to this one, and it is worth saying rather than leaving as an absence.

The range is kept as a squared distance in float32, the way the plugin holds it,
so the two comparisons that decide urgent and out-of-range are the comparisons
the plugin makes rather than ones a wider type would answer differently.
*/
package threat
