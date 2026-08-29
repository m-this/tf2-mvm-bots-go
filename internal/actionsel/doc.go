// Package actionsel is the choice of which behaviour a defender bot is handed,
// written as a total function over the inputs the choice actually reads.
//
// It is a port of GetDesiredBotAction and ShouldTakeUpPosition in
// source/redbots3/nextbot_behavior.sp, and a port changes nothing. Select is
// the shipped chain branch for branch, including the combinations where the
// plugin hands the bot back to the game with no behaviour: those are named
// outcomes rather than a fallthrough, so "nothing happens here" is a decision
// with a name, but the plugin does exactly what it does today.
//
// The one place the shipped code strands a bot, mvm-vnn, stays stranded here:
// ActionStrandedAsShipped. The fill for it lives in SelectFilled, which is a
// separate function nothing generates into the plugin's table, so the fix can
// be measured on its own instead of riding along with the port.
//
// An outcome is one call site of the shipped function, not one action, because
// the SuspendFor reason string is surfaced in debug output and read by the
// test-bed. CTFBotDefenderAttack with "Scout: Attacking robots" and
// CTFBotDefenderAttack with "CTFBotAttack_IsPossible" are two outcomes.
//
// The whole package is inside the internal/gosubset subset, checked by the
// test, because internal/spgen generates the SourcePawn it replaces.
package actionsel
