package engine

/*
The upgrade station: what it offers, what it costs, and how the mod ranks it.

The manager and the attribute definitions are the game's own objects, reached by
address through tf_upgrades.sp. The ranking is this repository's, emitted from
internal/upgrade.
*/

// UpgradeCalls are the answers.
type UpgradeCalls struct {
	UpgradePostAction         func(client int32) Outcome
	NearestTeammate           func(client int32, maxDistance float32) int32
	SetHasUpgraded            func(client int32, done bool)
	SetShoppedThisBreak       func(client int32, shopped bool)
	SetBuyUpgradesNumber      func(client int32, count int32)
	SetNestArea               func(actor int32, area Area)
	SetNestRelocate           func(actor int32, area Area)
	BoughtUpgradesCommand     func(client int32, args int32) Outcome
	UpgradeTier               func(upgrade int32) int32
	IsUpgradeTierEnabled      func(client int32, slot int32, tier int32) bool
	UpgradeTierCap            func(attribute Text) int32
	BuyUpgrade                func(client int32, count int32, slot int32, index int32)
	PurchasedUpgrades         func(client int32) int32
	SetPurchasedUpgrades      func(client int32, count int32)
	UpgradeCount              func() int32
	IsUpgradeManagerUp        func() bool
	UpgradeCountRaw           func() int32
	UpgradeByIndex            func(index int32) Address
	UpgradeUIGroup            func(upgrade Address) int32
	UpgradeAttribute          func(upgrade Address) Text
	AttributeDefinitionByName func(name Text) Address
	AttributeDefinitionIndex  func(attr Address) int32
	CanUpgradeWithAttrib      func(client int32, slot int32, index int32, upgrade Address) bool
	CostForUpgrade            func(upgrade Address, slot int32, playerClass int32, client int32) int32
	IsUpgradeWasted           func(client int32, attribute Text) bool
	AttributeID               func(name Text) int32
	UpgradeRankGeneral        func(attribute int32) int32
	UpgradeRankClass          func(playerClass Class, slot int32, attribute int32) int32
	UpgradeRankLoadout        func(itemDefIndex int32, attribute int32) int32
	UpgradeRankEngineerMetal  func(attribute int32) int32
	UnrankedUpgradePriority   func() int32
	PlayerUpgrades            func(client int32) List
	SetPlayerUpgrades         func(client int32, list List)
}

var upgrades UpgradeCalls

// InstallUpgrades puts a set of answers behind them.
func InstallUpgrades(c UpgradeCalls) func() {
	previous := upgrades
	upgrades = c
	return func() { upgrades = previous }
}

// UIGroupAttachedToPlayer is UIGROUP_UPGRADE_ATTACHED_TO_PLAYER: an upgrade the
// game hangs off the player rather than off a weapon.
//
//sp:global UIGROUP_UPGRADE_ATTACHED_TO_PLAYER
func UIGroupAttachedToPlayer() int32 { return 1 }

// UIGroupPowerupBottle is UIGROUP_POWERUPBOTTLE, the canteen charges.
//
//sp:global UIGROUP_POWERUPBOTTLE
func UIGroupPowerupBottle() int32 { return 2 }

// UpgradeCount is how many upgrades the station offers. Still in
// tf_upgrades.sp.
//
//sp:plugin UpgradeCount
func UpgradeCount() int32 {
	if upgrades.UpgradeCount == nil {
		missing("UpgradeCount")
	}
	return upgrades.UpgradeCount()
}

// UpgradeByIndex is the game's upgrade record at that index.
//
//sp:plugin UpgradeAddressByIndex
func UpgradeByIndex(index int32) Address {
	if upgrades.UpgradeByIndex == nil {
		missing("GetUpgradeByIndex")
	}
	return upgrades.UpgradeByIndex(index)
}

// UpgradeUIGroup is which part of the station's UI it belongs to, which is how
// a player upgrade is told from a weapon one.
//
//sp:plugin UpgradeUIGroupOf
func UpgradeUIGroup(upgrade Address) int32 {
	if upgrades.UpgradeUIGroup == nil {
		missing("UpgradeUIGroupOf")
	}
	return upgrades.UpgradeUIGroup(upgrade)
}

// UpgradeAttribute is the attribute name it grants.
//
//sp:plugin UpgradeAttributeOf returns
func UpgradeAttribute(upgrade Address) Text {
	if upgrades.UpgradeAttribute == nil {
		missing("UpgradeAttributeOf")
	}
	return upgrades.UpgradeAttribute(upgrade)
}

// AttributeDefinitionByName is the schema's record for that attribute.
//
//sp:plugin CEIAD_GetAttributeDefinitionByName
func AttributeDefinitionByName(name Text) Address {
	if upgrades.AttributeDefinitionByName == nil {
		missing("CEIAD_GetAttributeDefinitionByName")
	}
	return upgrades.AttributeDefinitionByName(name)
}

