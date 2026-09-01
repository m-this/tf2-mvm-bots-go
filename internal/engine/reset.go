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
	resets = c
	return func() { resets = previous }
}

// ResetMarkGiant is markgiant's own. Ported, markgiant.
//
//sp:body Go_ResetMarkGiant
func ResetMarkGiant(client int32) {
	if resets.ResetMarkGiant == nil {
		missing("Go_ResetMarkGiant")
	}
	resets.ResetMarkGiant(client)
}

// ResetGotoUpgrade is gotoupgrade's own. Ported, gotoupgrade.
//
//sp:body Go_ResetGotoUpgrade
func ResetGotoUpgrade(client int32) {
	if resets.ResetGotoUpgrade == nil {
		missing("Go_ResetGotoUpgrade")
	}
	resets.ResetGotoUpgrade(client)
}

// ResetGetAmmo is getammo's own. Ported, getammo.
//
//sp:body Go_ResetGetAmmo
func ResetGetAmmo(client int32) {
	if resets.ResetGetAmmo == nil {
		missing("Go_ResetGetAmmo")
	}
	resets.ResetGetAmmo(client)
}

// ResetMoveToFront is movetofront's own. Ported, movetofront.
//
//sp:body Go_ResetMoveToFront
func ResetMoveToFront(client int32) {
	if resets.ResetMoveToFront == nil {
		missing("Go_ResetMoveToFront")
	}
	resets.ResetMoveToFront(client)
}

// ResetGetHealth is gethealth's own. Ported, gethealth.
//
//sp:body Go_ResetGetHealth
func ResetGetHealth(client int32) {
	if resets.ResetGetHealth == nil {
		missing("Go_ResetGetHealth")
	}
	resets.ResetGetHealth(client)
}

// ResetSpySap is spysap's own. Ported, spysap.
//
//sp:body Go_ResetSpySap
func ResetSpySap(client int32) {
	if resets.ResetSpySap == nil {
		missing("Go_ResetSpySap")
	}
	resets.ResetSpySap(client)
}

// ResetSpySapPlayer is spysapplayer's own. Ported, spysapplayer.
//
//sp:body Go_ResetSpySapPlayer
func ResetSpySapPlayer(client int32) {
	if resets.ResetSpySapPlayer == nil {
		missing("Go_ResetSpySapPlayer")
	}
	resets.ResetSpySapPlayer(client)
}

// ResetAttackForUber is attackforuber's own. Ported, attackforuber.
//
//sp:body Go_ResetAttackForUber
func ResetAttackForUber(client int32) {
	if resets.ResetAttackForUber == nil {
		missing("Go_ResetAttackForUber")
	}
	resets.ResetAttackForUber(client)
}

// ResetAttackTank is attacktank's own. Ported, attacktank.
//
//sp:body Go_ResetAttackTank
func ResetAttackTank(client int32) {
	if resets.ResetAttackTank == nil {
		missing("Go_ResetAttackTank")
	}
	resets.ResetAttackTank(client)
}

// ResetDestroyTeleporter is destroyteleporter's own. Ported,
// destroyteleporter.
//
//sp:body Go_ResetDestroyTeleporter
func ResetDestroyTeleporter(client int32) {
	if resets.ResetDestroyTeleporter == nil {
		missing("Go_ResetDestroyTeleporter")
	}
	resets.ResetDestroyTeleporter(client)
}

// ResetGuardPoint is guardpoint's own. Ported, guardpoint.
//
//sp:body Go_ResetGuardPoint
func ResetGuardPoint(client int32) {
	if resets.ResetGuardPoint == nil {
		missing("Go_ResetGuardPoint")
	}
	resets.ResetGuardPoint(client)
}

// ResetAttack is attack's own. Ported, attack.
//
//sp:body Go_ResetAttack
func ResetAttack(client int32) {
	if resets.ResetAttack == nil {
		missing("Go_ResetAttack")
	}
	resets.ResetAttack(client)
}

// ResetCollectMoney is collectmoney's own. Ported, collectmoney.
//
//sp:body Go_ResetCollectMoney
func ResetCollectMoney(client int32) {
	if resets.ResetCollectMoney == nil {
		missing("Go_ResetCollectMoney")
	}
	resets.ResetCollectMoney(client)
}

