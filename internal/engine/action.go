package engine

/*
What a callback answers the engine with.

Written on the action the engine handed in, so these emit as action.Continue()
and action.Done("No money"). The action itself is never named in a body: it is
the first parameter of every callback and there is nothing else a body could do
with it.

The reason strings reach the game's own debug output and the test-bed's
telemetry, so two call sites with the same outcome and different reasons are two
different things and both are kept, byte for byte.
*/

// Outcome is what a behaviour callback returns, which SourceMod calls Action.
//
//sp:tag Action
type Outcome int32

// Behaviour is a BehaviorAction, the thing a constructor hands back and a
// callback can change to.
//
//sp:tag BehaviorAction
type Behaviour int32

// ActionCalls are the four answers, kept apart for the same reason BotCalls is.
type ActionCalls struct {
	Continue                     func() Outcome
	Done                         func(reason string) Outcome
	ChangeTo                     func(next Behaviour, reason string) Outcome
	ActionName                   func(a Behaviour) Text
	IterateActions               func(client int32, visit func(action Behaviour))
	AppendText                   func(from string) Text
	AppendTextFrom               func(from Text) Text
	SuspendFor                   func(next Behaviour, reason string) Outcome
	GotoUpgrade                  func() Behaviour
	MoveToFront                  func() Behaviour
	MarkGiant                    func() Behaviour
	CollectNearMoney             func() Behaviour
	StickyTrap                   func() Behaviour
	EngineerIdle                 func() Behaviour
	SpyLurk                      func() Behaviour
	MarkGiantIsPossible          func(client int32) bool
	CollectNearMoneySelectTarget func(client int32) bool
	StickyTrapIsPossible         func(client int32) bool
	ShouldEmptyStack             func(actor int32) bool
	SetCallback                  func(a Behaviour, slot string)
	ResultType                   func(r ActionResult) int32
	ResultAction                 func(r ActionResult) Behaviour
	AttackUber                   func() Behaviour
	AttackUberIsPossible         func(actor int32, medigun int32) bool
	MedicRevive                  func() Behaviour
	MedicReviveIsPossible        func(actor int32) bool
	PointMedicAtBiggestBody      func(action Behaviour, actor int32)
	MedicUberAndResist           func(actor int32, medigun int32, patient int32)
	DesiredBotAction             func(client int32, action Behaviour) Outcome
	Actor                        func() int32
	TryToSustain                 func() Outcome
	TryChangeTo                  func(next Behaviour, priority int32, reason string) Outcome
	TryContinue                  func() Outcome
	TryDone                      func(priority int32, reason string) Outcome
}

var actions ActionCalls

// InstallActions puts a set of answers behind the four and returns the undo.
func InstallActions(c ActionCalls) func() {
	previous := actions
	Fill(&c)
	actions = c
	return func() { actions = previous }
}

// Continue keeps the behaviour running.
//
//sp:native action.Continue
func Continue() Outcome { return actions.Continue() }

// ThisAction is the action the engine handed the callback, for a helper that
// takes it rather than being one.
//
//sp:global action
func ThisAction() Behaviour { return 0 }

// EndWith is Done written on an action a helper was given, which is the same
// call with the receiver spelled out.
//
//sp:method Done
func (a Behaviour) EndWith(reason string) Outcome { return actions.Done(reason) }

// Done ends the behaviour and says why.
//
//sp:native action.Done
func Done(reason string) Outcome { return actions.Done(reason) }

// ChangeTo replaces this behaviour with another.
//
//sp:native action.ChangeTo
func ChangeTo(next Behaviour, reason string) Outcome { return actions.ChangeTo(next, reason) }

// SuspendFor runs another behaviour and comes back to this one.
//
//sp:native action.SuspendFor
func SuspendFor(next Behaviour, reason string) Outcome { return actions.SuspendFor(next, reason) }

// TryToSustain asks to keep going, which the engine may refuse.
//
//sp:native action.TryToSustain
func TryToSustain() Outcome { return actions.TryToSustain() }

// TryChangeTo asks to become another behaviour, at a priority the engine
// weighs against whatever else wants to run.
//
//sp:native action.TryChangeTo
func TryChangeTo(next Behaviour, priority int32, reason string) Outcome {
	return actions.TryChangeTo(next, priority, reason)
}

// ResultCritical is RESULT_CRITICAL, which is a change the engine should not
// weigh against anything.
//
//sp:global RESULT_CRITICAL
func ResultCritical() int32 { return 3 }

