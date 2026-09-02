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

/*
	LoadMap reads the file for the map being played

Every list on the record is emptied first, so a map whose file is missing gets
empty lists rather than the last map's spots. The one log line at the end is
deliberate: a typo in a block name is otherwise silent, because the block is
skipped, the list stays empty, and the bots fall back to the nav mesh as though
nobody had written anything.

//sp:name Config_LoadMap
*/
func LoadMap() {
	engine.ResetMapConfig()

	mapName := engine.CurrentMap()
	filePath := engine.BuildPath("configs/defenderbots/map/%s.cfg", mapName)

	kv := engine.NewKeyValues("MapConfig")

	if !kv.ImportFromFile(filePath) {
		engine.CloseHandle(kv)
		engine.LogError("Config_LoadMap: File not found (%s)", filePath)
		return
	}

	LoadLocations(kv, "SniperSpot", engine.SniperSpots())
	LoadNestSpots(kv, "EngineerNest", engine.EngineerNestSpots(), engine.EngineerNestZones())
	LoadLocations(kv, "TeleporterEntrance", engine.TeleporterEntranceSpots())
	LoadLocations(kv, "TeleporterExit", engine.TeleporterExitSpots())
	LoadNestSpots(kv, "DispenserSpot", engine.DispenserSpots(), engine.DispenserZones())
	LoadLocations(kv, "NestTankOnly", engine.NestTankOnlySpots())
	LoadLocations(kv, "NestNoTank", engine.NestNoTankSpots())
	kv.StringInto("Composition", engine.MapComposition(), engine.MapCompositionSize(), "")
	engine.SetMovingNests(kv.Num("MovingNests", 0) != 0)

	engine.CloseHandle(kv)

	engine.LogMessage("Config_LoadMap: %s: %d sniper, %d nest, %d nest-tank, %d nest-notank, %d dispenser, %d tele-in, %d tele-out, moving nests %d",
		mapName,
		engine.SniperSpots().Length(),
		engine.EngineerNestSpots().Length(),
		engine.NestTankOnlySpots().Length(),
		engine.NestNoTankSpots().Length(),
		engine.DispenserSpots().Length(),
		engine.TeleporterEntranceSpots().Length(),
		engine.TeleporterExitSpots().Length(),
		engine.MovingNests())
}

/*
	MissionDifficultyFromName looks the mission up in the difficulty files

One file per difficulty, one mission name per line. A file that is not there is
skipped rather than fatal: a server that ships none of them still runs, and
falls through to the name search in GetMissionDifficulty.

//sp:name Config_GetMissionDifficultyFromName
//sp:writable missionName
*/
func MissionDifficultyFromName(missionName engine.Text) engine.MissionDifficulty {
	for i := engine.MissionNormal(); i < engine.MissionMaxCount(); i++ {
		filePath := engine.BuildPathText(engine.MissionDifficultyFilePath(i))

		hOpenedFile := engine.OpenFile(filePath, "r")

		if hOpenedFile == engine.NoFile() {
			if engine.ManagerDebug().Bool() {
				engine.LogMessage("Config_GetMissionDifficultyFromName: Could not locate file %s. Skipping...", filePath)
			}

			continue
		}

		for {
			ok, currentLine := engine.ReadFileLine(hOpenedFile)
			if !ok {
				break
			}

			engine.TrimString(currentLine)

			if engine.StrEqualText(currentLine, missionName) {
				hOpenedFile.Close()
				return i
			}
		}

		hOpenedFile.Close()
	}

	return engine.MissionUnknown()
}

/*
	MissionDifficulty is how hard the popfile being played is

The files are asked first, because a server owner listing a mission by name is
the only thing that can be right about a custom one. The name search after it is
a guess off the naming conventions the official missions follow.

//sp:name GetMissionDifficulty
*/
func MissionDifficulty() engine.MissionDifficulty {
	rsrc := engine.FindEntityByClassname(engine.MaxClients()+1, "tf_objective_resource")

	if rsrc == -1 {
		engine.LogError("GetMissionDifficulty: Could not find entity tf_objective_resource!")
		return engine.MissionUnknown()
	}

	missionName := engine.MvMPopfileName(rsrc)

	engine.ReplaceString(missionName, engine.PathMax(), "scripts/population/", "")
	engine.ReplaceString(missionName, engine.PathMax(), ".pop", "")

	missionType := MissionDifficultyFromName(missionName)

	if missionType == engine.MissionUnknown() {
		mapName := engine.CurrentMap()

		//nolint:gocritic // ifElseChain: the shipped function is this chain, and a port that reorders it into a switch cannot be compared against it
		if engine.StrEqualText(missionName, mapName) || engine.StrContains(missionName, "_norm_", false) != -1 {
			missionType = engine.MissionNormal()
		} else if engine.StrContains(missionName, "_intermediate", false) != -1 || engine.StrContains(missionName, "_int_", false) != -1 {
			missionType = engine.MissionIntermediate()
		} else if engine.StrContains(missionName, "_advanced", false) != -1 || engine.StrContains(missionName, "_adv_", false) != -1 {
			missionType = engine.MissionAdvanced()
		} else if engine.StrContains(missionName, "_expert", false) != -1 || engine.StrContains(missionName, "_exp_", false) != -1 {
			missionType = engine.MissionExpert()
		} else if engine.StrContains(missionName, "_night_", false) != -1 {
			missionType = engine.MissionNightmare()
		}
	}

	if engine.ManagerDebug().Bool() {
		engine.PrintToChatAll("GetMissionDifficulty: Current difficulty is %d", missionType)
	}

	return missionType
}
