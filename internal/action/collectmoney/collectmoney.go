/*
Package collectmoney is source/redbots3/behavior/collectmoney.sp.

Walking to a money pack and picking it up. Thirteenth behaviour across, and the
one that owns the state collectnearmoney reaches: m_iCurrencyPack is declared
here, so it comes across with this file.

//sp:action DefenderCollectMoney CTFBotCollectMoney
*/
package collectmoney

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

/*
How long a pack has left before it is worth crossing the map for, and what that
is worth.

Nearest first, because at the end of a wave the money is in a heap where the
last robot died and the whole point is to clear the heap. It used to be
soonest-to-vanish first, and a pack with more than thirty seconds left was not a
candidate at all: freshly dropped cash has its whole lifetime in front of it, so
nothing on the floor qualified until the last thirty seconds of it, by which
time everybody was stood at the front. Big stacks walked past, reported from
play.

One about to go still jumps the queue, priced as a discount on the walk rather
than as a rule of its own, so the bot nearest to it is still the one that goes.
*/
const (
	//sp:name MONEY_URGENT_TIME
	urgentTime = 15.0
	//sp:name MONEY_URGENT_WORTH
	urgentWorth = 3000.0
)

// Bounded, because this is asked every frame by the gate below and a heap of
// cash is a heap of entities.
//
//sp:name MONEY_ASK_INTERVAL
const askInterval = 0.3

//sp:name m_iCurrencyPack
var currencyPack [Slots]int32

//sp:name m_ctMoneyAsk
var moneyAsk [Slots]float32

// OnStart aims the path and picks a pack.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	SelectCurrencyPack(actor)

	return engine.Continue()
}

// Update walks to it.
func Update(actor int32) engine.Outcome {
	// TODO: if we're not a scout, see if we should attack instead if we have an active threat

	if !IsValidCurrencyPack(currencyPack[actor]) {
		return engine.Done("No credits to collect")
	}

	myBot := engine.NextBotOf(actor)

	if engine.RepathTime(actor) <= engine.GameTime() {
		engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(1.0, 2.0))
		engine.RepathToPos(actor, myBot, engine.WorldSpaceCenter(currencyPack[actor]))
	}

	engine.PathOf(actor).Update(myBot)

	return engine.Continue()
}

// OnEnd forgets the pack.
func OnEnd(actor int32) {
	currencyPack[actor] = -1
}

// TimeUntilRemoved is how long the pack has left.
//
//sp:name GetTimeUntilRemoved
func TimeUntilRemoved(powerup int32) float32 {
	return engine.EntityOf(powerup).NextThink("PowerupRemoveThink") - engine.GameTime()
}

// IsCurrencyPackClaimed says whoever else is already walking at this one, so a
// heap is shared out instead of raced for.
//
//sp:name IsCurrencyPackClaimed
func IsCurrencyPackClaimed(actor int32, pack int32) bool {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor || !engine.IsClientInGame(i) {
			continue
		}

		if currencyPack[i] == pack {
			return true
		}
	}

	return false
}

// SelectCurrencyPack picks the cheapest pack to walk to, with a discount for
// one about to vanish.
//
//sp:name SelectCurrencyPack
func SelectCurrencyPack(actor int32) int32 {
	// The held pack is re-asked on its own interval; losing it is what forces a fresh look
	if IsValidCurrencyPack(currencyPack[actor]) && moneyAsk[actor] > engine.GameTime() {
		return currencyPack[actor]
	}

	moneyAsk[actor] = engine.GameTime() + askInterval

	iBestPack := engine.InvalidEntReference()
	flBestCost := float32(-1.0)

	myOrigin := engine.AbsOriginOf(actor)

	x := engine.InvalidEntReference()

	for {
		x = engine.FindEntityByClassname(x, "item_currency*")
		if x == -1 {
			break
		}
		bDistributed := engine.EntProp(x, engine.PropSend(), "m_bDistributed") != 0

		if bDistributed {
			continue
		}

		if engine.EntityFlags(x)&engine.FlagOnGround() == 0 {
			continue
		}

		if IsCurrencyPackClaimed(actor, x) {
			continue
		}

		flCost := engine.VectorDistance(myOrigin, engine.WorldSpaceCenter(x))

		if TimeUntilRemoved(x) < urgentTime {
			flCost -= urgentWorth
		}

		if flBestCost < 0.0 || flCost < flBestCost {
			flBestCost = flCost
			iBestPack = x
		}
	}

	currencyPack[actor] = iBestPack
	return iBestPack
}

// IsValidCurrencyPack says the entity is still a money pack.
//
// The last two lines could be one return of a comparison. They are two because
// the shipped file is two, and a port that tidies as it goes cannot be diffed
// against what it replaces.
//
//nolint:staticcheck // S1008: kept as shipped
//sp:name IsValidCurrencyPack
func IsValidCurrencyPack(pack int32) bool {
	if !engine.IsValidEntity(pack) {
		return false
	}

	class := engine.EntityClassname(pack)

	if engine.StrContains(class, "item_currency", false) == -1 {
		return false
	}

	return true
}

// IsPossible says whether collecting is worth doing.
//
//sp:name CTFBotCollectMoney_IsPossible
func IsPossible(actor int32) bool {
	/* One of them in a wave, all of them in the break

	Mid-wave the money is a distraction from the robots walking a bomb up the map, so one goes and
	the rest keep shooting. Between waves there is nothing else to do with the time, and one bot
	clearing a heap on his own does not finish before the break does. */
	if engine.RoundState() != engine.RoundStateBetweenRounds() &&
		engine.CountOfBotsWithNamedAction("DefenderCollectMoney") > 0 {
		return false
	}

	if !IsValidCurrencyPack(SelectCurrencyPack(actor)) {
		return false
	}

	return true
}
