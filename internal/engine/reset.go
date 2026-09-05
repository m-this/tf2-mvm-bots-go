package engine

/*
Putting a seat back the way it was found.

A bot leaving takes its seat's state with it, and the next bot in that seat is a
different bot. The shipped ResetNextBot was one flat list of every field the mod
keeps per client, which is a list nobody could add to without reading the whole
mod; each package clears its own now and this is how the reset reaches them.
*/

// ResetCalls are the answers.
type ResetCalls struct {
	ResetMarkGiant         func(client int32)
	ResetGotoUpgrade       func(client int32)
	ResetGetAmmo           func(client int32)
	ResetMoveToFront       func(client int32)
	ResetGetHealth         func(client int32)
	ResetSpySap            func(client int32)
	ResetSpySapPlayer      func(client int32)
	ResetAttackForUber     func(client int32)
	ResetAttackTank        func(client int32)
	ResetDestroyTeleporter func(client int32)
	ResetBuildTeleporter   func(client int32)
	ResetGuardPoint        func(client int32)
	ResetAttack            func(client int32)
	ResetCollectMoney      func(client int32)
	ResetScoutJump         func(client int32)
	ResetBottle            func(client int32)
	ResetSpyCheck          func(client int32)
	ResetStickyTrap        func(client int32)
	SetNextUpgrade         func(client int32, when float32)
	SetPurchasedUpgrades   func(client int32, count int32)
	SetUpgradingTime       func(client int32, when float32)
	PluginBotReset         func(b PluginBot)
	NewRoutePath           func(ignoreActors int32, onlyActors int32) Path
	NewChasePath           func(lead int32, cost int32, ignoreActors int32, onlyActors int32) Path
	SetPath                func(actor int32, path Path)
	SetChasePath           func(actor int32, path Path)
}

var resets ResetCalls

// InstallResets puts a set of answers behind them.
func InstallResets(c ResetCalls) func() {
	previous := resets
	Fill(&c)
	resets = c
	return func() { resets = previous }
}

// ResetMarkGiant is markgiant's own. Ported, markgiant.
//
//sp:body Go_ResetMarkGiant
func ResetMarkGiant(client int32) { resets.ResetMarkGiant(client) }

// ResetGotoUpgrade is gotoupgrade's own. Ported, gotoupgrade.
//
//sp:body Go_ResetGotoUpgrade
func ResetGotoUpgrade(client int32) { resets.ResetGotoUpgrade(client) }

// ResetGetAmmo is getammo's own. Ported, getammo.
//
//sp:body Go_ResetGetAmmo
func ResetGetAmmo(client int32) { resets.ResetGetAmmo(client) }

// ResetMoveToFront is movetofront's own. Ported, movetofront.
//
//sp:body Go_ResetMoveToFront
func ResetMoveToFront(client int32) { resets.ResetMoveToFront(client) }

// ResetGetHealth is gethealth's own. Ported, gethealth.
//
//sp:body Go_ResetGetHealth
func ResetGetHealth(client int32) { resets.ResetGetHealth(client) }

// ResetSpySap is spysap's own. Ported, spysap.
//
//sp:body Go_ResetSpySap
func ResetSpySap(client int32) { resets.ResetSpySap(client) }

// ResetSpySapPlayer is spysapplayer's own. Ported, spysapplayer.
//
//sp:body Go_ResetSpySapPlayer
func ResetSpySapPlayer(client int32) { resets.ResetSpySapPlayer(client) }

// ResetAttackForUber is attackforuber's own. Ported, attackforuber.
//
//sp:body Go_ResetAttackForUber
func ResetAttackForUber(client int32) { resets.ResetAttackForUber(client) }

// ResetAttackTank is attacktank's own. Ported, attacktank.
//
//sp:body Go_ResetAttackTank
func ResetAttackTank(client int32) { resets.ResetAttackTank(client) }

