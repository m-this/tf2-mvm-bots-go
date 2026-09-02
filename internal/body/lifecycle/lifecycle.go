/*
Package lifecycle is the plugin's own forwards out of source/tf2_defenderbots.sp:
loading, unloading, a frame, an entity appearing, and the two conditions the mod
watches for.
*/
package lifecycle

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// OnLibraryAdded notices the campaign plugin arriving.
//
//sp:name OnLibraryAdded
//sp:public
func OnLibraryAdded(name engine.Text) {
	if engine.StrEqual(name, "tf2_archipelago") {
		engine.ArchipelagoRecheck()
	}
}

// OnLibraryRemoved notices it leaving.
//
//sp:name OnLibraryRemoved
//sp:public
func OnLibraryRemoved(name engine.Text) {
	if engine.StrEqual(name, "tf2_archipelago") {
		engine.ArchipelagoRecheck()
	}
}

// OnAllPluginsLoaded is the third place the campaign plugin can turn up: it may
// have loaded before this one did.
//
//sp:name OnAllPluginsLoaded
//sp:public
func OnAllPluginsLoaded() {
	engine.ArchipelagoRecheck()
}

// OnPluginEnd takes the bots with it.
//
//sp:name OnPluginEnd
//sp:public
func OnPluginEnd() {
	engine.RemoveAllDefenderBotsFor("BM3 OnPluginEnd")
}

// OnConfigsExecuted publishes what this build can do, once the convars are
// their configured values.
//
//sp:name OnConfigsExecuted
//sp:public
func OnConfigsExecuted() {
	engine.PublishActiveFeatures()
}

// OnGameFrame is the fault watcher, which is the only thing here that runs
// every frame.
//
//sp:name OnGameFrame
//sp:public
func OnGameFrame() {
	engine.DebugFaultsOnGameFrame()
}

// OnEntityCreated catches the population manager and hands everything else to
// the DHook wiring.
//
//sp:name OnEntityCreated
//sp:public
func OnEntityCreated(entity int32, classname engine.Text) {
	if engine.StrEqual(classname, "info_populator") {
		engine.SetPopulationManager(entity)
	}

	engine.DHooksOnEntityCreated(entity, classname)
}

// TF2OnConditionAdded watches for a sentry buster winding up, because what it
// is about to do decides where every bot near it should be.
//
//sp:name TF2_OnConditionAdded
//sp:public
func TF2OnConditionAdded(client int32, condition engine.Condition) {
	if condition == engine.ConditionTaunting() && engine.ClientTeam(client) == engine.TeamBlue() && engine.IsSentryBusterRobot(client) {
		// Keep track of the player that is detonating.
		engine.SetDetonatingPlayer(client)
		engine.CreateTimerData(2.0, TimerForgetDetonatingPlayer, client)
	}
}

/*
	TimerForgetDetonatingPlayer clears the buster once it should have gone off

Another player might have started detonating since, so the newest one is not
forgotten on this one's timer.
*/
//
//sp:name Timer_ForgetDetonatingPlayer
//sp:public
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerForgetDetonatingPlayer(timer engine.Timer, data engine.Cell) engine.Outcome {
	if engine.DetonatingPlayer() == int32(data) {
		engine.SetDetonatingPlayer(-1)
	}

	return engine.PluginStop()
}

// DefenderBotTouchPost calls out an enemy spy on contact, which is the one
// thing a bot can be certain about a disguise.
//
//sp:name DefenderBot_TouchPost
//sp:public
func DefenderBotTouchPost(entity int32, other int32) {
	if engine.IsPlayer(other) && engine.GetClientTeam(other) != engine.GetClientTeam(entity) && engine.IsPlayerInCondition(other, engine.ConditionDisguised()) {
		engine.NoticeThreat(entity, other)
	}
}
