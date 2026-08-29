package threat

import "github.com/m-this/tf2-mvm-bots-go/internal/tf"

/*
	The two ranges, and they are the plugin's

Both were widened after measuring. At 400 units the order cost more than it
bought: ten runs on Decoy put defender deaths at 54 against the old code's 43,
for the same waves cleared. 400 is a rocket's splash, not a firefight, so a bot
would walk its aim off the Heavy shooting it as soon as anything better appeared
anywhere.

They are float32 because the plugin compares them against a float32 range and
squares them in float32. A float64 constant here would answer differently at the
boundary, which is the kind of difference a port is not allowed to introduce.
*/
const (
	UrgentRange   float32 = 750.0
	PriorityRange float32 = 1500.0
)

// Threat is everything the decision reads about one target. The engine reads
// are done by the caller, so the decision is a pure function of this.
type Threat struct {
	// RangeSquared is the squared distance from the bot to the target, which
	// is what the plugin has: it never takes the square root.
	RangeSquared float32
	/* IsPlayer and InGame are the two the shipped code checks together before
	it reads anything class shaped.

	They have an order the caller must respect. The shipped test is || and
	short circuits, so IsClientInGame is never reached for something that is
	not a player, and calling it to fill this field would throw "Client index
	is invalid" on every tank. Fill InGame as IsPlayer && IsClientInGame. The
	decision cannot enforce that, because by the time it runs the call has
	already been made: see mvm-z83.46. */
	IsPlayer bool
	InGame   bool
	Class    tf.Class
	// Giant is TF2_IsMiniBoss, Carrier is TF2_HasTheFlag.
	Giant   bool
	Carrier bool
}

/*
	PriorityOf is the shipped ThreatPriority, in the shipped order

The order of the tests is the behaviour, not a tidiness question. Urgent is
first, so anything close enough outranks the list whatever it is, and that
includes a tank: a tank inside 750 units answers PriorityUrgent here because it
does in the plugin, even though the tank is invisible to every scan that finds
threats. See mvm-ds3, which this does not fix.

Out of range is second and also runs before the player test, so a distant tank
answers PriorityNone by the range rather than by not being a player.
*/
func PriorityOf(t Threat) Priority {
	switch BandOf(t.RangeSquared) {
	case BandUrgent:
		return PriorityUrgent

	// Too far to be worth walking the aim across the map for.
	case BandTooFar:
		return PriorityNone
	}

	if !t.IsPlayer || !t.InGame {
		return PriorityNone
	}

	switch t.Class {
	// A giant with a Medic on it is not killable until the Medic is dead.
	case tf.ClassMedic:
		return PriorityMedic

	// The two the rest of the team cannot get to: one sits out of reach, the
	// other builds.
	case tf.ClassSniper, tf.ClassEngineer:
		return PrioritySupport
	}

	// Carrying the bomb halves a robot's speed, except a giant's, so that one
	// is still running.
	if t.Giant && t.Carrier {
		return PriorityGiantBomb
	}
	if t.Giant {
		return PriorityGiant
	}
	if t.Carrier {
		return PriorityBomb
	}
	return PriorityNone
}
