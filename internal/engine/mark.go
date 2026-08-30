package engine

/*
Marking a giant for death, and the vision interface behind it.

The Fan O'War puts a mark on whoever it hits, which is worth a lot on a giant.
Getting the bot to hit the right one means taking the game's own idea of what it
has noticed away for a moment, which is what the vision calls here are for.
*/

// MarkCalls are the answers.
type MarkCalls struct {
	KnownCount             func(v Vision, team Team) int32
	GetKnown               func(v Vision, entity int32) Known
	ForgetAllKnownEntities func(v Vision)
	AddKnownEntity         func(v Vision, entity int32)
	RandomInt              func(low int32, high int32) int32
	EquipWeaponSlot        func(client int32, slot int32) bool
	IsClientConnected      func(client int32) bool
}

var marks MarkCalls

// InstallMarks puts a set of answers behind them.
func InstallMarks(c MarkCalls) func() {
	previous := marks
	marks = c
	return func() { marks = previous }
}

// InvalidEntReference is INVALID_ENT_REFERENCE.
//
//sp:global INVALID_ENT_REFERENCE
func InvalidEntReference() int32 { return -1 }

// TeamBlue is TFTeam_Blue, which in this mode is the robots.
//
//sp:global TFTeam_Blue
func TeamBlue() Team { return 3 }

// ConditionMarkedForDeath is TFCond_MarkedForDeath.
//
//sp:global TFCond_MarkedForDeath
func ConditionMarkedForDeath() Condition { return 29 }

// KnownCount is how many things of that team the bot has noticed.
//
//sp:method GetKnownCount
func (v Vision) KnownCount(team Team) int32 {
	if marks.KnownCount == nil {
		missing("IVision.GetKnownCount")
	}
	return marks.KnownCount(v, team)
}

// GetKnown is what the bot remembers about that entity, and nothing when it has
// not noticed it.
//
//sp:method GetKnown
func (v Vision) GetKnown(entity int32) Known {
	if marks.GetKnown == nil {
		missing("IVision.GetKnown")
	}
	return marks.GetKnown(v, entity)
}

// ForgetAllKnownEntities empties what the bot has noticed.
//
//sp:method ForgetAllKnownEntities
func (v Vision) ForgetAllKnownEntities() {
	if marks.ForgetAllKnownEntities == nil {
		missing("IVision.ForgetAllKnownEntities")
	}
	marks.ForgetAllKnownEntities(v)
}

// AddKnownEntity puts one back, which is how the bot is made to look at it.
//
//sp:method AddKnownEntity
func (v Vision) AddKnownEntity(entity int32) {
	if marks.AddKnownEntity == nil {
		missing("IVision.AddKnownEntity")
	}
	marks.AddKnownEntity(v, entity)
}

// RandomInt is the game's own randomness, which mvm-z83.18 has yet to make a
// parameter.
//
//sp:native GetRandomInt
func RandomInt(low int32, high int32) int32 {
	if marks.RandomInt == nil {
		missing("GetRandomInt")
	}
	return marks.RandomInt(low, high)
}

// EquipWeaponSlot puts whatever is in that slot in the bot's hands.
//
//sp:plugin EquipWeaponSlot
func EquipWeaponSlot(client int32, slot int32) bool {
	if marks.EquipWeaponSlot == nil {
		missing("EquipWeaponSlot")
	}
	return marks.EquipWeaponSlot(client, slot)
}

// IsClientConnected says whether the slot holds anybody at all, which is a
// weaker question than being in the game.
//
//sp:native IsClientConnected
func IsClientConnected(client int32) bool {
	if marks.IsClientConnected == nil {
		missing("IsClientConnected")
	}
	return marks.IsClientConnected(client)
}
