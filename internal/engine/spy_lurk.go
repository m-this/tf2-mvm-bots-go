package engine

/*
The spy's lurk, and the arithmetic behind a backstab.

Circling an enemy is vectors: where he is looking, where the spy is relative to
him, and which way to sidestep. All of it is SourceMod's own vector maths and
the plugin's button pressing.
*/

// LurkCalls are the answers.
type LurkCalls struct {
	ChasePath                 func(actor int32) Path
	AttackTarget              func(actor int32) int32
	SetAttackTarget           func(actor int32, target int32)
	ExtraButtons              func(actor int32) Buttons
	PressButtons              func(b Buttons, buttons int32, duration float32)
	EyeVectors                func(client int32) [3]float32
	NormalizeVector           func(v [3]float32) (float32, [3]float32)
	VectorDotProduct          func(a [3]float32, b [3]float32) float32
	VectorCrossProduct        func(a [3]float32, b [3]float32) [3]float32
	AimHeadTowards            func(body Body, lookAt [3]float32, priority int32, duration float32, replyWhenAimed int32, reason string)
	ModelScale                func(entity int32) float32
	HasBackstabPotential      func(client int32) bool
	ChasePathUpdate           func(p Path, bot Bot, target int32)
	BackstabSkill             func() ConVar
	BestTargetForSpy          func(client int32, maxDistance float32) int32
	SpySap                    func() Behaviour
	SpySapPlayers             func() Behaviour
	SpySapSelectTarget        func(actor int32) bool
	SpySapPlayersSelectTarget func(actor int32) bool
}

var lurks LurkCalls

// InstallLurks puts a set of answers behind them.
func InstallLurks(c LurkCalls) func() {
	previous := lurks
	lurks = c
	return func() { lurks = previous }
}

// Buttons is the plugin's per-client button presser.
//
//sp:tag ExtraButtons
type Buttons int32

// The three button and condition constants the lurk names.

// InMoveRight is IN_MOVERIGHT.
//
//sp:global IN_MOVERIGHT
func InMoveRight() int32 { return 1024 }

// InMoveLeft is IN_MOVELEFT.
//
//sp:global IN_MOVELEFT
func InMoveLeft() int32 { return 512 }

// ConditionDisguised is TFCond_Disguised.
//
//sp:global TFCond_Disguised
func ConditionDisguised() Condition { return 4 }

// AimMandatory is MANDATORY, the aim priority that wins.
//
//sp:global MANDATORY
func AimMandatory() int32 { return 3 }

// ChasePathOf is the path a bot uses when it is following somebody rather than
// walking to a place.
//
//sp:slot m_pChasePath
func ChasePathOf(actor int32) Path {
	if lurks.ChasePath == nil {
		missing("m_pChasePath")
	}
	return lurks.ChasePath(actor)
}

// AttackTargetOf is who the bot is going for, which IsHindrance reads.
//
//sp:slot m_iAttackTarget
func AttackTargetOf(actor int32) int32 {
	if lurks.AttackTarget == nil {
		missing("m_iAttackTarget")
	}
	return lurks.AttackTarget(actor)
}

// SetAttackTarget records who the bot is going for.
//
//sp:slotset m_iAttackTarget
func SetAttackTarget(actor int32, target int32) {
	if lurks.SetAttackTarget == nil {
		missing("m_iAttackTarget")
	}
	lurks.SetAttackTarget(actor, target)
}

// ExtraButtonsOf is the bot's button presser.
//
//sp:slot g_arrExtraButtons
func ExtraButtonsOf(actor int32) Buttons {
	if lurks.ExtraButtons == nil {
		missing("g_arrExtraButtons")
	}
	return lurks.ExtraButtons(actor)
}

// PressButtons holds those buttons down for that long.
//
//sp:method PressButtons
func (b Buttons) PressButtons(buttons int32, duration float32) {
	if lurks.PressButtons == nil {
		missing("ExtraButtons.PressButtons")
	}
	lurks.PressButtons(b, buttons, duration)
}

// UpdateChase walks the bot one step along a chase path.
//
//sp:method Update
func (p Path) UpdateChase(bot Bot, target int32) {
	if lurks.ChasePathUpdate == nil {
		missing("ChasePath.Update")
	}
	lurks.ChasePathUpdate(p, bot, target)
}

