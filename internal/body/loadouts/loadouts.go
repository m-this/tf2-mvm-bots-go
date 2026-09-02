package loadouts

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Whether the bot in this seat is carrying what the loadout gave it.
//
//sp:name g_bHasCustomLoadout
var hasCustomLoadout [Slots]bool

// Read by the command that stays in the plugin, which is why nothing here
// touches it.
//
//sp:name g_bHasBoughtUpgrades
//nolint:unused // the shipped file leaves it to Command_BoughtUpgrades as well
var hasBoughtUpgrades [Slots]bool

// Each cell is a weapon item definition index.
//
//sp:name m_iWeaponPrimary
var weaponPrimary [Slots]int32

//sp:name m_iWeaponSecondary
var weaponSecondary [Slots]int32

//sp:name m_iWeaponMelee
var weaponMelee [Slots]int32

//sp:name m_iWeaponPDA2
var weaponPDA2 [Slots]int32

// Each cell is an attribute index.
//
//sp:name m_iAttribPrimary
var attribPrimary [Slots][MaxRuntimeAttributes]int32

//sp:name m_iAttribSecondary
var attribSecondary [Slots][MaxRuntimeAttributes]int32

//sp:name m_iAttribMelee
var attribMelee [Slots][MaxRuntimeAttributes]int32

// Each cell is an attribute value.
//
//sp:name m_flAttrValPrimary
var attrValPrimary [Slots][MaxRuntimeAttributes]float32

//sp:name m_flAttrValSecondary
var attrValSecondary [Slots][MaxRuntimeAttributes]float32

//sp:name m_flAttrValMelee
var attrValMelee [Slots][MaxRuntimeAttributes]float32

// ClearSavedAttributes forgets what the last weapon carried.
//
//sp:name ClearSavedAttributes
func ClearSavedAttributes(client int32) {
	for i := int32(0); i < MaxRuntimeAttributes; i++ {
		attribPrimary[client][i] = 0
		attribSecondary[client][i] = 0
		attribMelee[client][i] = 0
		attrValPrimary[client][i] = 0.0
		attrValSecondary[client][i] = 0.0
		attrValMelee[client][i] = 0.0
	}
}

