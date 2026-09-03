package engine

/*
Seating the team: the modes, the sizes, and the console command that makes a
bot.
*/

// ManageCalls are the answers.
type ManageCalls struct {
	ServerCommand                    func(format string, args []any)
	ThrowError                       func(format string, args []any)
	BotsEnabled                      func() bool
	SetBotsEnabled                   func(on bool)
	MapComposition                   func() Text
	AddBotsFromChosenTeamComposition func()
	AddBotsFromTeamComposition       func(count int32) int32
	AddBotsBasedOnPreferences        func(count int32)
	PickAllowedBotClass              func(wanted Text, buffer Text, maxlen int32)
	HumanAndDefenderBotCount         func(team Team) int32
	ExtendUpgradeTimeForNewBots      func()
	KickClientText                   func(client int32, reason Text)
	MakePlayerDance                  func(client int32)
}

var manages ManageCalls

// InstallManages puts a set of answers behind them.
func InstallManages(c ManageCalls) func() {
	previous := manages
	Fill(&c)
	manages = c
	return func() { manages = previous }
}

// ServerCommand runs a line on the server console.
//
//sp:native ServerCommand
func ServerCommand(format string, args ...any) { manages.ServerCommand(format, args) }

// ThrowError aborts the callback with a logged message.
//
//sp:native ThrowError
func ThrowError(format string, args ...any) { manages.ThrowError(format, args) }

// ChooseString is the ternary between two written strings.
//
//sp:choice ?:
func ChooseString(cond bool, yes string, no string) string {
	if cond {
		return yes
	}
	return no
}

// The manager modes register.go does not already declare.

// ManagerModeManualBots is MANAGER_MODE_MANUAL_BOTS.
//
//sp:global MANAGER_MODE_MANUAL_BOTS
func ManagerModeManualBots() int32 { return 0 }

// The lineup modes register.go does not already declare.

// LineupModeRandom is BOT_LINEUP_MODE_RANDOM.
//
//sp:global BOT_LINEUP_MODE_RANDOM
func LineupModeRandom() int32 { return 0 }

// LineupModePreference is BOT_LINEUP_MODE_PREFERENCE.
//
//sp:global BOT_LINEUP_MODE_PREFERENCE
func LineupModePreference() int32 { return 1 }

// LineupModePreferenceChoose is BOT_LINEUP_MODE_PREFERENCE_CHOOSE.
//
//sp:global BOT_LINEUP_MODE_PREFERENCE_CHOOSE
func LineupModePreferenceChoose() int32 { return 3 }

// DefenderTeamSize is redbots_manager_defender_team_size, how many seats RED
// has.
//
//sp:global redbots_manager_defender_team_size
func DefenderTeamSize() ConVar { return 0 }

// TeamComposition is redbots_manager_team_composition, the lineup somebody
// typed into the console, empty when nobody did.
//
//sp:global redbots_manager_team_composition
func TeamComposition() ConVar { return 0 }

// BotsEnabled is g_bBotsEnabled, whether the manager is on.
//
//sp:global g_bBotsEnabled
func BotsEnabled() bool { return manages.BotsEnabled() }

// SetBotsEnabled writes it.
//
//sp:globalset g_bBotsEnabled
func SetBotsEnabled(on bool) { manages.SetBotsEnabled(on) }

// MapComposition is the lineup the map config wants, read off the record.
//
//sp:global g_arrMapConfig.strComposition
func MapComposition() Text { return manages.MapComposition() }

/*
What the seating still leaves in the plugin.

The three below walk the chosen lineup, the typed lineup and the preference
file, and PickAllowedBotClass explodes a comma list into a two dimensional
char array, which the subset has no shape for yet.
*/

// AddBotsFromChosenTeamComposition seats the lineup the vote chose.
//
//sp:body AddBotsFromChosenTeamComposition
func AddBotsFromChosenTeamComposition() { manages.AddBotsFromChosenTeamComposition() }

// AddBotsFromTeamComposition seats the typed or map lineup and says how many
// it seated.
//
//sp:body AddBotsFromTeamComposition
func AddBotsFromTeamComposition(count int32) int32 { return manages.AddBotsFromTeamComposition(count) }

// AddBotsBasedOnPreferences seats what the players asked for.
//
//sp:body AddBotsBasedOnPreferences
func AddBotsBasedOnPreferences(count int32) { manages.AddBotsBasedOnPreferences(count) }

// PickAllowedBotClass writes the wanted class, or a random allowed one when
// the blacklist refuses it.
//
//sp:body PickAllowedBotClass
func PickAllowedBotClass(wanted Text, buffer Text, maxlen int32) {
	manages.PickAllowedBotClass(wanted, buffer, maxlen)
}

// HumanAndDefenderBotCount is everybody on the team who is not somebody
// else's bot. Ported, roster_counts.
//
//sp:body GetHumanAndDefenderBotCount
func HumanAndDefenderBotCount(team Team) int32 { return manages.HumanAndDefenderBotCount(team) }

// ExtendUpgradeTimeForNewBots gives a late bot enough break to shop in.
// Ported, roster_counts.
//
//sp:body ExtendUpgradeTimeForNewBots
func ExtendUpgradeTimeForNewBots() { manages.ExtendUpgradeTimeForNewBots() }

// KickClientText is KickClient handed a buffer: RemoveAllDefenderBots takes
// the reason as a parameter and passes it straight through.
//
//sp:native KickClient
func KickClientText(client int32, reason Text) { manages.KickClientText(client, reason) }

// MakePlayerDance is the final wave's send-off, in place of a kick. Ported,
// manage.
//
//sp:body MakePlayerDance
func MakePlayerDance(client int32) { manages.MakePlayerDance(client) }
