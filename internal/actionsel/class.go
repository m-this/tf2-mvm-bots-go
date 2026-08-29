package actionsel

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

// numClasses is one past the last class, so Classes covers exactly the enum.
const numClasses = ClassEngineer + 1

// Classes is every class, in declared order.
func Classes() []Class {
	all := make([]Class, 0, numClasses)
	for c := ClassUnknown; c < numClasses; c++ {
		all = append(all, c)
	}
	return all
}