// ResetDestroyTeleporter is destroyteleporter's own. Ported,
// destroyteleporter.
//
//sp:body Go_ResetDestroyTeleporter
func ResetDestroyTeleporter(client int32) { resets.ResetDestroyTeleporter(client) }

// ResetBuildTeleporter is engineerbuildteleporter's own: the early entrance
// flags, and nothing that was already carried between bots.
//
//sp:body Go_ResetBuildTeleporter
func ResetBuildTeleporter(client int32) { resets.ResetBuildTeleporter(client) }

// ResetGuardPoint is guardpoint's own. Ported, guardpoint.
//
//sp:body Go_ResetGuardPoint
func ResetGuardPoint(client int32) { resets.ResetGuardPoint(client) }

// ResetAttack is attack's own. Ported, attack.
//
//sp:body Go_ResetAttack
func ResetAttack(client int32) { resets.ResetAttack(client) }

// ResetCollectMoney is collectmoney's own. Ported, collectmoney.
//
//sp:body Go_ResetCollectMoney
func ResetCollectMoney(client int32) { resets.ResetCollectMoney(client) }

// ResetScoutJump is scoutjump's own. Ported, scoutjump.
//
//sp:body Go_ResetScoutJump
func ResetScoutJump(client int32) { resets.ResetScoutJump(client) }

// ResetBottle is bottle's own. Ported, bottle.
//
//sp:body Go_ResetBottle
func ResetBottle(client int32) { resets.ResetBottle(client) }

// ResetSpyCheck is spycheck's own. Ported, spycheck.
//
//sp:body ResetSpyCheck
func ResetSpyCheck(client int32) { resets.ResetSpyCheck(client) }

// ResetStickyTrap is stickytrap's own. Ported, stickytrap.
//
//sp:body ResetStickyTrap
func ResetStickyTrap(client int32) { resets.ResetStickyTrap(client) }

/*
The three the shopping trip still owns.

behavior/upgrade.sp is the last hand-written behaviour, mvm-z83.64, so its state
is reached the way any unported state is: by slot. They join their neighbours
the day it is ported.
*/

// SetNextUpgrade writes m_flNextUpgrade.
//
//sp:slotset m_flNextUpgrade
func SetNextUpgrade(client int32, when float32) { resets.SetNextUpgrade(client, when) }

// SetPurchasedUpgrades writes m_nPurchasedUpgrades.
//
//sp:slotset m_nPurchasedUpgrades
func SetPurchasedUpgrades(client int32, count int32) { resets.SetPurchasedUpgrades(client, count) }

// SetUpgradingTime writes m_flUpgradingTime.
//
//sp:slotset m_flUpgradingTime
func SetUpgradingTime(client int32, when float32) { resets.SetUpgradingTime(client, when) }

// Reset puts the plugin's own pathing bot back to nothing.
//
//sp:method Reset
func (b PluginBot) Reset() { resets.PluginBotReset(b) }

// NewRoutePath is PathFollower where the answer is kept as the bot's own path
// rather than used and destroyed.
//
//sp:native PathFollower before _
func NewRoutePath(ignoreActors int32, onlyActors int32) Path {
	return resets.NewRoutePath(ignoreActors, onlyActors)
}

// NewChasePath makes a ChasePath, which leads its subject rather than following
// where it was.
//
//sp:native ChasePath
func NewChasePath(lead int32, cost int32, ignoreActors int32, onlyActors int32) Path {
	return resets.NewChasePath(lead, cost, ignoreActors, onlyActors)
}

// LeadSubject is LEAD_SUBJECT, the chase that aims where the target is going.
//
//sp:global LEAD_SUBJECT
func LeadSubject() int32 { return 0 }

// SetPath writes the bot's route handle.
//
//sp:slotset m_pPath
func SetPath(actor int32, path Path) { resets.SetPath(actor, path) }

// SetChasePath writes its chase route.
//
//sp:slotset m_pChasePath
func SetChasePath(actor int32, path Path) { resets.SetChasePath(actor, path) }
