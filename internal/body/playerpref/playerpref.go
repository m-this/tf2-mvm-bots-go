/*
Package playerpref is source/redbots3/player_pref.sp: what the players asked the
bots to be, and the loadout a server can set over the top of it.

The file loading, the save timer and the menus stay in the plugin. What is here
is the reading and the writing: which class a player wants a bot to play, which
weapon it carries, and which seat of the composition it fills.
*/
package playerpref

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// MaxPlayers is MAXPLAYERS, the highest seat a loadout may name.
//
//sp:name MAXPLAYERS
const MaxPlayers = 64

// ClassMaxNameLength is how long the longest class name is, heavyweapons plus a
// terminator.
//
//sp:name TF2_CLASS_MAX_NAME_LENGTH
const ClassMaxNameLength = 14

// The class a player will take a bot as, one bit each.
const (
	//sp:name PREF_FL_NONE
	PrefNone = 0
	//sp:name PREF_FL_SCOUT
	PrefScout = 1 << 0
	//sp:name PREF_FL_SOLDIER
	PrefSoldier = 1 << 1
	//sp:name PREF_FL_PYRO
	PrefPyro = 1 << 2
	//sp:name PREF_FL_DEMO
	PrefDemo = 1 << 3
	//sp:name PREF_FL_HEAVY
	PrefHeavy = 1 << 4
	//sp:name PREF_FL_ENGINEER
	PrefEngineer = 1 << 5
	//sp:name PREF_FL_MEDIC
	PrefMedic = 1 << 6
	//sp:name PREF_FL_SNIPER
	PrefSniper = 1 << 7
	//sp:name PREF_FL_SPY
	PrefSpy = 1 << 8
)

// ItemDefDefault is the stock item, which has no definition index of its own.
const ItemDefDefault = -1

/*
The seat of the composition a bot fills, counted from 1, and 0 for a bot that
fills none.

The composition is an ordered list, so the third name in it is a seat somebody
sits in and not just another engineer. Looking a loadout up by class alone hands
every engineer on the team the same two weapons.
*/
//
//sp:name m_iBotSeat
var botSeat [Slots]int32

// The seats asked for, waiting for bots the server has not created yet.
//
//sp:name m_adtPendingBotSeats
var pendingBotSeats engine.List

/*
The two config files and the admin override, owned here because this file reads
them and a global has to be declared before its first reader.

player_pref.sp still loads both files; a config the server did not write reads
as null.
*/

// ServerLoadout is configs/defenderbots/loadout.cfg.
//
//sp:name m_kvServerLoadout
var serverLoadout engine.KeyValues

// PlayerPrefData is what the players asked for.
//
//sp:name m_kvPlayerPrefData
var playerPrefData engine.KeyValues

// PlayerForcedPref is the client an admin told the mod to read every preference
// from, and -1 the rest of the time.
//
//sp:name g_iPlayerForcedPref
var playerForcedPref int32 = -1

// IsValidLoadoutSeat says the file named a seat a bot can actually fill.
//
//sp:name IsValidLoadoutSeat
func IsValidLoadoutSeat(seat int32) bool {
	return seat >= 1 && seat <= MaxPlayers
}

/*
WarnAboutInvalidLoadoutSeats complains about the seats the file names that no bot
can ever fill.

A seat out of range is a typo, and nothing else says so: the bot wears the
loadout of its class instead, which reads as the mod ignoring the file rather
than the file asking for seat 0.
*/
//
//sp:name WarnAboutInvalidLoadoutSeats
func WarnAboutInvalidLoadoutSeats() {
	serverLoadout.Rewind()

	if !serverLoadout.JumpToKey("seats", false) || !serverLoadout.GotoFirstSubKey() {
		serverLoadout.Rewind()
		return
	}

	for {
		section := serverLoadout.SectionName()

		if !IsValidLoadoutSeat(engine.StringToInt(section)) {
			engine.LogError("Config_LoadServerLoadout: seat \"%s\" is not between 1 and %d, ignoring it", section, MaxPlayers)
		}

		if !serverLoadout.GotoNextKey() {
			break
		}
	}

	serverLoadout.Rewind()
}

/*
JumpToServerLoadoutSeat stands on the block the file writes for one seat, when it
writes one this bot may wear.

A seat answers only for the class it names. The composition gets retyped between
waves, so the seat that was an engineer's is now a medic's, and that medic is
better off with the medic block than with an engineer's wrangler.
*/
//
//sp:name JumpToServerLoadoutSeat
func JumpToServerLoadoutSeat(seat int32, class string) bool {
	if !IsValidLoadoutSeat(seat) {
		return false
	}

	_, section := engine.IntToString(seat)

	if !serverLoadout.JumpToKey("seats", false) || !serverLoadout.JumpToKeyText(section, false) {
		serverLoadout.Rewind()
		return false
	}

	seatClass := serverLoadout.String("class")

	if engine.StrEqualFolded(seatClass, class, false) {
		return true
	}

	serverLoadout.Rewind()

	return false
}

