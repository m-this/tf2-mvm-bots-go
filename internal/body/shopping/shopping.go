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

/*
The upgrades this bot would like, best first.

A row rather than a handle. The shipped file made a JSONObject per candidate,
pushed it into a JSONArray, and allocated another handle to read each one back:
its own sort comment recorded twenty thousand handles per bot for a list of a
hundred and fifty. A candidate is five numbers, so it is five cells of one
ArrayList, and the sort is one SortCustom over it rather than a second list and
a rebuilt array.
*/

// MaxInt and MinInt are the bounds the tie-break draws between, as the plugin
// declared them: its own pair at 99999999, not the cell's bounds.
//
//sp:name MAX_INT
const MaxInt = 99999999

// MinInt is the lower one.
//
//sp:name MIN_INT
const MinInt = -99999999

// The cells of one candidate row.
const (
	rowClass    = 0
	rowSlot     = 1
	rowIndex    = 2
	rowRandom   = 3
	rowPriority = 4
	rowCells    = 5
)

//sp:name CTFPlayerUpgrades
var playerUpgrades [Slots]engine.List

/*
When this bot may buy again, how much it has bought, and when the window shuts.

They sit here rather than in the shopping behaviour because the generated code
is the first to read them, and a global has to be declared before its first
reader.
*/

//nolint:unused // emitted, not read from Go: botreset reaches them by slot
//sp:name m_flNextUpgrade
var nextUpgrade [Slots]float32

//sp:name m_nPurchasedUpgrades
var purchasedUpgrades [Slots]int32

//nolint:unused // emitted, not read from Go: botreset reaches them by slot
//sp:name m_flUpgradingTime
var upgradingTime [Slots]float32

/*
GetUpgradePriority is what one candidate is worth to this bot.

The name becomes a number once, here, and the three tables switch on it.
generated/upgrade_rank.sp is written from internal/upgrade/table.go, which holds
the scores this used to compare ninety-four strings to reach. ATTRIBUTE_NONE is
what a name the table does not rank becomes, and every table below falls through
it to the general one, which is what the comparison chain did with a name it did
not recognise.
*/
//
//sp:name GetUpgradePriority
func GetUpgradePriority(client int32, slot int32, index int32, pclass engine.Class) int32 {
	// A canteen is worth less to a bot than anything it can shoot with.
	if slot == engine.LoadoutSlotAction() {
		return -10
	}

	upgrade := engine.UpgradeByIndex(index)

	// Nothing to rank it on, so rank it the way the mod used to rank
	// everything.
	if upgrade == engine.NoAddress() {
		return engine.UnrankedUpgradePriority()
	}

	attribute := engine.UpgradeAttribute(upgrade)

	if attribute[0] == 0 {
		return engine.UnrankedUpgradePriority()
	}

	// An upgrade to something the bot is not carrying is credits set on fire.
	if engine.IsUpgradeWasted(client, attribute) {
		return -10
	}

	id := engine.AttributeID(attribute)

	priority := int32(0)

	// The metal upgrades do not hang off the gun, so they are asked before
	// the slot is.
	if engine.PlayerClass(client) == engine.ClassEngineer() && engine.EngineerGunSpendsMetal(client) {
		priority = engine.UpgradeRankEngineerMetal(id)
	}

	if priority <= 0 && slot >= engine.LoadoutSlotPrimary() && slot <= engine.LoadoutSlotMelee() {
		weapon := engine.PlayerWeaponSlot(client, slot)

		if weapon > 0 && engine.HasEntProp(weapon, engine.PropSend(), "m_iItemDefinitionIndex") {
			priority = engine.UpgradeRankLoadout(engine.EntProp(weapon, engine.PropSend(), "m_iItemDefinitionIndex"), id)
		}
	}

	if priority > 0 {
		return priority
	}

	priority = engine.UpgradeRankClass(pclass, slot, id)

	if priority > 0 {
		return priority
	}

	return engine.UpgradeRankGeneral(id)
}

