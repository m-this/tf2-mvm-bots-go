package engine

/*
What the tactical monitor reaches.

It is the busiest of the game's own behaviours and the mod overrides all of it,
so this is the widest seam in the file: the game's internal countdown timers,
the four behaviours it can hand off to, and the health and ammo questions that
decide which.
*/

// TacticalCalls are the answers.
type TacticalCalls struct {
	CountdownAt              func(address Address) Countdown
	CountdownAddress         func(c Countdown) Address
	CountdownStart           func(c Countdown, duration float32)
	OpportunisticTimer       func(client int32) Address
	EvadeBusterIsPossible    func(actor int32) bool
	EvadeBuster              func() Behaviour
	SpyCheckIsPossible       func(actor int32) bool
	SpyCheck                 func() Behaviour
	GetHealthIsPossible      func(actor int32) bool
	GetHealth                func() Behaviour
	GetAmmoIsPossible        func(actor int32) bool
	GetAmmo                  func() Behaviour
	ShouldDetonateStickies   func(actor int32) bool
	UpdateSpyIntel           func(actor int32)
	HealthRatio              func(actor int32) float32
	UpdateDefenderReadiness  func(actor int32)
	UpdateStuckWatchdog      func(actor int32)
	UpdateScoutCombatJump    func(client int32)
	ShouldUseTeleporterNow   func(client int32) bool
	ShouldLeaveToBePatchedUp func(client int32, healthRatio float32) bool
	IsAmmoLowNow             func(client int32) bool
}

var tacticals TacticalCalls

// InstallTacticals puts a set of answers behind them.
func InstallTacticals(c TacticalCalls) func() {
	previous := tacticals
	tacticals = c
	return func() { tacticals = previous }
}

/*
Countdown is the game's own CountdownTimer, reached by address.

Starting one is how the mod says no to a decision the game is about to make:
CTFBotTacticalMonitor::FindNearbyTeleporter returns null while its timer is
running, so a started timer is a refusal with no hook of its own.
*/
//
//sp:tag CountdownTimer
type Countdown int32

// CountdownAt is the timer at that address.
//
//sp:native CountdownTimer
func CountdownAt(address Address) Countdown {
	if tacticals.CountdownAt == nil {
		missing("CountdownTimer")
	}
	return tacticals.CountdownAt(address)
}

// Address says the timer is really there, which an offset that has moved is
// not.
//
//sp:property Address
func (c Countdown) Address() Address {
	if tacticals.CountdownAddress == nil {
		missing("CountdownTimer.Address")
	}
	return tacticals.CountdownAddress(c)
}

// Start runs it for that long.
//
//sp:method Start
func (c Countdown) Start(duration float32) {
	if tacticals.CountdownStart == nil {
		missing("CountdownTimer.Start")
	}
	tacticals.CountdownStart(c, duration)
}

// NoAddress is Address_Null.
//
//sp:global Address_Null
func NoAddress() Address { return 0 }

// FindTeleporterOffset is where CTFBotTacticalMonitor keeps the timer that
// gates its teleporter search.
//
//sp:global 0x70
func FindTeleporterOffset() int32 { return 0x70 }

// OpportunisticTimer is the address of the game's own opportunistic timer for
// that bot.
//
//sp:plugin GetOpportunisticTimer
func OpportunisticTimer(client int32) Address {
	if tacticals.OpportunisticTimer == nil {
		missing("GetOpportunisticTimer")
	}
	return tacticals.OpportunisticTimer(client)
}

// EvadeBusterIsPossible says a sentry buster is close enough to run from.
// Ported, evadebuster.
//
//sp:body CTFBotEvadeBuster_IsPossible
func EvadeBusterIsPossible(actor int32) bool {
	if tacticals.EvadeBusterIsPossible == nil {
		missing("CTFBotEvadeBuster_IsPossible")
	}
	return tacticals.EvadeBusterIsPossible(actor)
}

// EvadeBuster is CTFBotEvadeBuster. Ported, evadebuster.
//
//sp:body CTFBotEvadeBuster
func EvadeBuster() Behaviour {
	if tacticals.EvadeBuster == nil {
		missing("CTFBotEvadeBuster")
	}
	return tacticals.EvadeBuster()
}

// SpyCheckIsPossible says somebody is worth frisking. Ported, spycheck.
//
//sp:body CTFBotSpyCheck_IsPossible
func SpyCheckIsPossible(actor int32) bool {
	if tacticals.SpyCheckIsPossible == nil {
		missing("CTFBotSpyCheck_IsPossible")
	}
	return tacticals.SpyCheckIsPossible(actor)
}

// SpyCheck is CTFBotSpyCheck. Ported, spycheck.
//
//sp:body CTFBotSpyCheck
func SpyCheck() Behaviour {
	if tacticals.SpyCheck == nil {
		missing("CTFBotSpyCheck")
	}
	return tacticals.SpyCheck()
}