/*
GetServerLoadoutWeapon is what the file says this bot carries in that slot.

The seat decides the whole loadout when it names this bot, and the class decides
it otherwise. The seat is the more specific of the two, so it answers for every
slot the way the file itself does: a slot it leaves out is the stock weapon, not
the class block's answer.
*/
//
//sp:name GetServerLoadoutWeapon
func GetServerLoadoutWeapon(seat int32, class string, slot string) int32 {
	serverLoadout.Rewind()

	if !JumpToServerLoadoutSeat(seat, class) && !serverLoadout.JumpToKey(class, false) {
		return ItemDefDefault
	}

	weaponIndex := serverLoadout.Num(slot, ItemDefDefault)
	serverLoadout.Rewind()

	return weaponIndex
}

/*
NoteBotSeatPending remembers a seat asked for, waiting for the bot the server has
not created yet.

tf_bot_add is a console command, so the bot does not exist when its seat is
decided. Nobody came for the oldest one when the list is full, which means that
tf_bot_add was refused: one wrong loadout beats a list that only grows.
*/
//
//sp:name NoteBotSeatPending
func NoteBotSeatPending(seat int32) {
	if pendingBotSeats == engine.NoList() {
		pendingBotSeats = engine.NewList()
	}

	if pendingBotSeats.Length() >= MaxPlayers {
		pendingBotSeats.Erase(0)
	}

	pendingBotSeats.Push(seat)
}

// TakeBotSeat gives the bot that just entered the seat at the front.
//
//sp:name TakeBotSeat
func TakeBotSeat(client int32) {
	botSeat[client] = 0

	if pendingBotSeats == engine.NoList() || pendingBotSeats.Length() < 1 {
		return
	}

	botSeat[client] = pendingBotSeats.Get(0)
	pendingBotSeats.Erase(0)
}

// ForgetBotSeat drops it: whoever holds this client index next is another bot.
//
//sp:name ForgetBotSeat
func ForgetBotSeat(client int32) {
	botSeat[client] = 0
}

// GetClassPreferencesFlags is every class this player will take a bot as.
//
//sp:name GetClassPreferencesFlags
func GetClassPreferencesFlags(client int32) int32 {
	found, steamID := engine.ClientAuthID(client, engine.AuthIDSteam3())

	if !found {
		engine.LogError("GetClassPreferencesFlags: failed to get Steam ID for %L", client)
		return PrefNone
	}

	flags := int32(PrefNone)

	playerPrefData.JumpToKeyText(steamID, true)
	playerPrefData.JumpToKey("class", true)

	if playerPrefData.Num("scout", 0) == 1 {
		flags |= PrefScout
	}

	if playerPrefData.Num("soldier", 0) == 1 {
		flags |= PrefSoldier
	}

	if playerPrefData.Num("pyro", 0) == 1 {
		flags |= PrefPyro
	}

	if playerPrefData.Num("demoman", 0) == 1 {
		flags |= PrefDemo
	}

	if playerPrefData.Num("heavyweapons", 0) == 1 {
		flags |= PrefHeavy
	}

	if playerPrefData.Num("engineer", 0) == 1 {
		flags |= PrefEngineer
	}

	if playerPrefData.Num("medic", 0) == 1 {
		flags |= PrefMedic
	}

	if playerPrefData.Num("sniper", 0) == 1 {
		flags |= PrefSniper
	}

	if playerPrefData.Num("spy", 0) == 1 {
		flags |= PrefSpy
	}

	playerPrefData.Rewind()

	return flags
}

// SetClassPreferences writes one class answer down.
//
//sp:name SetClassPreferences
func SetClassPreferences(client int32, class string, value int32) {
	found, steamID := engine.ClientAuthID(client, engine.AuthIDSteam3())

	if !found {
		engine.LogError("SetClassPreferences: failed to get Steam ID for %L", client)
		return
	}

	playerPrefData.JumpToKeyText(steamID, true)
	playerPrefData.JumpToKey("class", true)
	playerPrefData.SetNum(class, value)
	playerPrefData.Rewind()
}