/*
SortUpgradesHighestFirst orders the candidates, dearest first.

The random cell is written and not read, which is mvm-z83.76: the tie-break it
was for was never wired up, and the port keeps it that way.
*/
//
//sp:name SortUpgradesHighestFirst
//
//nolint:revive // unused-parameter: SourceMod hands a comparator both handles
func SortUpgradesHighestFirst(index1 int32, index2 int32, array engine.Handle, hndl engine.Handle) int32 {
	list := engine.ListOf(array)

	first := list.GetAt(index1, rowPriority)
	second := list.GetAt(index2, rowPriority)

	if first > second {
		return -1
	}

	return engine.ChooseInt(first < second, 1, 0)
}

/*
CollectUpgrades builds the list of everything this bot could buy, best first.

The slots are chosen by class first, because most of what the station offers is
worth nothing to most classes. The engineer gets the most: his melee, his PDAs,
and both guns, because a Widowmaker spends the metal the sentry needs and a
Rescue Ranger had a rule nothing could reach. The sentry still outranks all of
it, so the gun is bought with what is left.
*/
//
//sp:name CollectUpgrades
//
//nolint:gocritic // ifElseChain: the shipped file is a chain, and a switch would be a different file
func CollectUpgrades(client int32) {
	if playerUpgrades[client] != engine.NoList() {
		playerUpgrades[client].Close()
	}

	playerUpgrades[client] = engine.NewBlocks(rowCells)

	iArraySlots := engine.NewList()
	defer iArraySlots.Close()

	// Always buy player upgrades.
	iArraySlots.Push(-1)

	bDemoKnight := engine.PlayerWeaponSlot(client, engine.WeaponSlotPrimary()) == -1
	bEngineer := engine.PlayerClass(client) == engine.ClassEngineer()

	if bEngineer {
		iArraySlots.Push(engine.LoadoutSlotMelee())
		iArraySlots.Push(engine.LoadoutSlotBuilding())
		iArraySlots.Push(engine.LoadoutSlotPDA())

		iArraySlots.Push(engine.LoadoutSlotPrimary())
		iArraySlots.Push(engine.LoadoutSlotSecondary())
	} else {
		if engine.PlayerClass(client) == engine.ClassSniper() {
			iArraySlots.Push(engine.LoadoutSlotPrimary())
			iArraySlots.Push(engine.LoadoutSlotMelee())
		} else if engine.PlayerClass(client) == engine.ClassMedic() {
			// Buy upgrades for our medigun.
			iArraySlots.Push(engine.LoadoutSlotSecondary())
		} else if engine.PlayerClass(client) == engine.ClassSpy() {
			// Buy upgrades for our sapper and knife.
			iArraySlots.Push(engine.LoadoutSlotBuilding())
			iArraySlots.Push(engine.LoadoutSlotMelee())
		}

		// A demoknight does not buy primary weapon upgrades.
		iArraySlots.Push(engine.ChooseInt(bDemoKnight, engine.LoadoutSlotMelee(), engine.LoadoutSlotPrimary()))

		if engine.IsShieldEquipped(client) {
			iArraySlots.Push(engine.LoadoutSlotSecondary())
		} else {
			secondary := engine.PlayerWeaponSlot(client, engine.WeaponSlotSecondary())
			weaponID := engine.ChooseWeapon(secondary != -1, engine.WeaponID(secondary), engine.WeaponNone())

			switch weaponID {
			case engine.WeaponJar(), engine.WeaponJarMilk(), engine.WeaponBuffItem(), engine.WeaponJarGas():
				// Secondary items that have some use.
				iArraySlots.Push(engine.LoadoutSlotSecondary())
			case engine.WeaponPipebombLauncher():
				// With no actual primary, the secondary is what it relies on.
				if bDemoKnight {
					iArraySlots.Push(engine.LoadoutSlotSecondary())
				}
			}
		}
	}

	for i := int32(0); i < iArraySlots.Length(); i++ {
		slot := iArraySlots.Get(i)

		upgradeCount := engine.UpgradeCount()

		for index := int32(0); index < upgradeCount; index++ {
			upgrade := engine.UpgradeByIndex(index)

			if engine.UpgradeUIGroup(upgrade) == engine.UIGroupAttachedToPlayer() && slot != -1 {
				continue
			}

			/* Canteens are not bought at all

			The player slot takes every upgrade the game does not attach to a
			weapon, which sweeps up the powerup bottle charges too. The game
			refuses those on slot -1, the bot pays nothing, and the next
			interval picks the same charge again for as long as the upgrade
			window lasts. See CTFBotUpgrade_OnEnd for why the leftovers stay
			in the wallet instead. */
			if engine.UpgradeUIGroup(upgrade) == engine.UIGroupPowerupBottle() {
				continue
			}

			attr := engine.AttributeDefinitionByName(engine.UpgradeAttribute(upgrade))

			if attr == engine.NoAddress() {
				continue
			}

			if !engine.CanUpgradeWithAttrib(client, slot, engine.AttributeDefinitionIndex(attr), upgrade) {
				continue
			}

			pclass := engine.PlayerClass(client)

			row := playerUpgrades[client].PushAt(int32(pclass))
			playerUpgrades[client].SetAt(row, slot, rowSlot)
			playerUpgrades[client].SetAt(row, index, rowIndex)
			playerUpgrades[client].SetAt(row, engine.RandomInt(MinInt, MaxInt), rowRandom)
			playerUpgrades[client].SetAt(row, GetUpgradePriority(client, slot, index, pclass), rowPriority)
		}
	}

	playerUpgrades[client].SortCustom(SortUpgradesHighestFirst)

	if engine.DebugActions().Bool() {
		engine.PrintToServer("\nPreferred upgrades for #%d \"%N\"\n", client, client)
		engine.PrintToServer("%3s %4s %4s %5s %-64s\n", "#", "SLOT", "COST", "INDEX", "ATTRIBUTE")

		for i := int32(0); i < playerUpgrades[client].Length(); i++ {
			index := playerUpgrades[client].GetAt(i, rowIndex)
			slot := playerUpgrades[client].GetAt(i, rowSlot)
			pclass := playerUpgrades[client].GetAt(i, rowClass)

			cost := engine.CostForUpgrade(engine.UpgradeByIndex(index), slot, pclass, client)

			engine.PrintToServer("%3d %4d %4d %5d %-64s", i, slot, cost, index, engine.UpgradeAttribute(engine.UpgradeByIndex(index)))
		}
	}
}

