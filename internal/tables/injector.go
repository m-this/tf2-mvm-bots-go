package tables

/*
	The faults the test-bed makes happen on purpose

Three engineer fixes shipped against faults the test-bed would not produce, so
each arm ran the same code and the run said nothing: mvm-0lo. mvm-zx0 is open
only because its injector was never armed, and mvm-81n's finding is that a
results file carries no record of which one was.

One row per injector, here, so the plugin's convar and the test-bed's knowledge
of it are one declaration. A run records which of these its arm turned on, so a
file of numbers says what was being done to the bots when they were produced.
*/

// InjectorKind is what the value means, which is what a reader needs to know
// before setting one.
type InjectorKind string

const (
	// InjectorSeconds holds the fault for that long after a wave starts.
	InjectorSeconds InjectorKind = "seconds"
	// InjectorCount is how many times to do it, per bot.
	InjectorCount InjectorKind = "count"
	// InjectorSwitch is on or off.
	InjectorSwitch InjectorKind = "switch"
	// InjectorName is a class name rather than a number, so it arms nothing
	// on its own: it says which bot the others take.
	InjectorName InjectorKind = "name"
)

// Injector is one fault, and the bug it exists to reproduce.
type Injector struct {
	// Name is the convar's tail, which is also how a report names it.
	Name string
	Kind InjectorKind
	// About is what it does to the bots.
	About string
	// Why is the bead it was written for. An injector with no bug behind it
	// is a switch somebody invented.
	Why string
}

// ConVar is the console variable the test-bed sets to arm it.
func (i Injector) ConVar() string { return "sm_redbots_debug_" + i.Name }

// Arms says whether setting this to value turns the fault on. A name selects
// rather than arms, and every other kind is off at zero.
func (i Injector) Arms(value string) bool {
	if i.Kind == InjectorName {
		return false
	}
	switch value {
	case "", "0", "0.0", "false":
		return false
	}
	return true
}

// Injectors is every fault the mod can be told to produce, in the order
// debug_faults.sp creates them.
var Injectors = []Injector{
	{
		Name: "wedge_seconds", Kind: InjectorSeconds,
		About: "Hold one defender in place after a wave starts, so the stuck watchdog has something to catch.",
		Why:   "mvm-hnb: the recovery could never move a bot standing on valid nav, and the fix was measured against a wedge that had to be made.",
	},
	{
		Name: "wedge_class", Kind: InjectorName,
		About: "Which class the hold takes.",
		Why:   "mvm-wb0 and mvm-ipf are both the engineer, so the hold has to be able to name him.",
	},
	{
		Name: "refuse_ammo_paths", Kind: InjectorCount,
		About: "Refuse path answers to a metal pack, per bot, so the ammo failover runs.",
		Why:   "mvm-a8g: the failover was measured against a condition that did not happen.",
	},
	{
		Name: "unreachable_goal", Kind: InjectorSwitch,
		About: "Send the held bot at a point off the mesh, so every path search walks the whole thing and finds nothing.",
		Why:   "mvm-cf3: an unreachable goal is what made NavAreaBuildPath cost a frame, and pinning a bot alone reproduces none of it.",
	},
	{
		Name: "old_wedge_recovery", Kind: InjectorSwitch,
		About: "Use the pre-2.21.3 recovery, which only ever tried the area the bot stands in.",
		Why:   "mvm-hnb again, from the other side: measuring what the fix is worth needs the old behaviour on demand.",
	},
	{
		Name: "empty_stack", Kind: InjectorSeconds,
		About: "Leave one defender with no behaviour at all after a wave starts, so the idle watchdog runs.",
		Why:   "mvm-bj8: the stalled sniper's shape, validated at 72 of 72 against internal/wave's idle detector.",
	},
	{
		Name: "trace_snipers", Kind: InjectorSwitch,
		About: "Write every sniper's action stack and position each tenth of a second.",
		Why:   "mvm-bj8: three fixes written from the core alone all failed, because the top frame was not the action that was running.",
	},
}

// InjectorByConVar is the injector a console variable arms, and whether the
// table knows it at all.
func InjectorByConVar(convar string) (Injector, bool) {
	for _, i := range Injectors {
		if i.ConVar() == convar {
			return i, true
		}
	}
	return Injector{}, false
}
