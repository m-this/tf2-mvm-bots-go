package actionsel

import "github.com/m-this/tf2-mvm-bots-go/internal/tf"

// Class is the game's TFClassType, which internal/tf owns because
// internal/threat branches on the same enum. Aliased rather than re-declared:
// two copies of one enum is the bug this repository exists to remove.
type Class = tf.Class

// The ten classes, in declared order.
const (
	ClassUnknown  = tf.ClassUnknown
	ClassScout    = tf.ClassScout
	ClassSniper   = tf.ClassSniper
	ClassSoldier  = tf.ClassSoldier
	ClassDemoMan  = tf.ClassDemoMan
	ClassMedic    = tf.ClassMedic
	ClassHeavy    = tf.ClassHeavy
	ClassPyro     = tf.ClassPyro
	ClassSpy      = tf.ClassSpy
	ClassEngineer = tf.ClassEngineer
)

// Classes is every class, in declared order.
func Classes() []Class { return tf.Classes() }