// EyeVectors is which way the player is looking.
//
//sp:native BasePlayer_EyeVectors
func EyeVectors(client int32) (forward [3]float32) {
	if lurks.EyeVectors == nil {
		missing("BasePlayer_EyeVectors")
	}
	return lurks.EyeVectors(client)
}

// NormalizeVector makes it unit length and answers how long it was.
//
//sp:native NormalizeVector
func NormalizeVector(v [3]float32) (length float32, unit [3]float32) {
	if lurks.NormalizeVector == nil {
		missing("NormalizeVector")
	}
	return lurks.NormalizeVector(v)
}

// VectorDotProduct is how much the two point the same way.
//
//sp:native GetVectorDotProduct
func VectorDotProduct(a [3]float32, b [3]float32) float32 {
	if lurks.VectorDotProduct == nil {
		missing("GetVectorDotProduct")
	}
	return lurks.VectorDotProduct(a, b)
}

// VectorCrossProduct is the vector at right angles to both, which is what says
// whether to step left or right.
//
//sp:native GetVectorCrossProduct
func VectorCrossProduct(a [3]float32, b [3]float32) (cross [3]float32) {
	if lurks.VectorCrossProduct == nil {
		missing("GetVectorCrossProduct")
	}
	return lurks.VectorCrossProduct(a, b)
}

// AimHeadTowards points the bot's head, over the game's own aiming.
//
//sp:plugin AimHeadTowards
func AimHeadTowards(body Body, lookAt [3]float32, priority int32, duration float32, replyWhenAimed int32, reason string) {
	if lurks.AimHeadTowards == nil {
		missing("AimHeadTowards")
	}
	lurks.AimHeadTowards(body, lookAt, priority, duration, replyWhenAimed, reason)
}

// ModelScale is how big the model is, which is what makes a giant's stab range
// longer than a normal robot's.
//
//sp:native BaseAnimating_GetModelScale
func ModelScale(entity int32) float32 {
	if lurks.ModelScale == nil {
		missing("BaseAnimating_GetModelScale")
	}
	return lurks.ModelScale(entity)
}

// HasBackstabPotential says the spy is behind him enough for the game to give
// it.
//
//sp:plugin HasBackstabPotential
func HasBackstabPotential(client int32) bool {
	if lurks.HasBackstabPotential == nil {
		missing("HasBackstabPotential")
	}
	return lurks.HasBackstabPotential(client)
}

// BackstabSkill is redbots_manager_bot_backstab_skill: whether the spy attacks
// when it knows it can stab or when it thinks it can.
//
//sp:global redbots_manager_bot_backstab_skill
func BackstabSkill() ConVar { return 0 }

// BestTargetForSpy is util.sp:1235, ported.
//
//sp:body GetBestTargetForSpy
func BestTargetForSpy(client int32, maxDistance float32) int32 {
	if lurks.BestTargetForSpy == nil {
		missing("GetBestTargetForSpy")
	}
	return lurks.BestTargetForSpy(client, maxDistance)
}

// The two behaviours the lurk suspends for, both ported.

// SpySap is CTFBotSpySap.
//
//sp:body CTFBotSpySap
func SpySap() Behaviour {
	if lurks.SpySap == nil {
		missing("CTFBotSpySap")
	}
	return lurks.SpySap()
}

// SpySapPlayers is CTFBotSpySapPlayers.
//
//sp:body CTFBotSpySapPlayers
func SpySapPlayers() Behaviour {
	if lurks.SpySapPlayers == nil {
		missing("CTFBotSpySapPlayers")
	}
	return lurks.SpySapPlayers()
}

// SpySapSelectTarget is that behaviour's own precondition.
//
//sp:body CTFBotSpySap_SelectTarget
func SpySapSelectTarget(actor int32) bool {
	if lurks.SpySapSelectTarget == nil {
		missing("CTFBotSpySap_SelectTarget")
	}
	return lurks.SpySapSelectTarget(actor)
}

// SpySapPlayersSelectTarget is the other one's.
//
//sp:body CTFBotSpySapPlayers_SelectTarget
func SpySapPlayersSelectTarget(actor int32) bool {
	if lurks.SpySapPlayersSelectTarget == nil {
		missing("CTFBotSpySapPlayers_SelectTarget")
	}
	return lurks.SpySapPlayersSelectTarget(actor)
}
