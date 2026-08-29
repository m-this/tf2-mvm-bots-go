/*
Package roster is the first body generated with engine calls in it.

Three shapes, one per kind mvm-bis asks for: a scan calling natives, a weapon
question through an SDKCall, and a DHook callback. The scan is the first of the
nine copies of one client loop that util.sp holds, which is mvm-z83.35.
*/
package roster

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

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

// MyTouchPre is CTFPowerup::MyTouch, hooked so a defender bot picking up credits
// is not counted as a bot while it does. dhooks.sp sets a plugin global here;
// the global is still SourcePawn's, so it is reached the way an engine call is.
//
//sp:dhook
func MyTouchPre(powerup int32, player int32) { //nolint:revive // the first parameter of a DHook is the hooked object, read or not
	if engine.IsDefenderBot(player) {
		engine.SetTouchCredits(true)
	}
}

// IsBotPre is CTFPlayer::IsBot. A defender bot in the middle of a credit touch
// answers false, so the game's own money code treats it as a player.
//
//sp:dhook
func IsBotPre(client int32, touchingCredits bool) (supercede bool, value bool) {
	if engine.IsClientInGame(client) && engine.IsDefenderBot(client) && touchingCredits {
		return true, false
	}
	return false, false
}