// The upgrades the game turned down this trip, so asking again does not spend
// the window on the same no.
//
//sp:name m_bRefusedUpgrade
var refusedUpgrade [Slots][128]bool

/*
ChooseUpgrade is the best thing this bot can still afford, as a row index, and
-1 when there is nothing left worth buying.

The list is built once a session, not once a purchase. It used to be rebuilt and
re-sorted every time a bot bought anything, which is every 0.1 to 1.25 seconds
each, and six bots at the station did it several times a second between them. It
bought nothing: everything the rebuild filtered on, the walk below re-asks per
entry anyway, and a rebuild can only ever remove entries because buying an
upgrade does not make another available.
*/
//
//sp:name CTFBotPurchaseUpgrades_ChooseUpgrade
func ChooseUpgrade(actor int32) int32 {
	currency := engine.Currency(actor)

	if playerUpgrades[actor] == engine.NoList() || playerUpgrades[actor].Length() == 0 {
		CollectUpgrades(actor)
	}

	for i := int32(0); i < playerUpgrades[actor].Length(); i++ {
		index := playerUpgrades[actor].GetAt(i, rowIndex)
		slot := playerUpgrades[actor].GetAt(i, rowSlot)
		pclass := playerUpgrades[actor].GetAt(i, rowClass)

		upgrade := engine.UpgradeByIndex(index)

		if upgrade == engine.NoAddress() {
			if engine.DebugActions().Bool() {
				engine.PrintToServer("CMannVsMachineUpgrades is NULL")
			}

			return -1
		}

		attr := engine.AttributeDefinitionByName(engine.UpgradeAttribute(upgrade))

		if attr == engine.NoAddress() {
			continue
		}

		// Already refused this trip, so asking again spends the window on the
		// same no.
		if refusedUpgrade[actor][index] {
			continue
		}

		if !engine.CanUpgradeWithAttrib(actor, slot, engine.AttributeDefinitionIndex(attr), upgrade) {
			continue
		}

		iCost := engine.CostForUpgrade(upgrade, slot, pclass, actor)

		// This one has had its share of the wallet already.
		if !WithinAttributeShare(actor, index, iCost) {
			continue
		}

		if iCost > currency {
			continue
		}

		/* A negative priority is a refusal, not a low bid

		It used to be only a bid, so once everything worth having was maxed or
		unaffordable the bot worked down the list and bought whatever was left.
		Reported as Pyros buying Airblast Pushback Scale, which is in the
		canteen slot and was ranked at minus ten for exactly that reason.
		Ranking it last is not the same as never buying it. */
		if GetUpgradePriority(actor, slot, index, engine.Class(pclass)) < 0 {
			continue
		}

		tier := engine.UpgradeTier(index)

		if tier != 0 {
			if !engine.IsUpgradeTierEnabled(actor, slot, tier) {
				continue
			}
		}

		return i
	}

	return -1
}

