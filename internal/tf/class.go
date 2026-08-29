/*
Package tf is the game's own vocabulary, shared by the decisions that read it.

Nothing here is a decision. It is the enums the plugin already has, in the order
the plugin declares them, so two ported decisions that both branch on a class
branch on the same class rather than on two copies of one enum that can drift.
*/
package tf

// Class is TFClassType from tf2.inc, in its declared order.
type Class int32

// The ten classes, in declared order.
const (
	ClassUnknown Class = iota
	ClassScout
	ClassSniper
	ClassSoldier
	ClassDemoMan
	ClassMedic
	ClassHeavy
	ClassPyro
	ClassSpy
	ClassEngineer
)

// NumClasses is one past the last class, so Classes covers exactly the enum.
const NumClasses = ClassEngineer + 1

// Classes is every class, in declared order.
func Classes() []Class {
	all := make([]Class, 0, NumClasses)
	for c := ClassUnknown; c < NumClasses; c++ {
		all = append(all, c)
	}
	return all
}

// className is what a class is called in a failure message, and it is the
// SourcePawn name with the TFClass_ prefix taken off.
var className = map[Class]string{
	ClassUnknown: "Unknown", ClassScout: "Scout", ClassSniper: "Sniper",
	ClassSoldier: "Soldier", ClassDemoMan: "DemoMan", ClassMedic: "Medic",
	ClassHeavy: "Heavy", ClassPyro: "Pyro", ClassSpy: "Spy",
	ClassEngineer: "Engineer",
}

func (c Class) String() string {
	if s, ok := className[c]; ok {
		return s
	}
	return "Class(" + string(rune('0'+int(c)%10)) + ")"
}