// GetHealthIsPossible says there is a pack worth walking to. Ported, gethealth.
//
//sp:body CTFBotGetHealth_IsPossible
func GetHealthIsPossible(actor int32) bool {
	if tacticals.GetHealthIsPossible == nil {
		missing("CTFBotGetHealth_IsPossible")
	}
	return tacticals.GetHealthIsPossible(actor)
}

// GetHealth is CTFBotGetHealth. Ported, gethealth.
//
//sp:body CTFBotGetHealth
func GetHealth() Behaviour {
	if tacticals.GetHealth == nil {
		missing("CTFBotGetHealth")
	}
	return tacticals.GetHealth()
}

// GetAmmoIsPossible is the same for ammo. Ported, getammo.
//
//sp:body CTFBotGetAmmo_IsPossible
func GetAmmoIsPossible(actor int32) bool {
	if tacticals.GetAmmoIsPossible == nil {
		missing("CTFBotGetAmmo_IsPossible")
	}
	return tacticals.GetAmmoIsPossible(actor)
}

// GetAmmo is CTFBotGetAmmo. Ported, getammo.
//
//sp:body CTFBotGetAmmo
func GetAmmo() Behaviour {
	if tacticals.GetAmmo == nil {
		missing("CTFBotGetAmmo")
	}
	return tacticals.GetAmmo()
}

// ShouldDetonateStickies says the trap is worth blowing now. Ported, stickies.
//
//sp:body ShouldDetonateStickies
func ShouldDetonateStickies(actor int32) bool {
	if tacticals.ShouldDetonateStickies == nil {
		missing("ShouldDetonateStickies")
	}
	return tacticals.ShouldDetonateStickies(actor)
}

// UpdateSpyIntel is what the bot can honestly say it has seen of a spy.
// Ported, spycheck.
//
//sp:body UpdateSpyIntel
func UpdateSpyIntel(actor int32) {
	if tacticals.UpdateSpyIntel == nil {
		missing("UpdateSpyIntel")
	}
	tacticals.UpdateSpyIntel(actor)
}

// HealthRatio is the fraction of maximum health left. Ported, state.
//
//sp:body HealthRatio
func HealthRatio(actor int32) float32 {
	if tacticals.HealthRatio == nil {
		missing("HealthRatio")
	}
	return tacticals.HealthRatio(actor)
}

// UpdateDefenderReadiness is the ready screen. Ported, readiness.
//
//sp:body UpdateDefenderReadiness
func UpdateDefenderReadiness(actor int32) {
	if tacticals.UpdateDefenderReadiness == nil {
		missing("UpdateDefenderReadiness")
	}
	tacticals.UpdateDefenderReadiness(actor)
}

// UpdateStuckWatchdog is the watch. Ported, stuckwatch.
//
//sp:body UpdateStuckWatchdog
func UpdateStuckWatchdog(actor int32) {
	if tacticals.UpdateStuckWatchdog == nil {
		missing("UpdateStuckWatchdog")
	}
	tacticals.UpdateStuckWatchdog(actor)
}

// UpdateScoutCombatJump is the dodge. Ported, scoutjump.
//
//sp:body UpdateScoutCombatJump
func UpdateScoutCombatJump(client int32) {
	if tacticals.UpdateScoutCombatJump == nil {
		missing("UpdateScoutCombatJump")
	}
	tacticals.UpdateScoutCombatJump(client)
}

// ShouldUseTeleporterNow is the ride question. Ported, botqueries.
//
//sp:body ShouldUseTeleporter
func ShouldUseTeleporterNow(client int32) bool {
	if tacticals.ShouldUseTeleporterNow == nil {
		missing("ShouldUseTeleporter")
	}
	return tacticals.ShouldUseTeleporterNow(client)
}

// ShouldLeaveToBePatchedUp is the medic exception. Ported, readiness.
//
//sp:body ShouldLeaveToBePatchedUp
func ShouldLeaveToBePatchedUp(client int32, healthRatio float32) bool {
	if tacticals.ShouldLeaveToBePatchedUp == nil {
		missing("ShouldLeaveToBePatchedUp")
	}
	return tacticals.ShouldLeaveToBePatchedUp(client, healthRatio)
}

// IsAmmoLowNow is the ammo question. Ported, botqueries.
//
//sp:body IsAmmoLow
func IsAmmoLowNow(client int32) bool {
	if tacticals.IsAmmoLowNow == nil {
		missing("IsAmmoLow")
	}
	return tacticals.IsAmmoLowNow(client)
}

// HealthOkRatio is tf_bot_health_ok_ratio.
//
//sp:global tf_bot_health_ok_ratio
func HealthOkRatio() ConVar { return 0 }
