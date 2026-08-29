/*
Package collectnearmoney is source/redbots3/behavior/collectnearmoney.sp.

A bot with nothing to fight walks to the nearest money and picks it up. The
smallest behaviour in the plugin, and the second across.

//sp:action DefenderCollectNearMoney CTFBotCollectNearMoney
*/
package collectnearmoney

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// OnStart aims the path. The pack was picked before the action started.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	// NOTE: we pick a money pack before entering this action

	return engine.Continue()
}

// Update walks to the pack, and gives up the moment anything is worth fighting.
func Update(actor int32) engine.Outcome {
	if !engine.IsValidCurrencyPack(engine.CurrencyPackOf(actor)) {
		return engine.Done("No money")
	}

	myBot := engine.NextBotOf(actor)
	threat := myBot.Vision().PrimaryKnownThreat(false)

	if threat != 0 {
		return engine.Done("I see a threat")
	}

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 1.0))
		engine.RepathToPos(actor, myBot, engine.WorldSpaceCenter(engine.CurrencyPackOf(actor)))
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnEnd forgets the pack, so the next behaviour does not inherit it.
func OnEnd(actor int32) {
	engine.SetCurrencyPack(actor, -1)
}

// SelectTarget picks a pack, and refuses to while there is a threat about.
//
//sp:name CTFBotCollectNearMoney_SelectTarget
func SelectTarget(client int32) bool {
	threat := engine.NextBotOf(client).Vision().PrimaryKnownThreat(false)

	// Not with an active threat around
	if threat != 0 {
		return false
	}

	engine.SetCurrencyPack(client, engine.NearestCurrencyPack(client, 6000.0))

	return engine.CurrencyPackOf(client) != -1
}