// GetWeaponPreference is the item definition index this player wants in that
// slot.
//
//sp:name GetWeaponPreference
func GetWeaponPreference(client int32, class string, slot string) int32 {
	found, steamID := engine.ClientAuthID(client, engine.AuthIDSteam3())

	if !found {
		engine.LogError("GetWeaponPreference: failed to get Steam ID for %L", client)
		return ItemDefDefault
	}

	var weaponIndex int32

	playerPrefData.JumpToKeyText(steamID, true)
	playerPrefData.JumpToKey("loadout", true)
	playerPrefData.JumpToKey(class, true)
	weaponIndex = playerPrefData.Num(slot, ItemDefDefault)
	playerPrefData.Rewind()

	return weaponIndex
}

/*
GetPreferredWeaponForClass is the weapon a bot of that class carries in that slot.

The server's own loadout answers first when there is one. Otherwise the players
who are in and on red have a say each, and one of their answers is drawn: drawing
rather than counting makes the choice proportional instead of majority.
*/
//
//sp:name GetPreferredWeaponForClass
func GetPreferredWeaponForClass(class string, slot string, client int32) int32 {
	if serverLoadout != engine.NoKeyValues() {
		return GetServerLoadoutWeapon(botSeat[client], class, slot)
	}

	if playerForcedPref != -1 {
		// Preference forced by admin, probably wants to use his or
		// someone else's.
		return GetWeaponPreference(playerForcedPref, class, slot)
	}

	weaponPref := engine.NewList()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && IsValidForBotPreferences(i) {
			prefWeapon := GetWeaponPreference(i, class, slot)

			if prefWeapon != ItemDefDefault {
				weaponPref.Push(prefWeapon)
			}
		}
	}

	// No preferences found, probably no human red players.
	if weaponPref.Length() < 1 {
		weaponPref.Close()
		return engine.RandomWeaponForClass(class, slot)
	}

	itemDefIndex := weaponPref.Get(engine.RandomInt(0, weaponPref.Length()-1))

	weaponPref.Close()

	return itemDefIndex
}

// SetWeaponPreference writes one weapon answer down.
//
//sp:name SetWeaponPreference
func SetWeaponPreference(client int32, class string, slot string, value int32) {
	_, steamID := engine.ClientAuthID(client, engine.AuthIDSteam3())

	playerPrefData.JumpToKeyText(steamID, true)
	playerPrefData.JumpToKey("loadout", true)
	playerPrefData.JumpToKey(class, true)
	playerPrefData.SetNum(slot, value)
	playerPrefData.Rewind()
}

// IsValidForBotPreferences says this player has an influence on what the bots
// are.
//
//sp:name IsValidForBotPreferences
func IsValidForBotPreferences(client int32) bool {
	return !engine.IsFakeClient(client) && engine.ClientTeam(client) == engine.TeamRed()
}

// CollectPlayerBotClassPreferences is every class every player asked for, one
// entry per player per class, which is what makes the draw proportional.
//
//sp:name CollectPlayerBotClassPreferences
func CollectPlayerBotClassPreferences(stringList engine.List) {
	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && IsValidForBotPreferences(i) {
			classFlags := GetClassPreferencesFlags(i)

			if classFlags&PrefScout != 0 {
				stringList.PushString("scout")
			}

			if classFlags&PrefSoldier != 0 {
				stringList.PushString("soldier")
			}

			if classFlags&PrefPyro != 0 {
				stringList.PushString("pyro")
			}

			if classFlags&PrefDemo != 0 {
				stringList.PushString("demoman")
			}

			if classFlags&PrefHeavy != 0 {
				stringList.PushString("heavyweapons")
			}

			if classFlags&PrefEngineer != 0 {
				stringList.PushString("engineer")
			}

			if classFlags&PrefMedic != 0 {
				stringList.PushString("medic")
			}

			if classFlags&PrefSniper != 0 {
				stringList.PushString("sniper")
			}

			if classFlags&PrefSpy != 0 {
				stringList.PushString("spy")
			}
		}
	}
}

// AddBotsBasedOnPreferences adds that many bots, drawing each one's class from
// what the players asked for.
//
//sp:name AddBotsBasedOnPreferences
func AddBotsBasedOnPreferences(amount int32) {
	// Can't add any more if the server is full.
	if engine.IsServerFull() {
		return
	}

	engine.PrintToChatAll("%s Adding %d bot(s)...", engine.PluginPrefix(), amount)

	if amount <= 0 {
		return
	}

	classPref := engine.NewStringList(ClassMaxNameLength)

	// Get the players' class preferences.
	CollectPlayerBotClassPreferences(classPref)

	if classPref.Length() > 0 {
		for i := int32(1); i <= amount; i++ {
			// Now pick a random class from preferences. This makes
			// class choice proportional, rather than majority.
			class := classPref.GetString(engine.RandomInt(0, classPref.Length()-1))

			engine.AddDefenderTFBotOf(1, class, "red", "expert")
		}
	} else {
		// Nobody had preferences, just add random bots.
		engine.AddRandomDefenderBots(amount)
	}

	classPref.Close()
}