// AttributeDefinitionIndex is its index.
//
//sp:plugin AttributeDefinitionIndexOf
func AttributeDefinitionIndex(attr Address) int32 {
	if upgrades.AttributeDefinitionIndex == nil {
		missing("AttributeDefinitionIndexOf")
	}
	return upgrades.AttributeDefinitionIndex(attr)
}

// CanUpgradeWithAttrib says the game would let this bot buy it for that slot.
//
//sp:body CanUpgradeWithAttrib
func CanUpgradeWithAttrib(client int32, slot int32, index int32, upgrade Address) bool {
	if upgrades.CanUpgradeWithAttrib == nil {
		missing("CanUpgradeWithAttrib")
	}
	return upgrades.CanUpgradeWithAttrib(client, slot, index, upgrade)
}

// CostForUpgrade is what the next step costs.
//
//sp:body GetCostForUpgrade
func CostForUpgrade(upgrade Address, slot int32, playerClass int32, client int32) int32 {
	if upgrades.CostForUpgrade == nil {
		missing("GetCostForUpgrade")
	}
	return upgrades.CostForUpgrade(upgrade, slot, playerClass, client)
}

// IsUpgradeWasted says the bot is not carrying what this upgrades, so the
// credits would be set on fire. Still in tf_upgrades.sp.
//
//sp:body IsUpgradeWasted
func IsUpgradeWasted(client int32, attribute Text) bool {
	if upgrades.IsUpgradeWasted == nil {
		missing("IsUpgradeWasted")
	}
	return upgrades.IsUpgradeWasted(client, attribute)
}

// AttributeID turns an attribute name into the number the ranking tables switch
// on. Emitted by internal/tables.
//
//sp:body AttributeID
func AttributeID(name Text) int32 {
	if upgrades.AttributeID == nil {
		missing("AttributeID")
	}
	return upgrades.AttributeID(name)
}

// The four ranking tables, emitted from internal/upgrade.

// UpgradeRankGeneral is the score an attribute has for anybody.
//
//sp:body UpgradeRankGeneral
func UpgradeRankGeneral(attribute int32) int32 {
	if upgrades.UpgradeRankGeneral == nil {
		missing("UpgradeRankGeneral")
	}
	return upgrades.UpgradeRankGeneral(attribute)
}

// UpgradeRankClass is the score for one class and slot.
//
//sp:body UpgradeRankClass
func UpgradeRankClass(playerClass Class, slot int32, attribute int32) int32 {
	if upgrades.UpgradeRankClass == nil {
		missing("UpgradeRankClass")
	}
	return upgrades.UpgradeRankClass(playerClass, slot, attribute)
}

// UpgradeRankLoadout is the score for one weapon.
//
//sp:body UpgradeRankLoadout
func UpgradeRankLoadout(itemDefIndex int32, attribute int32) int32 {
	if upgrades.UpgradeRankLoadout == nil {
		missing("UpgradeRankLoadout")
	}
	return upgrades.UpgradeRankLoadout(itemDefIndex, attribute)
}

// UpgradeRankEngineerMetal is the score for the metal upgrades, which do not
// hang off the gun and so are asked before the slot is.
//
//sp:body UpgradeRankEngineerMetal
func UpgradeRankEngineerMetal(attribute int32) int32 {
	if upgrades.UpgradeRankEngineerMetal == nil {
		missing("UpgradeRankEngineerMetal")
	}
	return upgrades.UpgradeRankEngineerMetal(attribute)
}

// UnrankedUpgradePriority is what a name the table does not rank becomes.
//
//sp:body UnrankedUpgradePriority
func UnrankedUpgradePriority() int32 {
	if upgrades.UnrankedUpgradePriority == nil {
		missing("UnrankedUpgradePriority")
	}
	return upgrades.UnrankedUpgradePriority()
}

// The three loadout slots the ranking asks about by number, as the preamble
// emits them.

// LoadoutSlotPrimary is TF_LOADOUT_SLOT_PRIMARY.
//
//sp:global TF_LOADOUT_SLOT_PRIMARY
func LoadoutSlotPrimary() int32 { return 0 }

// LoadoutSlotMelee is TF_LOADOUT_SLOT_MELEE.
//
//sp:global TF_LOADOUT_SLOT_MELEE
func LoadoutSlotMelee() int32 { return 2 }

// LoadoutSlotAction is TF_LOADOUT_SLOT_ACTION, where the canteen sits.
//
//sp:global TF_LOADOUT_SLOT_ACTION
func LoadoutSlotAction() int32 { return 9 }

// ChooseWeapon is the ternary between two weapon ids.
//
//sp:choice ?:
func ChooseWeapon(cond bool, yes Weapon, no Weapon) Weapon {
	if cond {
		return yes
	}
	return no
}

// WeaponNone is TF_WEAPON_NONE, which the shipped file spells -1 where it means
// "no weapon in that slot".
//
//sp:global -1
func WeaponNone() Weapon { return -1 }

