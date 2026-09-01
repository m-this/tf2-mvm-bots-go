/*
Package mapconfig reads the per-map config out of source/tf2_defenderbots.sp:
the lists of spots, with and without a zone name each, and the file of bot
names.

Config_LoadMap itself stays in the plugin for now: it is glue over the
esMapConfiguration record, whose two scalar fields the generator cannot write
yet.
*/
package mapconfig

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// LoadNestSpots reads a block of spots that may each name a zone, into two
// lists kept in step.
//
//sp:name Config_LoadNestSpots
func LoadNestSpots(kv engine.KeyValues, key string, locations engine.List, zones engine.List) {
	if !kv.JumpToKey(key, false) {
		return
	}

	if kv.GotoFirstSubKeyKeysOnly(false) {
		for {
			vec := kv.Vector("origin")
			locations.PushArray(vec)

			zone := kv.StringOr("zone")
			zones.PushStringText(zone)

			if !kv.GotoNextKeyKeysOnly(false) {
				break
			}
		}

		kv.GoBack()
	}

	kv.GoBack()
}

// LoadLocations reads a block of spots with no zone.
//
//sp:name Config_LoadLocations
func LoadLocations(kv engine.KeyValues, key string, locations engine.List) {
	if !kv.JumpToKey(key, false) {
		return
	}

	if kv.GotoFirstSubKeyKeysOnly(false) {
		for {
			vec := kv.Vector("origin")
			locations.PushArray(vec)

			if !kv.GotoNextKeyKeysOnly(false) {
				break
			}
		}

		kv.GoBack()
	}

	kv.GoBack()
}

// LoadBotNames reads one name per line, blank lines dropped.
//
//sp:name Config_LoadBotNames
func LoadBotNames() {
	filePath := engine.BuildPath("configs/defenderbots/bot_names.txt")
	hConfigFile := engine.OpenFile(filePath, "r")

	if hConfigFile == engine.NoFile() {
		engine.LogError("Config_LoadBotNames: Could not locate file %s!", filePath)
		return
	}

	defer hConfigFile.Close()

	engine.BotNames().Clear()

	for {
		ok, currentLine := engine.ReadFileLine(hConfigFile)
		if !ok {
			break
		}

		engine.TrimString(currentLine)

		if engine.TextLength(currentLine) > 0 {
			engine.BotNames().PushStringText(currentLine)
		}
	}
}