// ResetScoutJump is scoutjump's own. Ported, scoutjump.
//
//sp:body Go_ResetScoutJump
func ResetScoutJump(client int32) {
	if resets.ResetScoutJump == nil {
		missing("Go_ResetScoutJump")
	}
	resets.ResetScoutJump(client)
}

// ResetBottle is bottle's own. Ported, bottle.
//
//sp:body Go_ResetBottle
func ResetBottle(client int32) {
	if resets.ResetBottle == nil {
		missing("Go_ResetBottle")
	}
	resets.ResetBottle(client)
}

// ResetSpyCheck is spycheck's own. Ported, spycheck.
//
//sp:body ResetSpyCheck
func ResetSpyCheck(client int32) {
	if resets.ResetSpyCheck == nil {
		missing("ResetSpyCheck")
	}
	resets.ResetSpyCheck(client)
}

// ResetStickyTrap is stickytrap's own. Ported, stickytrap.
//
//sp:body ResetStickyTrap
func ResetStickyTrap(client int32) {
	if resets.ResetStickyTrap == nil {
		missing("ResetStickyTrap")
	}
	resets.ResetStickyTrap(client)
}

/*
The three the shopping trip still owns.

behavior/upgrade.sp is the last hand-written behaviour, mvm-z83.64, so its state
is reached the way any unported state is: by slot. They join their neighbours
the day it is ported.
*/

// SetNextUpgrade writes m_flNextUpgrade.
//
//sp:slotset m_flNextUpgrade
func SetNextUpgrade(client int32, when float32) {
	if resets.SetNextUpgrade == nil {
		missing("m_flNextUpgrade")
	}
	resets.SetNextUpgrade(client, when)
}

// SetPurchasedUpgrades writes m_nPurchasedUpgrades.
//
//sp:slotset m_nPurchasedUpgrades
func SetPurchasedUpgrades(client int32, count int32) {
	if resets.SetPurchasedUpgrades == nil {
		missing("m_nPurchasedUpgrades")
	}
	resets.SetPurchasedUpgrades(client, count)
}

// SetUpgradingTime writes m_flUpgradingTime.
//
//sp:slotset m_flUpgradingTime
func SetUpgradingTime(client int32, when float32) {
	if resets.SetUpgradingTime == nil {
		missing("m_flUpgradingTime")
	}
	resets.SetUpgradingTime(client, when)
}

// Reset puts the plugin's own pathing bot back to nothing.
//
//sp:method Reset
func (b PluginBot) Reset() {
	if resets.PluginBotReset == nil {
		missing("PluginBot.Reset")
	}
	resets.PluginBotReset(b)
}

// NewRoutePath is PathFollower where the answer is kept as the bot's own path
// rather than used and destroyed.
//
//sp:native PathFollower before _
func NewRoutePath(ignoreActors int32, onlyActors int32) Path {
	if resets.NewRoutePath == nil {
		missing("PathFollower")
	}
	return resets.NewRoutePath(ignoreActors, onlyActors)
}

// NewChasePath makes a ChasePath, which leads its subject rather than following
// where it was.
//
//sp:native ChasePath
func NewChasePath(lead int32, cost int32, ignoreActors int32, onlyActors int32) Path {
	if resets.NewChasePath == nil {
		missing("ChasePath")
	}
	return resets.NewChasePath(lead, cost, ignoreActors, onlyActors)
}

// LeadSubject is LEAD_SUBJECT, the chase that aims where the target is going.
//
//sp:global LEAD_SUBJECT
func LeadSubject() int32 { return 0 }

// SetPath writes the bot's route handle.
//
//sp:slotset m_pPath
func SetPath(actor int32, path Path) {
	if resets.SetPath == nil {
		missing("m_pPath")
	}
	resets.SetPath(actor, path)
}

// SetChasePath writes its chase route.
//
//sp:slotset m_pChasePath
func SetChasePath(actor int32, path Path) {
	if resets.SetChasePath == nil {
		missing("m_pChasePath")
	}
	resets.SetChasePath(actor, path)
}