// TryContinue asks to keep going, at the engine's discretion.
//
//sp:native action.TryContinue
func TryContinue() Outcome { return actions.TryContinue() }

// TryDone asks to stop, at a priority the engine weighs.
//
//sp:native action.TryDone
func TryDone(priority int32, reason string) Outcome { return actions.TryDone(priority, reason) }

// ResultImportant is RESULT_IMPORTANT.
//
//sp:global RESULT_IMPORTANT
func ResultImportant() int32 { return 2 }

// ActionName is what the behaviour calls itself, filled into a buffer.
//
//sp:method GetName sized
func (a Behaviour) ActionName() (name Text) { return actions.ActionName(a) }

// IterateActions walks a bot's action stack, calling the visitor for each,
// taking it by name.
//
//sp:native ActionsManager.Iterator
//nolint:revive // unused-parameter: the visitor is a name the emitter writes, not something the Go calls
func IterateActions(client int32, visit func(action Behaviour)) {
	actions.IterateActions(client, visit)
}

// AppendText adds to the end of a buffer, which is how the stack is built up.
//
//sp:native StrCat fills
func AppendText(from string) (into Text) { return actions.AppendText(from) }

// AppendTextFrom is StrCat where what is added is another buffer.
//
//sp:native StrCat fills
func AppendTextFrom(from Text) (into Text) { return actions.AppendTextFrom(from) }

/*
The constructors and questions the dispatcher hands off through, every one of
them generated by internal/action.
*/

// GotoUpgrade is CTFBotGotoUpgrade. Ported, gotoupgrade.
//
//sp:body CTFBotGotoUpgrade
func GotoUpgrade() Behaviour { return actions.GotoUpgrade() }

// MoveToFront is CTFBotMoveToFront. Ported, movetofront.
//
//sp:body CTFBotMoveToFront
func MoveToFront() Behaviour { return actions.MoveToFront() }

// MarkGiant is CTFBotMarkGiant. Ported, markgiant.
//
//sp:body CTFBotMarkGiant
func MarkGiant() Behaviour { return actions.MarkGiant() }

// CollectNearMoney is CTFBotCollectNearMoney. Ported, collectnearmoney.
//
//sp:body CTFBotCollectNearMoney
func CollectNearMoney() Behaviour { return actions.CollectNearMoney() }

// StickyTrap is CTFBotStickyTrap. Ported, stickytrap.
//
//sp:body CTFBotStickyTrap
func StickyTrap() Behaviour { return actions.StickyTrap() }

// EngineerIdle is CTFBotMvMEngineerIdle. Ported, engineeridle.
//
//sp:body CTFBotMvMEngineerIdle
func EngineerIdle() Behaviour { return actions.EngineerIdle() }

// SpyLurk is CTFBotSpyLurkMvM. Ported, spylurk.
//
//sp:body CTFBotSpyLurkMvM
func SpyLurk() Behaviour { return actions.SpyLurk() }

// MarkGiantIsPossible says there is a giant worth marking. Ported, markgiant.
//
//sp:body CTFBotMarkGiant_IsPossible
func MarkGiantIsPossible(client int32) bool { return actions.MarkGiantIsPossible(client) }

// CollectNearMoneySelectTarget says there is money within reach. Ported,
// collectnearmoney.
//
//sp:body CTFBotCollectNearMoney_SelectTarget
func CollectNearMoneySelectTarget(client int32) bool {
	return actions.CollectNearMoneySelectTarget(client)
}

// StickyTrapIsPossible says the ground is worth trapping. Ported, stickytrap.
//
//sp:body CTFBotStickyTrap_IsPossible
func StickyTrapIsPossible(client int32) bool { return actions.StickyTrapIsPossible(client) }

// End is Done with no reason, which the shims that only refuse write.
//
//sp:method Done
func (a Behaviour) End() Outcome { return actions.Done("") }

// ShouldEmptyStack is the faults injector asking for a bot with no behaviour.
// Ported, faults.
//
//sp:body DebugFaults_ShouldEmpty
func ShouldEmptyStack(actor int32) bool { return actions.ShouldEmptyStack(actor) }

// ActionResult is the out-parameter the game hands a behaviour callback, which
// the mod never writes: every answer here goes through the return value.
//
//sp:tag ActionResult
type ActionResult int32

/*
The callback slots on a BehaviorAction.

SourceMod's actions extension exposes each one as a settable property, so
overriding a game behaviour is an assignment rather than a hook: the callback is
named, the extension keeps it, and the game calls it. Every one of these takes
its callback by name the way CreateTimer does.

The signatures are written the Go way, with the by-reference answer as a second
result: that is what a body declares, and the emitter is what turns it into the
by-reference parameter SourcePawn passes.
*/

