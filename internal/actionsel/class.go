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
