/*
Package tfupgrades is source/redbots3/tf_upgrades.sp: reading the game's own
upgrade table.

The table is a C++ object nothing in SourceMod names, so it is reached by
address: a methodmap over an Address, its fields at offsets the gamedata file
gives, and one wrapper per thing the shopping code needs. The wrappers exist
because a generated body reaches a methodmap and not the memory behind it.

The offsets and their arithmetic are the game's, and the layout they follow is
written out in the comment block at the end of the shipped file.
*/
package tfupgrades

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
MaxUpgrades is a ceiling on the number of upgrades, not the number of them.

The count comes from the game, through UpgradeCount below. This is only the
point past which the manager is not what we think it is and walking further
would read memory that is not a list of upgrades.
*/
//
//sp:name MAX_UPGRADES
const MaxUpgrades = 128

/*
UpgradeCountMeasured is what the game held when this was last measured, for
when it will not say.

sm_dump_upgrades walked CMannVsMachineUpgradeManager and printed sixty three.
The constant here used to be sixty two and was used as the loop bound, so the
last upgrade the game holds was one no loop in this mod ever reached.
*/
//
//sp:name UPGRADE_COUNT_MEASURED
const UpgradeCountMeasured = 63

// AttributeDescriptionLength is the size of the attribute string.
//
//sp:name MAX_ATTRIBUTE_DESCRIPTION_LENGTH
const AttributeDescriptionLength = 128

// The fields of CEconItemAttributeDefinition, by offset.
const (
	//sp:name m_pKVAttribute
	//nolint:unused // emitted, not read from Go: it names the field the game keeps at offset zero
	kvAttribute = 0
	//sp:name m_nDefIndex
	defIndex = 4
)

// The fields of CMannVsMachineUpgradeManager, by offset.
const (
	//sp:name m_Upgrades
	upgrades = 12
	//sp:name CMannVsMachineUpgradeManager_Size
	//nolint:unused // emitted, not read from Go: it records the object's width, which the shipped file wrote down and nothing computes from
	upgradeManagerSize = 28
)

/*
AttributeDefinition is CEconItemAttributeDefinition, the schema's record for
one attribute.
*/
//
//sp:name CEconItemAttributeDefinition
//sp:methodmap
type AttributeDefinition int32

/*
	GetIndex is the attribute's index in the schema

Read forwards, and backwards when the answer is not believable: the field sits
at a different side of the record on some builds, and an index outside the
schema's range is the only way to tell which one this is.
*/
//
//sp:name GetIndex
func (a AttributeDefinition) GetIndex() int32 {
	iAttribIndex := engine.LoadFromAddress(engine.Address(a) + engine.Address(defIndex))

	if iAttribIndex > 3018 || iAttribIndex < 0 {
		iAttribIndex = engine.LoadFromAddress(engine.Address(a) - engine.Address(defIndex))
	}

	return iAttribIndex
}

// Upgrades is CMannVsMachineUpgrades, one row of the game's upgrade table.
//
//sp:name CMannVsMachineUpgrades
//sp:methodmap
type Upgrades int32

/*
	Attribute is the attribute name the upgrade grants

szAttrib sits at offset zero, so there is nothing to add.
*/
//
//sp:name m_szAttribute
//sp:returns
func (u Upgrades) Attribute() (attribute [AttributeDescriptionLength]byte) {
	for i := int32(0); i < AttributeDescriptionLength; i++ {
		//nolint:gosec // G115: the field is a char and the native answers a cell, which is what SourcePawn narrows here too
		attribute[i] = byte(engine.LoadFromAddress(engine.Address(u) + engine.Address(i)))
	}

	return attribute
}

// Cap is the most of that attribute one weapon may hold.
//
//sp:name m_flCap
func (u Upgrades) Cap() float32 {
	return float32(engine.LoadFromAddress(engine.Address(u) + engine.Address(engine.OffsetCap())))
}

// UIGroup is which part of the upgrade station shows it, which is how a player
// upgrade is told from a weapon one.
//
//sp:name m_iUIGroup
func (u Upgrades) UIGroup() int32 {
	return engine.LoadFromAddress(engine.Address(u) + engine.Address(engine.OffsetUIGroup()))
}

// UpgradeManager is CMannVsMachineUpgradeManager, the object that owns the
// table.
//
//sp:name CMannVsMachineUpgradeManager
//sp:methodmap CMannVsMachineUpgrades
type UpgradeManager int32

// NewUpgradeManager is the manager the plugin found at load. A methodmap's
// constructor, so it is written inside the braces and called by the tag's name.
//
//sp:name CMannVsMachineUpgradeManager
//nolint:revive // unused-receiver: a constructor is a method with nothing to read off the receiver
func (m UpgradeManager) NewUpgradeManager() UpgradeManager {
	return UpgradeManager(engine.UpgradesAddress())
}

