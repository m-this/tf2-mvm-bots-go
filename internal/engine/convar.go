package engine

/*
The convars the plugin creates at load, and reading one.

A convar is a handle the plugin makes in OnPluginStart and keeps in a global, so
a body names the global and reads a property off it. SourcePawn writes that
without parentheses, convar.BoolValue, which Go has no form for but a method;
//sp:property is the difference.

The feature switches go through Feature() rather than through here, because that
is what the plugin writes and the port is behaviour identical. When features.sp
owns the convars on this side too, the two can become one path.
*/

// ConVarCalls are the answers.
type ConVarCalls struct {
	BoolValue  func(c ConVar) bool
	IntValue   func(c ConVar) int32
	FloatValue func(c ConVar) float32
}

var convars ConVarCalls

// InstallConVars puts a set of answers behind them.
func InstallConVars(c ConVarCalls) func() {
	previous := convars
	convars = c
	return func() { convars = previous }
}

// ConVar is SourceMod's ConVar.
//
//sp:tag ConVar
type ConVar int32

// ManagerDebug is redbots_manager_debug, the older and wider of the two debug
// switches.
//
//sp:global redbots_manager_debug
func ManagerDebug() ConVar { return 0 }

// DebugActions is redbots_manager_debug_actions, which turns on the running
// commentary about which behaviour a bot picked.
//
//sp:global redbots_manager_debug_actions
func DebugActions() ConVar {
	return 0
}

// Bool is the convar as a switch.
//
//sp:property BoolValue
func (c ConVar) Bool() bool {
	if convars.BoolValue == nil {
		missing("ConVar.BoolValue")
	}
	return convars.BoolValue(c)
}

// Int is the convar as a number.
//
//sp:property IntValue
func (c ConVar) Int() int32 {
	if convars.IntValue == nil {
		missing("ConVar.IntValue")
	}
	return convars.IntValue(c)
}

// Float is the convar as a distance or a time.
//
//sp:property FloatValue
func (c ConVar) Float() float32 {
	if convars.FloatValue == nil {
		missing("ConVar.FloatValue")
	}
	return convars.FloatValue(c)
}