// LoadoutSlotBuilding is TF_LOADOUT_SLOT_BUILDING, the spy's sapper and the
// engineer's build PDA.
//
//sp:global TF_LOADOUT_SLOT_BUILDING
func LoadoutSlotBuilding() int32 { return 4 }

// LoadoutSlotPDA is TF_LOADOUT_SLOT_PDA, the engineer's destroy PDA.
//
//sp:global TF_LOADOUT_SLOT_PDA
func LoadoutSlotPDA() int32 { return 5 }

// UpgradeTiersMax is UPGRADE_TIERS_MAX, and no stock upgrade has more steps
// than this. Emitted with the rules.
//
//sp:global UPGRADE_TIERS_MAX
func UpgradeTiersMax() int32 { return 4 }

// UpgradeTier is which tier the station files an upgrade under. Still in
// sdkcalls.sp.
//
//sp:body GetUpgradeTier
func UpgradeTier(upgrade int32) int32 {
	if upgrades.UpgradeTier == nil {
		missing("GetUpgradeTier")
	}
	return upgrades.UpgradeTier(upgrade)
}

// IsUpgradeTierEnabled says the mission lets this tier be bought yet. Still in
// sdkcalls.sp.
//
//sp:body IsUpgradeTierEnabled
func IsUpgradeTierEnabled(client int32, slot int32, tier int32) bool {
	if upgrades.IsUpgradeTierEnabled == nil {
		missing("IsUpgradeTierEnabled")
	}
	return upgrades.IsUpgradeTierEnabled(client, slot, tier)
}

// UpgradeTierCap is how many steps of one attribute are worth buying at once.
// Ported, upgraderules.
//
//sp:body UpgradeTierCap
func UpgradeTierCap(attribute Text) int32 {
	if upgrades.UpgradeTierCap == nil {
		missing("UpgradeTierCap")
	}
	return upgrades.UpgradeTierCap(attribute)
}

// PurchasedUpgrades is how many steps this bot has bought this trip, and
// SetPurchasedUpgradesOf writes it.
//
//sp:slot m_nPurchasedUpgrades
func PurchasedUpgrades(client int32) int32 {
	if upgrades.PurchasedUpgrades == nil {
		missing("m_nPurchasedUpgrades")
	}
	return upgrades.PurchasedUpgrades(client)
}

// SetPurchasedUpgradesOf writes it.
//
//sp:slotset m_nPurchasedUpgrades
func SetPurchasedUpgradesOf(client int32, count int32) {
	if upgrades.SetPurchasedUpgrades == nil {
		missing("m_nPurchasedUpgrades")
	}
	upgrades.SetPurchasedUpgrades(client, count)
}

// BoughtUpgradesCommand records what this bot bought, for the stats plugin.
// Ported, loadouts.
//
//sp:body Command_BoughtUpgrades
func BoughtUpgradesCommand(client int32, args int32) Outcome {
	if upgrades.BoughtUpgradesCommand == nil {
		missing("Command_BoughtUpgrades")
	}
	return upgrades.BoughtUpgradesCommand(client, args)
}

// UpgradePostAction is what a bot does once the trip is over. Ported, dispatch.
//
//sp:body GetUpgradePostAction after action
func UpgradePostAction(client int32) Outcome {
	if upgrades.UpgradePostAction == nil {
		missing("GetUpgradePostAction")
	}
	return upgrades.UpgradePostAction(client)
}

// NearestTeammate is the closest friendly player within that range. Ported,
// finders.
//
//sp:body GerNearestTeammate
func NearestTeammate(client int32, maxDistance float32) int32 {
	if upgrades.NearestTeammate == nil {
		missing("GerNearestTeammate")
	}
	return upgrades.NearestTeammate(client, maxDistance)
}

// IsUpgradeManagerUp says the game has an upgrade manager, which it has not
// until an MvM map has loaded. Still in tf_upgrades.sp.
//
//sp:plugin IsUpgradeManagerUp
func IsUpgradeManagerUp() bool {
	if upgrades.IsUpgradeManagerUp == nil {
		missing("IsUpgradeManagerUp")
	}
	return upgrades.IsUpgradeManagerUp()
}

// UpgradeCountRaw is the count the manager gives, unclamped: UpgradeCount
// hides an unbelievable one and sm_dump_upgrades exists to report it. Still in
// tf_upgrades.sp.
//
//sp:plugin UpgradeCountRaw
func UpgradeCountRaw() int32 {
	if upgrades.UpgradeCountRaw == nil {
		missing("UpgradeCountRaw")
	}
	return upgrades.UpgradeCountRaw()
}

// AttributeDescriptionMax is MAX_ATTRIBUTE_DESCRIPTION_LENGTH.
//
//sp:global MAX_ATTRIBUTE_DESCRIPTION_LENGTH
func AttributeDescriptionMax() int32 { return 128 }
