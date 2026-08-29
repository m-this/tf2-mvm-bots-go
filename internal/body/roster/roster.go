/*
Package roster is the first body generated with engine calls in it.

Four shapes, one per kind the port needs: a scan calling natives, a weapon
question through an SDKCall, a DHook callback, and the plugin state two of them
read. The scan is the first of the nine copies of one client loop that util.sp
holds, which is mvm-z83.35.
*/
package roster

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1. Slot 0 is never a client and
// is there so the array indexes the way SourcePawn's do.
const Slots = 65

// defenderBot is dhooks.sp's g_bIsDefenderBot: which clients are ours.
var defenderBot [Slots]bool

// touchingCredits is dhooks.sp's m_bTouchCredits, true while a defender bot is
// inside CTFPowerup::MyTouch and the game's money code must not see a bot.
var touchingCredits bool

// ResetState puts the plugin state back to what it is at load. A global that is
// per map and never put back leaves the next map with the last one's answers.
func ResetState() {
	for i := range defenderBot {
		defenderBot[i] = false
	}
	touchingCredits = false
}

// SetDefenderBot records that a client is one of ours, which the plugin does
// when it takes a slot over.
func SetDefenderBot(client int32, ours bool) {
	defenderBot[client] = ours
}

// AliveOnTeam counts the clients in the game and alive on team, over the slots
// 1..maxClients. It is util.sp's loop with the predicate left out: the callers
// that filter further pass their own count of what they found.
func AliveOnTeam(maxClients int32, team int32) int32 {
	count := int32(0)
	for i := int32(1); i <= maxClients; i++ {
		if !engine.IsClientInGame(i) {
			continue
		}
		if !engine.IsPlayerAlive(i) {
			continue
		}
		if engine.GetClientTeam(i) != team {
			continue
		}
		count++
	}
	return count
}

// LoadedRounds is what the weapon has to fire with. A weapon with no ammo at
// all reads as empty rather than as whatever the clip last held.
func LoadedRounds(weapon int32) int32 {
	if !engine.HasAmmo(weapon) {
		return 0
	}
	clip := engine.Clip1(weapon)
	if clip < 0 {
		return 0
	}
	return clip
}

// MyTouchPre is CTFPowerup::MyTouch, hooked so a defender bot picking up
// credits is not counted as a bot while it does.
//
//sp:dhook
func MyTouchPre(powerup int32, player int32) { //nolint:revive // the first parameter of a DHook is the hooked object, read or not
	if defenderBot[player] {
		touchingCredits = true
	}
}

// MyTouchPost puts it back. The pair is the whole point: a hook that sets a
// flag and never clears it leaves every defender bot invisible to the money
// code for the rest of the map.
//
//sp:dhook
func MyTouchPost(powerup int32, player int32) { //nolint:revive // the first parameter of a DHook is the hooked object, read or not
	if defenderBot[player] {
		touchingCredits = false
	}
}

// IsBotPre is CTFPlayer::IsBot. A defender bot in the middle of a credit touch
// answers false, so the game's own money code treats it as a player.
//
//sp:dhook
func IsBotPre(client int32) (supercede bool, value bool) {
	if engine.IsClientInGame(client) && defenderBot[client] && touchingCredits {
		return true, false
	}
	return false, false
}

// TeamCentre is where a team is on average, which the nest and the guard point
// both want and which util.sp works out inline in three places. An empty team
// has no centre, and the zero vector is what the shipped code leaves in the
// caller's array.
func TeamCentre(maxClients int32, team int32) (centre [3]float32) {
	found := int32(0)
	for i := int32(1); i <= maxClients; i++ {
		if !engine.IsClientInGame(i) {
			continue
		}
		if engine.GetClientTeam(i) != team {
			continue
		}
		origin := engine.Origin(i)
		for axis := range centre {
			centre[axis] += origin[axis]
		}
		found++
	}
	if found == 0 {
		return centre
	}
	for axis := range centre {
		centre[axis] /= float32(found)
	}
	return centre
}

/*
Where Go and SourcePawn disagree about precedence.

Measured, not assumed. With a, b, c = 3, 5, 2, `a + b << c` is 23 in Go and 32
under spcomp, and `a | b ^ c` is 5 and 7: spcomp puts the shift below the sum
and the xor above the or, and Go does neither. It also binds & tighter than ==,
which C does not, so guessing the rules from C gets it wrong in both directions.

Both of those compile in either language and answer differently. The generator
parenthesises an operand whose operator differs from its parent's, and these are
here so that a change to that rule is a failing test rather than a plugin that
quietly does something else.

They are in the probe rather than in a golden file because a golden says the
text did not move and this says the two languages answer the same.
*/

// Shifted is the sum against the shift, which is one of the two that differ.
func Shifted(a int32, b int32, c int32) int32 {
	return a + b<<c
}

// Ored is the or against the xor, which is the other.
func Ored(a int32, b int32, c int32) int32 {
	return a | b ^ c
}

// Mixed walks the pairs that happen to agree, so the rule is not only tested
// where it changes something.
func Mixed(a int32, b int32, c int32) int32 {
	masked := a&b | c
	xored := a ^ b&c
	summed := a + b*c
	return masked + xored + summed
}

// Chained is one operator throughout, which groups the same way in both and so
// needs no parentheses at all.
func Chained(a int32, b int32, c int32) int32 {
	return a + b + c
}