/*
PrepareCustomLoadout decides what this bot will carry, per class and slot.

The sniper block is mvm-bj8: a stock primary leaves the definition index at
TF_ITEMDEF_DEFAULT, the classname lookup then fails on an item definition that is
not one, and the mission was skipped entirely, so the sniper stood where he
shopped for the rest of the mission. A sniper carrying the default primary is
carrying the stock rifle: there is nothing to look up and no lookup that can fail.
*/
//
//sp:name PrepareCustomLoadout
func PrepareCustomLoadout(client int32) {
	switch engine.PlayerClass(client) {
	case engine.ClassScout():
		weaponPrimary[client] = engine.PreferredWeaponForClass("scout", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("scout", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("scout", "melee", client)
	case engine.ClassSoldier():
		weaponPrimary[client] = engine.PreferredWeaponForClass("soldier", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("soldier", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("soldier", "melee", client)
	case engine.ClassPyro():
		weaponPrimary[client] = engine.PreferredWeaponForClass("pyro", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("pyro", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("pyro", "melee", client)
	case engine.ClassDemoMan():
		weaponPrimary[client] = engine.PreferredWeaponForClass("demoman", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("demoman", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("demoman", "melee", client)
	case engine.ClassHeavyweapons():
		weaponPrimary[client] = engine.PreferredWeaponForClass("heavyweapons", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("heavyweapons", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("heavyweapons", "melee", client)
	case engine.ClassEngineer():
		weaponPrimary[client] = engine.PreferredWeaponForClass("engineer", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("engineer", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("engineer", "melee", client)
	case engine.ClassMedic():
		weaponPrimary[client] = engine.PreferredWeaponForClass("medic", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("medic", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("medic", "melee", client)
	case engine.ClassSniper():
		weaponPrimary[client] = engine.PreferredWeaponForClass("sniper", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("sniper", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("sniper", "melee", client)
	case engine.ClassSpy():
		weaponPrimary[client] = engine.PreferredWeaponForClass("spy", "primary", client)
		weaponSecondary[client] = engine.PreferredWeaponForClass("spy", "secondary", client)
		weaponMelee[client] = engine.PreferredWeaponForClass("spy", "melee", client)
		weaponPDA2[client] = engine.PreferredWeaponForClass("spy", "pda2", client)
	}

	if engine.PlayerClass(client) == engine.ClassSniper() {
		holdingRifle := weaponPrimary[client] <= ItemDefDefault

		named, itemClassname := engine.ItemClassName(weaponPrimary[client])

		if !holdingRifle && named {
			holdingRifle = engine.StrEqual(itemClassname, "tf_weapon_sniperrifle") ||
				engine.StrEqual(itemClassname, "tf_weapon_sniperrifle_decap") ||
				engine.StrEqual(itemClassname, "tf_weapon_sniperrifle_classic")
		}

		if holdingRifle {
			engine.SetMission(client, engine.MissionSniper())
		}
	}

	hasCustomLoadout[client] = true
}

// ResetLoadouts puts the seat back to the stock items.
//
//sp:name ResetLoadouts
func ResetLoadouts(client int32) {
	hasCustomLoadout[client] = false
	weaponPrimary[client] = ItemDefDefault
	weaponSecondary[client] = ItemDefDefault
	weaponMelee[client] = ItemDefDefault
	weaponPDA2[client] = ItemDefDefault
}

/*
ReapplyItemUpgrades puts back what the bot bought, weapon by weapon.

The game hands a bot new weapons on every respawn and they come back stock, so
what the upgrade station wrote has to be written again. A zero index is the end
of the list: the rest of the array is the same.
*/
//
//sp:name ReapplyItemUpgrades
func ReapplyItemUpgrades(client int32, primary int32, secondary int32, melee int32) {
	var i int32

	if engine.IsValidEntity(primary) {
		for i = 0; i < MaxRuntimeAttributes; i++ {
			if attribPrimary[client][i] == 0 {
				break
			}

			engine.SetAttributeByDefIndex(primary, attribPrimary[client][i], attrValPrimary[client][i])
		}
	}

	if engine.IsValidEntity(secondary) {
		for i = 0; i < MaxRuntimeAttributes; i++ {
			if attribSecondary[client][i] == 0 {
				break
			}

			engine.SetAttributeByDefIndex(secondary, attribSecondary[client][i], attrValSecondary[client][i])
		}
	}

	if engine.IsValidEntity(melee) {
		for i = 0; i < MaxRuntimeAttributes; i++ {
			if attribMelee[client][i] == 0 {
				break
			}

			engine.SetAttributeByDefIndex(melee, attribMelee[client][i], attrValMelee[client][i])
		}
	}
}

// GiveGoldPanStats puts a killstreak on a weapon, drawn rather than chosen.
//
//sp:name GiveGoldPanStats
func GiveGoldPanStats(weapon int32) {
	// These may need to become arrays if the effect indexes update in the future.
	sheen := engine.RandomInt(1, 7)
	killstreaker := engine.RandomInt(2002, 2008)

	engine.SetAttribByName(weapon, "item style override", 0.0)
	engine.SetAttribByName(weapon, "killstreak tier", 3.0)
	engine.SetAttribByName(weapon, "killstreak idleeffect", float32(sheen))
	engine.SetAttribByName(weapon, "killstreak effect", float32(killstreaker))
}

/*
GetRandomWeaponForClass draws one item out of the pool for that class and slot.

The pools are the plugin's own lists, index for index. The spy's are filed one
slot along from where they are asked for, which is how the plugin has always read
them.
*/
//
//sp:name GetRandomWeaponForClass
//nolint:gocritic // ifElseChain: the shipped file is a chain, and a switch would be a different file
func GetRandomWeaponForClass(class string, slot string) int32 {
	if engine.StrEqualLiteral(class, "scout", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsScoutPrimary[engine.RandomInt(0, int32(len(weaponsScoutPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsScoutSecondary[engine.RandomInt(0, int32(len(weaponsScoutSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsScoutMelee[engine.RandomInt(0, int32(len(weaponsScoutMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "soldier", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSoldierPrimary[engine.RandomInt(0, int32(len(weaponsSoldierPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSoldierSecondary[engine.RandomInt(0, int32(len(weaponsSoldierSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSoldierMelee[engine.RandomInt(0, int32(len(weaponsSoldierMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "pyro", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsPyroPrimary[engine.RandomInt(0, int32(len(weaponsPyroPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsPyroSecondary[engine.RandomInt(0, int32(len(weaponsPyroSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsPyroMelee[engine.RandomInt(0, int32(len(weaponsPyroMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "demoman", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsDemomanPrimary[engine.RandomInt(0, int32(len(weaponsDemomanPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsDemomanSecondary[engine.RandomInt(0, int32(len(weaponsDemomanSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsDemomanMelee[engine.RandomInt(0, int32(len(weaponsDemomanMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "heavyweapons", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsHeavyPrimary[engine.RandomInt(0, int32(len(weaponsHeavyPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsHeavySecondary[engine.RandomInt(0, int32(len(weaponsHeavySecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsHeavyMelee[engine.RandomInt(0, int32(len(weaponsHeavyMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "engineer", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsEngineerPrimary[engine.RandomInt(0, int32(len(weaponsEngineerPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsEngineerSecondary[engine.RandomInt(0, int32(len(weaponsEngineerSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsEngineerMelee[engine.RandomInt(0, int32(len(weaponsEngineerMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "medic", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsMedicPrimary[engine.RandomInt(0, int32(len(weaponsMedicPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsMedicSecondary[engine.RandomInt(0, int32(len(weaponsMedicSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsMedicMelee[engine.RandomInt(0, int32(len(weaponsMedicMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "sniper", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSniperPrimary[engine.RandomInt(0, int32(len(weaponsSniperPrimary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSniperSecondary[engine.RandomInt(0, int32(len(weaponsSniperSecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSniperMelee[engine.RandomInt(0, int32(len(weaponsSniperMelee))-1)]
		}
	} else if engine.StrEqualLiteral(class, "spy", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSpySecondary[engine.RandomInt(0, int32(len(weaponsSpySecondary))-1)]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSpyBuilding[engine.RandomInt(0, int32(len(weaponsSpyBuilding))-1)]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSpyMelee[engine.RandomInt(0, int32(len(weaponsSpyMelee))-1)]
		}

		if engine.StrEqualLiteral(slot, "pda2", false) {
			return weaponsSpyPda2[engine.RandomInt(0, int32(len(weaponsSpyPda2))-1)]
		}
	} else {
		engine.PrintToChatAll("[GetRandomWeaponForClass] Unknown class of %s", class)
		engine.LogError("GetRandomWeaponForClass: Unknown class %s", class)
	}

	return -1
}

// LoadLoadoutFunctions puts the command the bots use on the console.
//
//sp:name LoadLoadoutFunctions
func LoadLoadoutFunctions() {
	engine.RegConsoleCmd("sm_redbot_upgraded", CommandBoughtUpgrades)
}

/*
CommandBoughtUpgrades remembers what the upgrade station wrote on this bot's
weapons.

The bot says so itself: the station is a menu the bot walks through, and this is
the line it types when it comes out. Only worth remembering with custom loadouts
on, because that is the only case where the weapons are handed back rather than
kept.
*/
//
//sp:name Command_BoughtUpgrades
//sp:public
//
//nolint:revive // unused-parameter: the argument count is the console's, and this command takes none
func CommandBoughtUpgrades(client int32, args int32) engine.Outcome {
	// Only need to remember upgrades if using custom loadouts.
	if !engine.UseCustomLoadouts().Bool() {
		return engine.PluginHandled()
	}

	// Only our bots should execute this command.
	if !engine.IsFakeClient(client) {
		return engine.PluginHandled()
	}

	primaryWep := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary())
	secondaryWep := engine.PlayerWeaponSlot(client, engine.WeaponSlotSecondary())
	meleeWep := engine.PlayerWeaponSlot(client, engine.WeaponSlotMelee())

	ClearSavedAttributes(client)

	var count int32
	var attr engine.Address

	if engine.IsValidEntity(primaryWep) {
		count = engine.ListDefIndices(primaryWep, attribPrimary[client])

		if count > 0 {
			for i := int32(0); i < count; i++ {
				attr = engine.AttribByDefIndex(primaryWep, attribPrimary[client][i])
				attrValPrimary[client][i] = engine.AttribValueAt(attr)
			}
		}
	}

	if engine.IsValidEntity(secondaryWep) {
		count = engine.ListDefIndices(secondaryWep, attribSecondary[client])

		if count > 0 {
			for i := int32(0); i < count; i++ {
				attr = engine.AttribByDefIndex(secondaryWep, attribSecondary[client][i])
				attrValSecondary[client][i] = engine.AttribValueAt(attr)
			}
		}
	}

	if engine.IsValidEntity(meleeWep) {
		count = engine.ListDefIndices(meleeWep, attribMelee[client])

		if count > 0 {
			for i := int32(0); i < count; i++ {
				attr = engine.AttribByDefIndex(meleeWep, attribMelee[client][i])
				attrValMelee[client][i] = engine.AttribValueAt(attr)
			}
		}
	}

	if engine.ManagerDebug().Bool() {
		engine.PrintToChatAll("[Command_BoughtUpgrades] SAVED WEAPON STATS FOR %N", client)
	}

	hasBoughtUpgrades[client] = true

	return engine.PluginHandled()
}

/*
WeaponPoolCount is how many items the pool for that class and slot holds.

It and WeaponPoolAt exist so the loadout menu can walk a pool without repeating
the class-and-slot chain a third time. The menu used to carry its own copy of
it, twenty seven blocks of the same four lines; the pools live here, so the way
into them does too.
*/
//
//sp:name WeaponPoolCount
//nolint:gocritic // ifElseChain: the same chain as GetRandomWeaponForClass, beside it on purpose
func WeaponPoolCount(class string, slot string) int32 {
	if engine.StrEqualLiteral(class, "scout", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsScoutPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsScoutSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsScoutMelee))
		}
	} else if engine.StrEqualLiteral(class, "soldier", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsSoldierPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsSoldierSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsSoldierMelee))
		}
	} else if engine.StrEqualLiteral(class, "pyro", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsPyroPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsPyroSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsPyroMelee))
		}
	} else if engine.StrEqualLiteral(class, "demoman", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsDemomanPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsDemomanSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsDemomanMelee))
		}
	} else if engine.StrEqualLiteral(class, "heavyweapons", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsHeavyPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsHeavySecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsHeavyMelee))
		}
	} else if engine.StrEqualLiteral(class, "engineer", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsEngineerPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsEngineerSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsEngineerMelee))
		}
	} else if engine.StrEqualLiteral(class, "medic", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsMedicPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsMedicSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsMedicMelee))
		}
	} else if engine.StrEqualLiteral(class, "sniper", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsSniperPrimary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsSniperSecondary))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsSniperMelee))
		}
	} else if engine.StrEqualLiteral(class, "spy", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return int32(len(weaponsSpySecondary))
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return int32(len(weaponsSpyBuilding))
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return int32(len(weaponsSpyMelee))
		}

		if engine.StrEqualLiteral(slot, "pda2", false) {
			return int32(len(weaponsSpyPda2))
		}
	}

	return 0
}

// WeaponPoolAt is one item out of that pool.
//
//sp:name WeaponPoolAt
//nolint:gocritic // ifElseChain: the same chain as GetRandomWeaponForClass, beside it on purpose
func WeaponPoolAt(class string, slot string, index int32) int32 {
	if engine.StrEqualLiteral(class, "scout", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsScoutPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsScoutSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsScoutMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "soldier", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSoldierPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSoldierSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSoldierMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "pyro", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsPyroPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsPyroSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsPyroMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "demoman", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsDemomanPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsDemomanSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsDemomanMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "heavyweapons", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsHeavyPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsHeavySecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsHeavyMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "engineer", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsEngineerPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsEngineerSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsEngineerMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "medic", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsMedicPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsMedicSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsMedicMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "sniper", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSniperPrimary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSniperSecondary[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSniperMelee[index]
		}
	} else if engine.StrEqualLiteral(class, "spy", false) {
		if engine.StrEqualLiteral(slot, "primary", false) {
			return weaponsSpySecondary[index]
		}

		if engine.StrEqualLiteral(slot, "secondary", false) {
			return weaponsSpyBuilding[index]
		}

		if engine.StrEqualLiteral(slot, "melee", false) {
			return weaponsSpyMelee[index]
		}

		if engine.StrEqualLiteral(slot, "pda2", false) {
			return weaponsSpyPda2[index]
		}
	}

	return 0
}