/*
	Count is how many upgrades the game actually holds

m_Upgrades is a CUtlVector, and GetUpgradeByIndex below reads its m_pMemory at
offset zero to find the elements. m_Size is the fourth int of that structure,
after the pointer, the allocation count and the growth size, which puts it
twelve bytes in.

Read rather than assumed, because the whole point of asking is that nobody
should be counting lines in a text file and hoping the game agrees.
*/
//
//sp:name Count
func (m UpgradeManager) Count() int32 {
	return engine.LoadFromAddress(engine.Address(m) + engine.Address(upgrades+12))
}

// GetUpgradeByIndex is one row of the table, by the index the game uses.
//
//sp:name GetUpgradeByIndex
func (m UpgradeManager) GetUpgradeByIndex(index int32) Upgrades {
	rawUpgrades := engine.Address(m) + engine.Address(upgrades)
	pUpgrades := engine.DereferencePointer(rawUpgrades)

	return Upgrades(pUpgrades + engine.Address(index*engine.UpgradeSize()))
}

/*
	AttributeDefinitionByName is the schema's record for an attribute, by name

Null when the schema itself is not up, which is what a server that has not
loaded an MvM map looks like.
*/
//
//sp:name CEIAD_GetAttributeDefinitionByName
//sp:public
func AttributeDefinitionByName(szAttribute engine.Text) AttributeDefinition {
	CEconItemSchema := engine.GEconItemSchema()

	if CEconItemSchema == engine.NoAddress() {
		return AttributeDefinition(engine.NoAddress())
	}

	return AttributeDefinition(engine.GetAttributeDefinitionByName(CEconItemSchema, szAttribute))
}

// UpgradeCount is how many upgrades the game holds, asked of the game rather
// than counted in a text file.
//
//sp:name UpgradeCount
func UpgradeCount() int32 {
	count := engine.MvMUpgradeManagerCount()

	if count < 1 || count > MaxUpgrades {
		return UpgradeCountMeasured
	}

	return count
}

// InitMvMUpgrades reads the offsets one row of the table is laid out by.
//
//sp:name InitMvMUpgrades
func InitMvMUpgrades(hGamedata engine.GameData) {
	engine.SetOffsetCap(hGamedata.OffsetOf("CMannVsMachineUpgrades::flCap"))
	engine.SetOffsetUIGroup(hGamedata.OffsetOf("CMannVsMachineUpgrades::nUIGroup"))
	engine.SetOffsetTier(hGamedata.OffsetOf("CMannVsMachineUpgrades::nTier"))
	engine.SetUpgradeSize(engine.OffsetTier() + 4)
}

/*
	The wrappers.

The generated shopping code reaches these by name. Each is one methodmap call:
the methodmap is this file's, built on the gamedata offsets, and a generated
body has no form for a methodmap on an Address.
*/

// UpgradeUIGroupOf is which part of the station shows that upgrade.
//
//sp:name UpgradeUIGroupOf
func UpgradeUIGroupOf(upgrade engine.Address) int32 {
	return Upgrades(upgrade).UIGroup()
}

// UpgradeAttributeOf is the attribute name it grants.
//
//sp:name UpgradeAttributeOf
//sp:returns
func UpgradeAttributeOf(upgrade engine.Address) (attribute [AttributeDescriptionLength]byte) {
	attribute = Upgrades(upgrade).Attribute()

	return attribute
}

// UpgradeAddressByIndex is the row at that index.
//
//sp:name UpgradeAddressByIndex
func UpgradeAddressByIndex(index int32) engine.Address {
	return engine.Address(UpgradeManager(engine.MvMUpgradeManager()).GetUpgradeByIndex(index))
}

// IsUpgradeManagerUp says the game has one, which it has not until an MvM map
// has loaded.
//
//sp:name IsUpgradeManagerUp
func IsUpgradeManagerUp() bool {
	return engine.MvMUpgradeManager() != engine.NoAddress()
}

/*
	UpgradeCountRaw is the count the manager itself gives, unclamped

UpgradeCount above falls back to UPGRADE_COUNT_MEASURED when the answer is not
believable, which is what the shopping code wants and what sm_dump_upgrades is
for reporting.
*/
//
//sp:name UpgradeCountRaw
func UpgradeCountRaw() int32 {
	return engine.MvMUpgradeManagerCount()
}

// AttributeDefinitionIndexOf is the schema index of that attribute record.
//
//sp:name AttributeDefinitionIndexOf
func AttributeDefinitionIndexOf(attr engine.Address) int32 {
	return AttributeDefinition(attr).GetIndex()
}
