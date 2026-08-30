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
