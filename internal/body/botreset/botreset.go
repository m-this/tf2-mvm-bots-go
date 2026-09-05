/*
Package botreset is what happens to a seat between bots.

InitNextBotPathing builds the two route objects a bot walks with, once per
client at load. ResetNextBot clears everything the mod keeps about whoever was
in that seat.

The shipped ResetNextBot was one flat list of every field the mod keeps per
client. Nobody could add a behaviour without finding that list and remembering
to add to it, and forgetting was silent: the next bot in the seat inherited the
last one's target. Each package clears its own now, and this calls them, so a
new behaviour brings its own reset with it or has none to bring.
*/
package botreset

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// InitNextBotPathing gives every client slot its two route objects.
//
//sp:name InitNextBotPathing
func InitNextBotPathing() {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		engine.SetPath(i, engine.NewRoutePath(engine.FilterIgnoreActors(), engine.FilterOnlyActors()))
		engine.SetChasePath(i, engine.NewChasePath(engine.LeadSubject(), engine.Default(), engine.FilterIgnoreActors(), engine.FilterOnlyActors()))
	}
}

// ResetNextBot forgets everything about whoever was in this seat.
//
//sp:name ResetNextBot
func ResetNextBot(client int32) {
	engine.SetRepathTime(client, 0.0)
	engine.ResetScoutJump(client)
	engine.ResetSpyCheck(client)
	engine.ResetStickyTrap(client)

	engine.ResetBottle(client)

	engine.ResetAttack(client)
	engine.ResetMarkGiant(client)
	engine.ResetCollectMoney(client)
	engine.ResetGotoUpgrade(client)

	// The shopping trip is the last hand-written behaviour, so its three
	// fields are still reached by slot. See mvm-z83.64.
	engine.SetNextUpgrade(client, 0.0)
	engine.SetPurchasedUpgrades(client, 0)
	engine.SetUpgradingTime(client, 0.0)

	engine.ResetGetAmmo(client)
	engine.ResetMoveToFront(client)
	engine.ResetGetHealth(client)

	// The engineer's own state is reset inside its action.
	engine.ResetSpySap(client)
	engine.ResetSpySapPlayer(client)
	engine.ResetAttackForUber(client)
	engine.ResetAttackTank(client)
	engine.ResetDestroyTeleporter(client)
	engine.ResetBuildTeleporter(client)
	engine.ResetGuardPoint(client)

	engine.PluginBotOf(client).Reset()
}
