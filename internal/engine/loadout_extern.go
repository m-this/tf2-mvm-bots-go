package engine

/*
What the custom loadout reaches: the preference lookup, the mission, and the
attributes an upgrade left on a weapon.
*/

// LoadoutCalls are the answers.
type LoadoutCalls struct {
	PreferredWeaponForClass func(class string, slot string, client int32) int32
	SetMission              func(client int32, mission int32)
	AttribByDefIndex        func(entity int32, defIndex int32) Address
	AttribValueAt           func(attrib Address) float32
}

var loadouts LoadoutCalls

// InstallLoadouts puts a set of answers behind them.
func InstallLoadouts(c LoadoutCalls) func() {
	previous := loadouts
	loadouts = c
	return func() { loadouts = previous }
}

// PreferredWeaponForClass is what this player asked for in that slot, or a draw
// from the pool when they asked for nothing.
//
//sp:plugin GetPreferredWeaponForClass
func PreferredWeaponForClass(class string, slot string, client int32) int32 {
	if loadouts.PreferredWeaponForClass == nil {
		missing("GetPreferredWeaponForClass")
	}
	return loadouts.PreferredWeaponForClass(class, slot, client)
}

// SetMission tells the game what this bot is for, which is what makes a sniper
// lurk rather than stand where he shopped.
//
//sp:plugin SetMission
func SetMission(client int32, mission int32) {
	if loadouts.SetMission == nil {
		missing("SetMission")
	}
	loadouts.SetMission(client, mission)
}

// AttribByDefIndex is the address of one attribute on an entity.
//
//sp:native TF2Attrib_GetByDefIndex
func AttribByDefIndex(entity int32, defIndex int32) Address {
	if loadouts.AttribByDefIndex == nil {
		missing("TF2Attrib_GetByDefIndex")
	}
	return loadouts.AttribByDefIndex(entity, defIndex)
}

// AttribValueAt reads one through its address, which is what the attribute list
// hands back.
//
//sp:native TF2Attrib_GetValue
func AttribValueAt(attrib Address) float32 {
	if loadouts.AttribValueAt == nil {
		missing("TF2Attrib_GetValue")
	}
	return loadouts.AttribValueAt(attrib)
}
