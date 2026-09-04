/*
Package stocks is what is left of source/redbots3/util.sp once the decisions have
moved out: the short answers everything else is written in terms of.
*/
package stocks

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// DoesAnyPlayerUseThisName says somebody on the server is already called that,
// which is how a bot avoids taking a player's name.
//
//sp:name DoesAnyPlayerUseThisName
func DoesAnyPlayerUseThisName(name string) bool {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientConnected(i) {
			continue
		}

		named, playerName := engine.ClientName(i)

		if named && engine.StrEqualFolded(playerName, name, false) {
			return true
		}
	}

	return false
}

// NormalizeAngle folds an angle into the half turn either side of zero, which is
// the shortest way round.
//
//sp:name NormalizeAngle
func NormalizeAngle(angle float32) float32 {
	//nolint:gocritic // assignOp: the shipped file writes the subtraction out, and the port keeps its shape
	angle = angle - float32(engine.RoundToFloor(angle/360.0))*360.0

	if angle > 180.0 {
		angle -= 360.0
	} else if angle < -180.0 {
		angle += 360.0
	}

	return angle
}

// SnapViewToPosition turns the bot's head to look at a point, at once rather than
// over time.
//
//sp:name SnapViewToPosition
//sp:const pos
func SnapViewToPosition(client int32, pos [3]float32) {
	clientEyePos := engine.ClientEyePosition(client)

	desiredDir := engine.MakeVectorFromPoints(clientEyePos, pos)
	desiredDir = engine.VectorAngles(desiredDir)

	clientEyeAng := engine.ClientEyeAngles(client)

	var eyeAngles [3]float32

	eyeAngles[0] = clientEyeAng[0] + NormalizeAngle(desiredDir[0]-clientEyeAng[0])
	eyeAngles[1] = clientEyeAng[1] + NormalizeAngle(desiredDir[1]-clientEyeAng[1])
	eyeAngles[2] = 0.0

	engine.TeleportEntity(client, engine.NullVector(), eyeAngles, engine.NullVector())
}

// IsValidClientIndex says the number is a client that is actually here.
//
//sp:name IsValidClientIndex
func IsValidClientIndex(client int32) bool {
	return client > 0 && client <= engine.MaxClients() && engine.IsClientInGame(client)
}

// IsBaseBoss says the entity is a tank, which is the one thing carrying that
// property.
//
//sp:name IsBaseBoss
func IsBaseBoss(entity int32) bool {
	return engine.HasEntProp(entity, engine.PropSend(), "m_lastHealthPercentage")
}

// IsPlayerReady says the player has pressed ready, which is what holds a wave.
//
//sp:name IsPlayerReady
func IsPlayerReady(client int32) bool {
	return engine.GameRulesPropAt("m_bPlayerReady", 1, client) != 0
}

// IsMeleeWeapon says the weapon swings rather than fires.
//
//sp:name IsMeleeWeapon
func IsMeleeWeapon(entity int32) bool {
	// THINKFUNC Smack
	return engine.HasEntProp(entity, engine.PropData(), "CTFWeaponBaseMeleeSmack")
}

// IsZeroVector says the position is the origin, which is what an unset one reads
// as.
//
//sp:name IsZeroVector
func IsZeroVector(origin [3]float32) bool {
	return origin[0] == engine.NullVector()[0] && origin[1] == engine.NullVector()[1] && origin[2] == engine.NullVector()[2]
}

// SetPlayerReady presses ready for the bot, when it is not already pressed.
//
//sp:name SetPlayerReady
func SetPlayerReady(client int32, state bool) {
	if IsPlayerReady(client) == state {
		return
	}

	engine.FakeClientCommand(client, "tournament_player_readystate %d", state)
}

// IsPluginMvMCreditsLoaded says the credits plugin is on this server.
//
//sp:name IsPluginMvMCreditsLoaded
func IsPluginMvMCreditsLoaded() bool {
	// tf_mvm_credits
	return engine.FindConVar("sm_mvmcredits_version") != engine.NoConVar()
}

// IsPluginRTDLoaded says the roll-the-dice plugin is on this server.
//
//sp:name IsPluginRTDLoaded
func IsPluginRTDLoaded() bool {
	// rtd
	return engine.FindConVar("sm_rtd2_version") != engine.NoConVar()
}

// UseActionSlotItem uses the canteen, which the game takes as key values rather
// than as a command.
//
//sp:name UseActionSlotItem
func UseActionSlotItem(client int32) {
	kv := engine.NewKeyValues("use_action_slot_item_server")
	defer kv.Close()

	engine.FakeCommandKV(client, kv)
}

// PlayerBuyback spends the credits that put a dead bot back in the wave.
//
//sp:name PlayerBuyback
func PlayerBuyback(client int32) {
	engine.FakeClientCommand(client, "td_buyback")
}

// IsServerFull says there is no seat left.
//
//sp:name IsServerFull
func IsServerFull() bool {
	return engine.ClientCount(false) >= engine.MaxClients()
}

// GetTeamHumanClientCount is how many people, rather than bots, are on that team.
//
//sp:name GetTeamHumanClientCount
func GetTeamHumanClientCount(team int32) int32 {
	count := int32(0)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && !engine.IsFakeClient(i) && engine.GetClientTeam(i) == team {
			count++
		}
	}

	return count
}

// TEMPGetPlayerMaxHealth is the player's maximum, read off the resource entity
// because the player's own property lags a class change. The shipped name says
// TEMP_ and has for years.
//
//sp:name TEMP_GetPlayerMaxHealth
func TEMPGetPlayerMaxHealth(client int32) int32 {
	return engine.EntPropAt(engine.ResourceEntity(), engine.PropSend(), "m_iMaxHealth", 4, client)
}