// SetSelectTargetPoint overrides where the bot aims.
//
//sp:propertyset SelectTargetPoint
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetSelectTargetPoint(callback func(action Behaviour, nextbot Bot, entity int32) (Outcome, [3]float32)) {
	actions.SetCallback(a, "SelectTargetPoint")
}

// SetShouldAttack overrides whether a threat is worth shooting.
//
//sp:propertyset ShouldAttack
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetShouldAttack(callback func(action Behaviour, nextbot Bot, knownEntity Known) (Outcome, Answer)) {
	actions.SetCallback(a, "ShouldAttack")
}

// SetUpdate overrides the per-think body.
//
//sp:propertyset Update
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetUpdate(callback func(action Behaviour, actor int32, interval float32, result ActionResult) Outcome) {
	actions.SetCallback(a, "Update")
}

// SetUpdatePost runs after the game's own think rather than instead of it,
// which is what the medic nudge needs: the heal action keeps its walking.
//
//sp:propertyset UpdatePost
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetUpdatePost(callback func(action Behaviour, actor int32, interval float32, result ActionResult) Outcome) {
	actions.SetCallback(a, "UpdatePost")
}

// SetOnStart overrides the moment the behaviour begins.
//
//sp:propertyset OnStart
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetOnStart(callback func(action Behaviour, actor int32, priorAction Behaviour, result ActionResult) Outcome) {
	actions.SetCallback(a, "OnStart")
}

// SetSelectMoreDangerousThreat overrides which of two robots the bot fears.
//
//sp:propertyset SelectMoreDangerousThreat
//nolint:revive // unused-parameter: the callback is a name the emitter writes
func (a Behaviour) SetSelectMoreDangerousThreat(callback func(action Behaviour, nextbot Bot, entity int32, threat1 Known, threat2 Known) (Outcome, Known)) {
	actions.SetCallback(a, "SelectMoreDangerousThreat")
}

// ResultType is the kind of answer a callback's result carries.
//
//sp:property type
func (r ActionResult) ResultType() int32 { return actions.ResultType(r) }

// ResultAction is the behaviour the result names, when it names one.
//
//sp:property action
func (r ActionResult) ResultAction() Behaviour { return actions.ResultAction(r) }

// ChangeToResult is CHANGE_TO, the result that swaps the behaviour out next
// frame.
//
//sp:global CHANGE_TO
func ChangeToResult() int32 { return 1 }

// AttackUber is CTFBotAttackUber. Ported, attackforuber.
//
//sp:body CTFBotAttackUber
func AttackUber() Behaviour { return actions.AttackUber() }

// AttackUberIsPossible says there is an uber worth breaking. Ported,
// attackforuber.
//
//sp:body CTFBotAttackUber_IsPossible
func AttackUberIsPossible(actor int32, medigun int32) bool {
	return actions.AttackUberIsPossible(actor, medigun)
}

// MedicRevive is CTFBotMedicRevive. Ported, medicrevive.
//
//sp:body CTFBotMedicRevive
func MedicRevive() Behaviour { return actions.MedicRevive() }

// MedicReviveIsPossible says somebody is down and worth raising. Ported,
// medicrevive.
//
//sp:body CTFBotMedicRevive_IsPossible
func MedicReviveIsPossible(actor int32) bool { return actions.MedicReviveIsPossible(actor) }

// FeatureMedicPocketsBiggest is FEATURE_MEDIC_POCKETS_BIGGEST.
//
//sp:global FEATURE_MEDIC_POCKETS_BIGGEST
func FeatureMedicPocketsBiggest() int32 { return 10 }

// PointMedicAtBiggestBodyNow writes the patient handle. Ported, medicnudge.
//
//sp:body PointMedicAtBiggestBody
func PointMedicAtBiggestBodyNow(action Behaviour, actor int32) {
	actions.PointMedicAtBiggestBody(action, actor)
}

// MedicUberAndResistNow deploys and cycles. Ported, medicnudge.
//
//sp:body MedicUberAndResist
func MedicUberAndResistNow(actor int32, medigun int32, patient int32) {
	actions.MedicUberAndResist(actor, medigun, patient)
}

// DesiredBotAction is the dispatcher. Ported, dispatch.
//
//sp:body GetDesiredBotAction
func DesiredBotAction(client int32, action Behaviour) Outcome {
	return actions.DesiredBotAction(client, action)
}
