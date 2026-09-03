package engine

// AmmoCalls are the answers for the walk to ammo.
type AmmoCalls struct {
	IsAmmoFull      func(client int32) bool
	OnAmmoWalkStart func(client int32)
	RefuseAmmoPath  func(client int32) bool
	ListSetAt       func(l List, index int32, value int32, block int32)
}

var ammo AmmoCalls

// InstallAmmo puts a set of answers behind them.
func InstallAmmo(c AmmoCalls) func() {
	previous := ammo
	Fill(&c)
	ammo = c
	return func() { ammo = previous }
}

// AmmoSearchRange is how far a bot looks for a pack.
//
//sp:global tf_bot_ammo_search_range
func AmmoSearchRange() ConVar { return 0 }

// ConceptDispenserHere is MP_CONCEPT_PLAYER_DISPENSERHERE, which is what a bot
// walking for ammo says out loud.
//
//sp:global MP_CONCEPT_PLAYER_DISPENSERHERE
func ConceptDispenserHere() int32 { return 5 }

// FeatureAmmoFailover is the switch on walking to the next ranked pack when the
// route to this one stops existing.
//
//sp:global FEATURE_AMMO_FAILOVER
func FeatureAmmoFailover() int32 { return 19 }

// IsAmmoFull says the bot has nothing left to collect, metal included for an
// engineer.
//
//sp:body IsAmmoFull
func IsAmmoFull(client int32) bool { return ammo.IsAmmoFull(client) }

// OnAmmoWalkStart arms the injected path refusals, and does nothing unless a
// debug convar is set.
//
//sp:body DebugFaults_OnAmmoWalkStart
func OnAmmoWalkStart(client int32) { ammo.OnAmmoWalkStart(client) }

// RefuseAmmoPath is one of those refusals being handed out.
//
//sp:body DebugFaults_RefuseAmmoPath
func RefuseAmmoPath(client int32) bool { return ammo.RefuseAmmoPath(client) }

// SetAt writes one cell of a wide entry.
//
//sp:method Set
func (l List) SetAt(index int32, value int32, block int32) { ammo.ListSetAt(l, index, value, block) }
