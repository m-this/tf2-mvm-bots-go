/*
Package dumpspot is the sm_dump_spot command out of source/tf2_defenderbots.sp.

Somebody authoring a map config stands where they want a spot and reads the
coordinates back, either from their feet or from their crosshair.
*/
package dumpspot

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
AimRange is how far the crosshair may reach before the answer is taken as no
answer.

Declared here rather than in the plugin: the generated file is included long
before the line the define used to sit on, and a define has to precede its
reader.
*/
//
//sp:name DUMP_SPOT_AIM_RANGE
const AimRange = 8192.0

// TraceFilterIgnorePlayers lets the ray through everybody, so the answer is a
// point on the map and not on whoever was standing in the way.
//
//sp:name TraceFilter_IgnorePlayers
//nolint:revive // unused-parameter: a trace filter is handed the mask and the caller's cell
func TraceFilterIgnorePlayers(entity int32, mask int32, data engine.Cell) bool {
	return entity > engine.MaxClients()
}

// TraceAimToWorld is where the crosshair lands, and whether it lands anywhere
// near enough to be meant.
//
//sp:name TraceAimToWorld
func TraceAimToWorld(client int32) (hit bool, result [3]float32) {
	eyes := engine.ClientEyePosition(client)
	angles := engine.ClientEyeAngles(client)

	trace := engine.TraceRayFilterEx(eyes, angles, engine.MaskSolid(), engine.RayTypeInfinite(), TraceFilterIgnorePlayers, client)

	defer trace.Close()

	hit = engine.DidHitTrace(trace)

	if hit {
		result = engine.TraceEndPositionOf(trace)
	}

	if !hit || engine.VectorDistance(eyes, result) > AimRange {
		return false, result
	}

	return true, result
}

/*
	CommandDumpSpot prints coordinates for a map config block

Either the caller's feet or, with "aim" as the second argument, wherever their
crosshair lands. The line is printed and logged in the exact shape the config
file wants, so it can be pasted straight in.
*/
//
//sp:name Command_DumpSpot
//sp:public
func CommandDumpSpot(client int32, args int32) engine.Outcome {
	if client < 1 || !engine.IsClientInGame(client) {
		engine.ReplyToCommand(client, "[SM] This command requires standing somewhere in the map.")
		return engine.PluginHandled()
	}

	block := engine.LiteralText("EngineerNest")

	if args >= 1 {
		_, block = engine.CmdArg(1)
	}

	var mode engine.Text

	if args >= 2 {
		_, mode = engine.CmdArg(2)
	}

	var origin [3]float32

	if engine.StrEqualCased(mode, "aim", false) {
		hit, aimed := TraceAimToWorld(client)
		if !hit {
			engine.ReplyToCommand(client, "[SM] Your crosshair is not on anything within %.0f units.", AimRange)
			return engine.PluginHandled()
		}
		origin = aimed
	} else {
		origin = engine.Origin(client)
	}

	mapName := engine.CurrentMap()

	engine.ReplyToCommand(client, "[SM] %s on %s:", block, mapName)
	engine.ReplyToCommand(client, "\t\t\t\"origin\" \"%.0f %.0f %.0f\"", origin[0], origin[1], origin[2])

	engine.LogMessage("%s %s: \"origin\" \"%.0f %.0f %.0f\"", mapName, block, origin[0], origin[1], origin[2])

	return engine.PluginHandled()
}