/*
PurchaseUpgrade buys every tier of one upgrade the bot can afford, in one go.

The game takes a count and applies it, refusing each step it cannot. Buying one
step per interval instead is what a play-test heard as a bot announcing the same
upgrade over and over, because each step is announced and four steps of one
attribute all read the same.

Nothing about what gets bought changes. The list is a strict priority and the top
of it stays the top until it maxes out, so the steps bought here in one go are
the ones the next four intervals would have bought anyway.
*/
//
//sp:name CTFBotPurchaseUpgrades_PurchaseUpgrade
func PurchaseUpgrade(actor int32, row int32) bool {
	slot := playerUpgrades[actor].GetAt(row, rowSlot)
	index := playerUpgrades[actor].GetAt(row, rowIndex)
	pclass := playerUpgrades[actor].GetAt(row, rowClass)

	cost := engine.CostForUpgrade(engine.UpgradeByIndex(index), slot, pclass, actor)
	currencyBefore := engine.Currency(actor)

	count := int32(1)

	if cost > 0 {
		upgrade := engine.UpgradeByIndex(index)

		tiers := engine.UpgradeTiersMax()

		if upgrade != engine.NoAddress() {
			tiers = engine.UpgradeTierCap(engine.UpgradeAttribute(upgrade))
		}

		count = currencyBefore / cost

		if count > tiers {
			count = tiers
		}

		if count < 1 {
			count = 1
		}
	}

	engine.BuyUpgrade(actor, count, slot, index)

	spent := currencyBefore - engine.Currency(actor)

	// The credits never moved, which is the game turning the purchase down.
	if cost > 0 && spent <= 0 {
		return false
	}

	// An upgrade that costs nothing cannot be counted this way, so it counts
	// as one.
	purchasedUpgrades[actor] += engine.ChooseInt(cost > 0, spent/cost, 1)

	if index >= 0 && index < engine.MaxUpgrades() {
		spentOnUpgrade[actor][index] += spent
	}

	return true
}

// RowIndexOf is the upgrade index a chosen row names, which the caller needs to
// remember a refusal by.
//
//sp:name Go_RowIndexOf
func RowIndexOf(actor int32, row int32) int32 {
	return playerUpgrades[actor].GetAt(row, rowIndex)
}

// SetRefusedUpgrade remembers that the game turned one down.
//
//sp:name Go_SetRefusedUpgrade
func SetRefusedUpgrade(actor int32, index int32) {
	refusedUpgrade[actor][index] = true
}
