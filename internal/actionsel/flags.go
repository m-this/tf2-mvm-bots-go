package actionsel

// Flags is every per-bot fact GetDesiredBotAction branches on, in the order it
// reads them. Each one is a call the hand-written SourcePawn makes before it
// asks for an action.
//
// Filling all fourteen eagerly is not what the plugin does: three of them have
// side effects, so the plugin's edge walks the lazy table internal/spgen emits
// and asks for one only when the walk reaches it. This struct is the shape the
// differential test feeds, where asking costs nothing and total coverage is
// the point.
type Flags struct {
	// MoneyToCollect is CTFBotCollectMoney_IsPossible.
	MoneyToCollect bool
	// InUpgradeZone is TF2_IsInUpgradeZone.
	InUpgradeZone bool
	// ShoppedThisBreak is g_bShoppedThisBreak.
	ShoppedThisBreak bool
	// MovingToFront is a DefenderMoveToFront already on the action stack.
	MovingToFront bool
	// UpgradesEnabled is redbots_manager_bot_use_upgrades.
	UpgradesEnabled bool
	// HasUpgraded is g_bHasUpgraded.
	HasUpgraded bool
	// UpgradeMidRound is ShouldUpgradeMidRound.
	UpgradeMidRound bool
	// HasSniperRifle is HasSniperRifle.
	HasSniperRifle bool
	// SniperStalled is IsSniperStalled.
	SniperStalled bool
	// AttackTargetFound is CTFBotDefenderAttack_SelectTarget. It writes
	// nothing itself but calls SelectRandomReachableEnemy, which consumes
	// randomness, so asking it when the shipped chain would not is a
	// behaviour change.
	AttackTargetFound bool
	// TankTargetFound is CTFBotAttackTank_SelectTarget, which writes
	// m_iTankTarget.
	TankTargetFound bool
	// GiantToMark is CTFBotMarkGiant_IsPossible.
	GiantToMark bool
	// NearbyMoney is CTFBotCollectNearMoney_SelectTarget, which writes
	// m_iCurrencyPack.
	NearbyMoney bool
	// StickyTrapPossible is CTFBotStickyTrap_IsPossible.
	StickyTrapPossible bool
}

// Reachable refuses the combinations the engine cannot produce, so that
// exhaustiveness is asserted over the real domain and not over the product.
// Both rifle and stall are sniper state: m_bSniperStalled is only ever set by
// the sniper stall rescue, and HasSniperRifle reads the primary slot of a
// class that has one.
func Reachable(class Class, f Flags) bool {
	if class == ClassSniper {
		return true
	}
	return !f.HasSniperRifle && !f.SniperStalled
}