// Where the players' preferences are kept between maps.
//
//sp:name g_sPlayerPrefPath
var playerPrefPath [256]byte

/*
ConfigLoadServerLoadout reads the server's own loadout file, if it wrote one.

The pending seats go with it: a map change means the seats the last map asked
for belong to bots that will never enter.
*/
//
//sp:name Config_LoadServerLoadout
func ConfigLoadServerLoadout() {
	serverLoadout.Close()

	pendingBotSeats.Close()

	filePath := engine.BuildPath("configs/defenderbots/loadout.cfg")

	if !engine.FileExists(filePath) {
		return
	}

	serverLoadout = engine.NewKeyValues("loadout")

	if !serverLoadout.ImportFromFile(filePath) {
		engine.LogError("Config_LoadServerLoadout: Could not read %s!", filePath)
		serverLoadout.Close()
		return
	}

	WarnAboutInvalidLoadoutSeats()
}

// TimerSavePrefData writes the preferences to disk every twenty seconds, so a
// crash costs at most that.
//
//sp:name Timer_SavePrefData
//sp:public
//nolint:revive // unused-parameter: the timer handle is SourceMod's
func TimerSavePrefData(timer engine.Timer) engine.Outcome {
	if !playerPrefData.ExportToFile(engine.TextOfPath(playerPrefPath)) {
		engine.LogError("Timer_SavePrefData: Failed to save player preference data!")
		engine.PrintToChatAll("%s ERROR: Player preference data failed to save!", engine.PluginPrefix())
		return engine.PluginContinue()
	}

	if engine.ManagerDebug().Bool() {
		engine.PrintToServer("%s Saved player preference data.", engine.PluginPrefix())
	}

	return engine.PluginContinue()
}

// LoadPreferencesData reads them back at load and starts the save timer.
//
//sp:name LoadPreferencesData
func LoadPreferencesData() {
	playerPrefData = engine.NewKeyValues("PlayerBotPreferences")
	playerPrefData.ImportFromFile(engine.TextOfPath(playerPrefPath))

	engine.CreateTimer(20.0, TimerSavePrefData, engine.Default(), engine.TimerRepeat())
}

/*
ShowCurrentBotClassChances is each class's share of the draw, as a panel.

A share rather than a count, because what a player wants to know is how likely
a class is, and that depends on what everybody else asked for as much as on
what they did.
*/
//
//sp:name ShowCurrentBotClassChances
//sp:default client -1
func ShowCurrentBotClassChances(client int32) {
	// Each index is a class, 0 = scout, 1 = soldier, and so on. Float because
	// the percentage below divides by the total.
	var classChoiceCount [9]float32

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && IsValidForBotPreferences(i) {
			classFlags := GetClassPreferencesFlags(i)

			for c := int32(0); c < 9; c++ {
				if classFlags&PrefFlagOf(c) != 0 {
					classChoiceCount[c]++
				}
			}
		}
	}

	var totalChoices float32

	for i := int32(0); i < 9; i++ {
		totalChoices += classChoiceCount[i]
	}

	if totalChoices == 0.0 {
		if client > 0 {
			engine.PrintHintText(client, "Nobody has any preferences!")
		} else {
			engine.PrintHintTextToAll("Nobody has any preferences!")
		}

		return
	}

	// Like before, each index is a class. The share is the times that class
	// was chosen over the total of every choice.
	var classPercents [9]float32

	for i := int32(0); i < 9; i++ {
		classPercents[i] = (classChoiceCount[i] / totalChoices) * 100
	}

	if client > 0 {
		engine.DisplayPanelBotPercentages(client, classPercents)
	} else {
		for i := int32(1); i <= engine.MaxClients(); i++ {
			if engine.IsClientInGame(i) {
				engine.DisplayPanelBotPercentages(i, classPercents)
			}
		}
	}
}

// PrefFlagOf is the preference bit for one class, indexed the way the panel
// counts them: 0 is the scout. The shipped file wrote nine ifs; the bits are
// consecutive powers of two, so one shift is the same nine answers.
//
//sp:name Go_PrefFlagOf
func PrefFlagOf(index int32) int32 {
	return 1 << index
}
