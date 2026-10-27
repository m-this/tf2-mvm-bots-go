package engine

/*
The nextbot interfaces, which are methodmaps.

SourceMod's API for these is not functions: CBaseNPC_GetNextBotOfEntity hands
back an INextBot and everything after that is written on a receiver. There is no
plain function behind GetVisionInterface to call instead, so refusing method
calls refused the engine, and the behaviour files are almost entirely this.

Each type carries the SourcePawn tag it stands for and each method the name
SourcePawn calls it by, the same way a native does. internal/spbody resolves a
call by asking go/types what the receiver is, so the Go reads as Go and the
emitted SourcePawn reads as the plugin already writes it.
*/

// Bot is CBaseNPC's INextBot, the entity's nextbot side.
//
//sp:tag INextBot
type Bot int32

// Vision is INextBot's vision interface: what the bot has noticed.
//
//sp:tag IVision
type Vision int32

// Locomotion is INextBot's locomotion interface: how the bot moves.
//
//sp:tag ILocomotion
type Locomotion int32

// Body is INextBot's body interface: posture and where it is looking.
//
//sp:tag IBody
type Body int32

// Known is a CKnownEntity, something the bot has seen and remembers.
//
//sp:tag CKnownEntity
type Known int32

// BotCalls are the nextbot side of the engine, kept apart from Calls because a
// body that touches none of it installs none of it.
type BotCalls struct {
	NextBotOf          func(entity int32) Bot
	Vision             func(bot Bot) Vision
	Locomotion         func(bot Bot) Locomotion
	Body               func(bot Bot) Body
	Entity             func(bot Bot) int32
	PrimaryKnownThreat func(vision Vision, onlyVisible bool) Known
}

var bots BotCalls

// InstallBots puts a set of answers behind the nextbot calls and returns the
// undo, the same way Install does for the rest.
func InstallBots(c BotCalls) func() {
	previous := bots
	Fill(&c)
	bots = c
	return func() { bots = previous }
}

// NextBotOf is the nextbot of an entity, and Address_Null when it has none.
//
//sp:native CBaseNPC_GetNextBotOfEntity
func NextBotOf(entity int32) Bot { return bots.NextBotOf(entity) }

// Vision is what the bot has noticed.
//
//sp:method GetVisionInterface
func (b Bot) Vision() Vision { return bots.Vision(b) }

// Locomotion is how the bot moves.
//
//sp:method GetLocomotionInterface
func (b Bot) Locomotion() Locomotion { return bots.Locomotion(b) }

// Body is the bot's posture and aim.
//
//sp:method GetBodyInterface
func (b Bot) Body() Body { return bots.Body(b) }

// Entity is the client the bot belongs to.
//
//sp:method GetEntity
func (b Bot) Entity() int32 { return bots.Entity(b) }

// PrimaryKnownThreat is what the bot is most worried about, and 0 for nothing.
//
//sp:method GetPrimaryKnownThreat
func (v Vision) PrimaryKnownThreat(onlyVisible bool) Known {
	return bots.PrimaryKnownThreat(v, onlyVisible)
}
