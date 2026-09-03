/*
Package offsets is source/redbots3/offsets.sp: reading the fields the game has
and SourceMod does not name.

An offset comes out of the gamedata file, sometimes as a plain number and
sometimes as a number past a send prop the game does name. Either way it is
looked up once at load and read out of a map afterwards, because
FindSendPropInfo is not something to do per frame.

The offset arithmetic follows Mikusch's MannVsMann, which is where the base
offset idea comes from:
https://github.com/Mikusch/MannVsMann/blob/571737b5ae0aadc1e743360e94311ca64e693bd9/addons/sourcemod/scripting/mannvsmann/offsets.sp
*/
package offsets

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

//sp:name m_adtOffsets
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var offsetMap engine.Properties

// InitOffsets reads every offset the mod needs, once.
//
//sp:name InitOffsets
func InitOffsets(hGamedata engine.GameData) {
	engine.SetOffsetMap(engine.NewProperties())

	SetOffset(hGamedata, "CTFPlayer", "m_LastDamageType")
	SetOffset(hGamedata, "CObjectSentrygun", "m_bPlacementOK")
	SetOffset(hGamedata, "CObjectSentrygun", "m_vecCurAngles")
	SetOffset(hGamedata, "CTFBot", "m_isLookingAroundForEnemies")
	SetOffset(hGamedata, "CTFBot", "m_mission")
	SetOffset(hGamedata, "CTFBot", "m_opportunisticTimer")
	SetOffset(hGamedata, "CPopulationManager", "m_nStartingCurrency")
	SetOffset(hGamedata, "CTFBuffItem", "m_bPlayingHorn")
	SetOffset(hGamedata, "CTFRevolver", "m_flLastAccuracyCheck")
	SetOffset(hGamedata, "CTFNavArea", "m_distanceToBombTarget")
}

/*
	SetOffset works out one offset and remembers it

A class whose gamedata names a base prop is read as that prop's offset plus the
one the file gives, because the field sits inside a structure the game does name.
CTFBot is looked up on CTFPlayer: the game declares the send table there.
*/
//
//sp:name SetOffset
func SetOffset(hGamedata engine.GameData, cls string, prop string) {
	key := engine.FormatInto("%s::%s", cls, prop)
	baseKey := engine.FormatInto("%s_BaseOffset", cls)

	// The actual offset, calculated using a base offset if present.
	found, baseProp := hGamedata.KeyValue(baseKey)

	if found {
		baseOffset := engine.FindSendPropInfoText(cls, baseProp)

		if engine.StrEqualLiterals(cls, "CTFBot") {
			baseOffset = engine.FindSendPropInfoText("CTFPlayer", baseProp)
		}

		if baseOffset == -1 {
			// Nothing found, so search on CBaseEntity instead.
			baseOffset = engine.FindSendPropInfoText("CBaseEntity", baseProp)

			if baseOffset == -1 {
				engine.ThrowErrorText("Base offset '%s::%s' could not be found", cls, baseProp)
			}
		}

		offset := baseOffset + hGamedata.Offset(key)
		engine.OffsetMap().SetPropertyAt(key, offset)
	} else {
		offset := hGamedata.Offset(key)

		if offset == -1 {
			engine.ThrowErrorText("Offset '%s' could not be found", key)
		}

		engine.OffsetMap().SetPropertyAt(key, offset)
	}
}

// GetOffset is the offset that was worked out at load.
//
//sp:name GetOffset
func GetOffset(cls string, prop string) int32 {
	key := engine.FormatInto("%s::%s", cls, prop)

	found, offset := engine.OffsetMap().ValueAt(key)
	if !found {
		engine.ThrowErrorText("Offset '%s' not present in map", key)
	}

	return offset
}

// GetLastDamageType is how the player was last hurt.
//
//sp:name GetLastDamageType
func GetLastDamageType(client int32) int32 {
	return engine.EntDataDefault(client, GetOffset("CTFPlayer", "m_LastDamageType"))
}

// IsPlacementOK says the sentry blueprint is somewhere it could be built.
//
//sp:name IsPlacementOK
func IsPlacementOK(iObject int32) bool {
	return engine.EntDataSized(iObject, GetOffset("CObjectSentrygun", "m_bPlacementOK"), 1) != 0
}

// GetTurretAngles is where the sentry is currently pointing.
//
//sp:name GetTurretAngles
func GetTurretAngles(sentry int32) (buffer [3]float32) {
	buffer = engine.EntDataVector(sentry, GetOffset("CObjectSentrygun", "m_vecCurAngles"))

	return buffer
}

/*
	SetLookingAroundForEnemies tells the game's own bot code to stop scanning

Guarded, because an action's OnEnd runs after its actor may already be gone. The
server hibernates at the end of a mission and punts every bot on the way, and
the engineer's build actions end after that: three exceptions a map, all of them
this, writing into an entity index that is nobody.
*/
//
//sp:name SetLookingAroundForEnemies
func SetLookingAroundForEnemies(client int32, shouldLook bool) {
	if client < 1 || client > engine.MaxClients() || !engine.IsClientInGame(client) {
		return
	}

	engine.SetEntDataSized(client, GetOffset("CTFBot", "m_isLookingAroundForEnemies"), engine.CellOfBool(shouldLook), 1)
}

// GetTFBotMission is what the game's own bot code was told to do.
//
//sp:name GetTFBotMission
func GetTFBotMission(client int32) int32 {
	return engine.EntDataDefault(client, GetOffset("CTFBot", "m_mission"))
}

// GetOpportunisticTimer is the timer the game uses to decide when a bot may
// take a free shot at something it passed.
//
//sp:name GetOpportunisticTimer
func GetOpportunisticTimer(client int32) engine.Address {
	return engine.EntityAddress(client) + engine.Address(GetOffset("CTFBot", "m_opportunisticTimer"))
}

// GetStartingCurrency is what the mission handed out before the first wave. The
// real figure is two variables and the other one has never mattered.
//
//sp:name GetStartingCurrency
func GetStartingCurrency(populator int32) int32 {
	return engine.EntDataDefault(populator, GetOffset("CPopulationManager", "m_nStartingCurrency"))
}

// IsPlayingHorn says the buff banner is mid-blow.
//
//sp:name IsPlayingHorn
func IsPlayingHorn(weapon int32) bool {
	return engine.EntDataSized(weapon, GetOffset("CTFBuffItem", "m_bPlayingHorn"), 1) != 0
}

// GetLastAccuracyCheck is when the revolver last decided its shot was accurate.
//
//sp:name GetLastAccuracyCheck
func GetLastAccuracyCheck(weapon int32) float32 {
	return engine.EntDataFloat(weapon, GetOffset("CTFRevolver", "m_flLastAccuracyCheck"))
}

// GetTravelDistanceToBombTarget is how far this area is from the hatch along
// the path the robots walk, which is the number the whole nest search is scored
// on.
//
//sp:name GetTravelDistanceToBombTarget
func GetTravelDistanceToBombTarget(area engine.NavArea) float32 {
	return engine.LoadFloatFromAddress(engine.AddressOfArea(area) + engine.Address(GetOffset("CTFNavArea", "m_distanceToBombTarget")))
}
