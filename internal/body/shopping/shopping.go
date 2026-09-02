/*
Package shopping is the parts of source/redbots3/behavior/upgrade.sp that do
not touch the candidate list: what is left to build, what a bot may still spend
on one attribute, how fast it presses the button, and what it says when it
stops.

The candidate list itself is mvm-z83.64, and needs a shape the JSONObject per
row does not have.
*/
package shopping

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// UpgradeAttributeShare is the fraction of one trip's wallet any single
// attribute may take.
//
//sp:name UPGRADE_ATTRIBUTE_SHARE
const UpgradeAttributeShare = 0.5

// What this bot had to spend when the trip started.
//
//sp:name m_iSessionWallet
var sessionWallet [Slots]int32

// What it has put into each attribute so far.
//
//sp:name m_iSpentOnUpgrade
var spentOnUpgrade [Slots][128]int32

/*
NothingLeftToBuild says the engineer has nothing further to do to this
building.

A building still going up is not finished, and neither is one below level three.
A mini is: it cannot be upgraded at all, which is the whole point of it.
*/
//
//sp:name NothingLeftToBuild
func NothingLeftToBuild(building int32) bool {
	if building == engine.InvalidEntReference() || !engine.IsValidEntity(building) {
		return false
	}

	if engine.IsBuildingUp(building) {
		return false
	}

	return engine.IsMiniBuilding(building) || engine.UpgradeLevel(building) >= 3
}

/*
WithinAttributeShare says this purchase leaves the trip balanced.

Without it a bot spends its whole wallet on the first attribute the ranking
likes, which on a good roll is one very sharp weapon and nothing else. The floor
of twice the cost is there so a cheap attribute is never refused outright: half
of a small wallet can be less than one step.
*/
//
//sp:name WithinAttributeShare
func WithinAttributeShare(client int32, index int32, cost int32) bool {
	if index < 0 || index >= engine.MaxUpgrades() {
		return true
	}

	allowed := engine.RoundToNearest(float32(sessionWallet[client]) * UpgradeAttributeShare)

	if allowed < cost*2 {
		allowed = cost * 2
	}

	return spentOnUpgrade[client][index]+cost <= allowed
}

/*
LogUpgradeSessionEnd says why the trip stopped, and what the wave looked like
while it was spending.

A resistance is ranked at 210 when the coming wave deals that damage and 25 when
it does not, so the three answers decide whether a resistance was ever worth
buying on this trip. Reported from play as "the bots really need to buy
resistance upgrades", and the ranking was already there and already ahead of
most of the table: what nobody could see was whether WaveHasClassIcon had
anything to read. tf_objective_resource carries the wave bar, and a trip that
shops before the game has filled it in sees an empty wave and prices every
resistance at nothing.
*/
//
//sp:name LogUpgradeSessionEnd
//sp:const why
func LogUpgradeSessionEnd(actor int32, why engine.Text) {
	engine.LogMessage("Shopping: %N stopped, %s, %d credits left, wave deals blast=%d bullet=%d fire=%d",
		actor, why, engine.Currency(actor),
		engine.WaveHasExplosiveRobots(), engine.WaveHasBulletRobots(), engine.WaveHasFireRobots())
}

/*
GetUpgradeInterval is how long a bot waits between purchases.

Fast during a round, because a bot shopping mid-wave is a bot not fighting, and
unhurried between them. The spread is so six bots at one station do not press
the button on the same frame.
*/
//
//sp:name GetUpgradeInterval
func GetUpgradeInterval() float32 {
	customInterval := engine.UpgradeInterval().Float()

	if customInterval >= 0.0 {
		return customInterval
	}

	// Upgrading during an active round, buy upgrades fast.
	if engine.RoundState() == engine.RoundStateRunning() {
		return engine.RandomFloat(0.1, 0.75)
	}

	const interval = 1.25
	const variance = 0.3

	return engine.RandomFloat(interval-variance, interval+variance)
}
