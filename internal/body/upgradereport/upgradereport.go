/*
Package upgradereport prints what a bot is carrying, by attribute name, out of
source/tf2_defenderbots.sp.

Every line of this used PrintToChat, which needs a client, and rcon has not got
one: run from the console it printed nothing at all and looked like a bot with
no upgrades. That is the one place anybody would run it from on a test server.

The attribute index alone was not much better. "INDEX 56, VALUE 0.800000" is the
answer to a question nobody asked; the schema has the name and it costs a
lookup.
*/
package upgradereport

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Attributes is MAX_RUNTIME_ATTRIBUTES, the most the game keeps on one entity.
const Attributes = 20

// ShowUpgradesOn prints every attribute on one entity, named.
//
//sp:name ShowUpgradesOn
func ShowUpgradesOn(client int32, entity int32, what engine.Text) {
	var attribIndexes [Attributes]int32
	count := engine.ListDefIndices(entity, attribIndexes)

	engine.ReplyToCommand(client, "%s: %d upgrades", what, count)

	for i := int32(0); i < count; i++ {
		pAttr := engine.AttribByDefIndex(entity, attribIndexes[i])
		value := engine.AttribValueAt(pAttr)

		ok, name := engine.AttributeName(attribIndexes[i])

		if !ok {
			engine.StrcopyText(name, engine.AttributeNameSize(), "(unnamed)")
		}

		engine.ReplyToCommand(client, "  %-48s %.3f", name, value)
	}
}

// ShowPlayerUpgrades prints one player's, either the player itself or one slot.
//
//sp:name ShowPlayerUpgrades
func ShowPlayerUpgrades(client int32, target int32, slot int32) {
	_, who := engine.ClientName(target)

	if slot == -1 {
		label := engine.FormatEx("%s, on himself", who)

		ShowUpgradesOn(client, target, label)

		return
	}

	weapon := engine.PlayerWeaponSlot(target, slot)

	if weapon == -1 {
		engine.ReplyToCommand(client, "%s has nothing in slot %d.", who, slot)

		return
	}

	label := engine.FormatEx("%s, slot %d", who, slot)

	ShowUpgradesOn(client, weapon, label)
}
