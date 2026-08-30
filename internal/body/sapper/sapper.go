/*
Package sapper is what is left of source/redbots3/util.sp's entity work: putting a
sapper on something, finding the capture trigger, and the two reads that follow a
raw address.
*/
package sapper

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
ParentEntity hangs one entity off another.

The shipped declaration says bool and the body never returns one, so nothing can
read the answer and nothing does: the port says void, which is what it is.
*/
//
//sp:name ParentEntity
//sp:default attachPoint ""
//sp:default maintainOffset false
func ParentEntity(parent int32, attachment int32, attachPoint string, maintainOffset bool) {
	engine.SetVariantString("!activator")
	engine.AcceptEntityInput(attachment, "SetParent", parent, attachment, 0)

	if engine.TextLengthOf(attachPoint) > 0 {
		engine.SetVariantString(attachPoint)
		engine.AcceptEntityInput(attachment,
			engine.Choose(maintainOffset, "SetParentAttachmentMaintainOffset", "SetParentAttachment"),
			parent, parent, 0)
	}
}

// SpawnSapper puts one on a building or a player, which is how a spy's sapper is
// made to exist without him placing it.
//
//sp:name SpawnSapper
//sp:default weapon -1
func SpawnSapper(owner int32, entity int32, weapon int32) int32 {
	sapper := engine.CreateEntityByName("obj_attachment_sapper")

	if sapper != -1 {
		engine.AcceptEntityInput(sapper, "SetBuilder", owner, sapper, 0)

		if weapon > 0 {
			engine.SetObjectMode(sapper, engine.EntProp(weapon, engine.PropSend(), "m_iObjectMode"))
		}

		ParentEntity(entity, sapper, engine.Choose(engine.IsPlayer(entity), "head", "weapon_bone"), false)
		engine.SetEntPropEnt(sapper, engine.PropSend(), "m_hBuiltOnEntity", entity)
		engine.SetEntPropSend(sapper, engine.PropSend(), "m_bBuilding", 1)
		engine.DispatchSpawn(sapper)
		engine.RemoveEffectsFrom(sapper, engine.EffectNoDraw())
	}

	return sapper
}

// GetCapturableAreaTrigger is the capture trigger this team may still take, and
// -1 when there is none.
//
//sp:name GetCapturableAreaTrigger
func GetCapturableAreaTrigger(team engine.Team) int32 {
	trigger := int32(-1)

	for {
		trigger = engine.FindEntityByClassname(trigger, "trigger_*")

		if trigger == -1 {
			break
		}

		// Only want capture areas
		if !engine.HasEntProp(trigger, engine.PropData(), "CTriggerAreaCaptureCaptureThink") {
			continue
		}

		// Ignore disabled triggers
		if engine.EntProp(trigger, engine.PropData(), "m_bDisabled") != 0 {
			continue
		}

		// Apparently some community maps don't disable the trigger when capped
		capPointName := engine.EntPropString(trigger, engine.PropData(), "m_iszCapPointName")

		// Trigger has no point associated with it
		if engine.TextLength(capPointName) < 3 {
			continue
		}

		// Now find the matching control point
		point := int32(-1)

		for {
			point = engine.FindEntityByClassname(point, "team_control_point")

			if point == -1 {
				break
			}

			pointIndex := engine.EntProp(point, engine.PropData(), "m_iPointIndex")

			if !engine.TeamMayCapturePoint(team, pointIndex) {
				continue
			}

			name := engine.EntPropString(point, engine.PropData(), "m_iName")

			if engine.CompareText(name, capPointName, false) == 0 {
				return trigger
			}
		}
	}

	return -1
}

/*
IsUpgradeStationEnabled says the station is switched on.

By offset, because m_bIsEnabled has no name in the datamap: it sits a fixed
distance after one that does, and the offset is found once and kept.
*/
//
//sp:name IsUpgradeStationEnabled
func IsUpgradeStationEnabled(station int32) bool {
	// m_bIsEnabled
	if offsetIsEnabled == -1 {
		offsetIsEnabled = engine.FindDataMapInfo(station, "m_nStartDisabled") + 28
	}

	return engine.EntData(station, offsetIsEnabled, 1) != 0
}

// The offset found once and kept, which the shipped file holds in a static local.
//
//sp:name iOffsetIsEnabled
var offsetIsEnabled int32 = -1

// DereferencePointer follows a pointer one step.
//
//sp:name DereferencePointer
func DereferencePointer(addr engine.Address) engine.Address {
	// maybe someday we'll do 64-bit addresses
	return engine.Address(engine.LoadFromAddress(addr))
}

// ReadInt is the same read, guarded, which is what every caller wants.
//
//sp:name ReadInt
func ReadInt(addr engine.Address) int32 {
	if addr == engine.Address(engine.AddressNull()) {
		return -1
	}

	return engine.LoadFromAddress(addr)
}
