package engine

/*
What a behaviour reaches when the engine asks it about threats.

All of it is either SourceMod's or the plugin's, and the plugin's go when the
port reaches them. GetEyePosition and SelectCloserThreat return their arrays and
handles the way util.sp writes them, which is why the two shapes differ.
*/

// ThreatCalls are the answers.
type ThreatCalls struct {
	Entity                     func(known Known) int32
	SelectCloserThreat         func(bot Bot, threat1 Known, threat2 Known) Known
	IsMeleeWeapon              func(entity int32) bool
	EyePosition                func(client int32) [3]float32
	IsLineOfFireClearEntity    func(client int32, from [3]float32, who int32) bool
	CountOfBotsWithNamedAction func(name string) int32
	SpeakConceptIfAllowed      func(client int32, concept int32)
	NearestEnemyTeleporter     func(client int32, maxDistance float32) int32
}

var threats ThreatCalls

// InstallThreats puts a set of answers behind them.
func InstallThreats(c ThreatCalls) func() {
	previous := threats
	threats = c
	return func() { threats = previous }
}

// SentryMaxRange is how far a sentry shoots, which is what makes one dangerous
// on the way to somewhere else.
//
//sp:global SENTRY_MAX_RANGE
func SentryMaxRange() float32 { return 1100.0 }

// NoKnownEntity is NULL_KNOWN_ENTITY: the behaviour has no opinion to give.
//
//sp:global NULL_KNOWN_ENTITY
func NoKnownEntity() Known { return 0 }

// ObjectSentry is TFObject_Sentry.
//
//sp:global TFObject_Sentry
func ObjectSentry() Object { return 2 }

// WeaponFlamethrower is TF_WEAPON_FLAMETHROWER.
//
//sp:global TF_WEAPON_FLAMETHROWER
func WeaponFlamethrower() Weapon { return 21 }

// ConceptJeers is MP_CONCEPT_PLAYER_JEERS, which is the bot swearing at a
// teleporter it is about to hit.
//
//sp:global MP_CONCEPT_PLAYER_JEERS
func ConceptJeers() int32 { return 0 }

// Entity is what the remembered thing actually is.
//
//sp:method GetEntity
func (k Known) Entity() int32 {
	if threats.Entity == nil {
		missing("CKnownEntity.GetEntity")
	}
	return threats.Entity(k)
}

// SelectCloserThreat is whichever of the two is nearer, which is all a melee
// bot can act on.
//
//sp:plugin SelectCloserThreat
func SelectCloserThreat(bot Bot, threat1 Known, threat2 Known) Known {
	if threats.SelectCloserThreat == nil {
		missing("SelectCloserThreat")
	}
	return threats.SelectCloserThreat(bot, threat1, threat2)
}

// IsMeleeWeapon says whether the weapon only reaches what it is standing next
// to.
//
//sp:plugin IsMeleeWeapon
func IsMeleeWeapon(entity int32) bool {
	if threats.IsMeleeWeapon == nil {
		missing("IsMeleeWeapon")
	}
	return threats.IsMeleeWeapon(entity)
}

// EyePosition is where the client is looking from. Its SourcePawn returns the
// array.
//
//sp:plugin GetEyePosition returns
func EyePosition(client int32) [3]float32 {
	if threats.EyePosition == nil {
		missing("GetEyePosition")
	}
	return threats.EyePosition(client)
}

// IsLineOfFireClearEntity says whether the bot could actually hit it from
// there.
//
//sp:body IsLineOfFireClearEntity
func IsLineOfFireClearEntity(client int32, from [3]float32, who int32) bool {
	if threats.IsLineOfFireClearEntity == nil {
		missing("IsLineOfFireClearEntity")
	}
	return threats.IsLineOfFireClearEntity(client, from, who)
}

// CountOfBotsWithNamedAction is how many bots are already doing that, which is
// how one job is kept to one bot.
//
//sp:plugin GetCountOfBotsWithNamedAction
func CountOfBotsWithNamedAction(name string) int32 {
	if threats.CountOfBotsWithNamedAction == nil {
		missing("GetCountOfBotsWithNamedAction")
	}
	return threats.CountOfBotsWithNamedAction(name)
}

// SpeakConceptIfAllowed makes the bot say something, if the game lets it.
//
//sp:native BaseMultiplayerPlayer_SpeakConceptIfAllowed
func SpeakConceptIfAllowed(client int32, concept int32) {
	if threats.SpeakConceptIfAllowed == nil {
		missing("BaseMultiplayerPlayer_SpeakConceptIfAllowed")
	}
	threats.SpeakConceptIfAllowed(client, concept)
}

// NearestEnemyTeleporter is util.sp's, ported: internal/body/scan generates it.
//
//sp:body GetNearestEnemyTeleporter
func NearestEnemyTeleporter(client int32, maxDistance float32) int32 {
	if threats.NearestEnemyTeleporter == nil {
		missing("GetNearestEnemyTeleporter")
	}
	return threats.NearestEnemyTeleporter(client, maxDistance)
}
