package engine

/*
The names the registration table hands out.

Each of these is a function that lives in a generated file now, and the emitter
can only write a callback name it has in the same package. //sp:callback is how
it writes one it has not: the extern exists to be passed and is never called.
Nothing here means anything in a Go process.
*/

// CommandAddBots is the console command of that name.
//
//sp:callback Command_AddBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandAddBots(client int32, args int32) Outcome { return 0 }

// CommandBotPreferences is the console command of that name.
//
//sp:callback Command_BotPreferences
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandBotPreferences(client int32, args int32) Outcome { return 0 }

// CommandChooseBotClasses is the console command of that name.
//
//sp:callback Command_ChooseBotClasses
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandChooseBotClasses(client int32, args int32) Outcome { return 0 }

// CommandDumpCredits is the console command of that name.
//
//sp:callback Command_DumpCredits
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpCredits(client int32, args int32) Outcome { return 0 }

// CommandDumpFront is the console command of that name.
//
//sp:callback Command_DumpFront
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpFront(client int32, args int32) Outcome { return 0 }

// CommandDumpHats is the console command of that name.
//
//sp:callback Command_DumpHats
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpHats(client int32, args int32) Outcome { return 0 }

// CommandDumpMedic is the console command of that name.
//
//sp:callback Command_DumpMedic
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpMedic(client int32, args int32) Outcome { return 0 }

// CommandDumpNest is the console command of that name.
//
//sp:callback Command_DumpNest
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpNest(client int32, args int32) Outcome { return 0 }

// CommandDumpSpawnNav is the console command of that name.
//
//sp:callback Command_DumpSpawnNav
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpSpawnNav(client int32, args int32) Outcome { return 0 }

// CommandDumpSpot is the console command of that name.
//
//sp:callback Command_DumpSpot
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpSpot(client int32, args int32) Outcome { return 0 }

// CommandDumpUpgrades is the console command of that name.
//
//sp:callback Command_DumpUpgrades
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandDumpUpgrades(client int32, args int32) Outcome { return 0 }

// CommandJoinBluePlayWithBots is the console command of that name.
//
//sp:callback Command_JoinBluePlayWithBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandJoinBluePlayWithBots(client int32, args int32) Outcome { return 0 }

// CommandRecoverSpawnBots is the console command of that name.
//
//sp:callback Command_RecoverSpawnBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandRecoverSpawnBots(client int32, args int32) Outcome { return 0 }

// CommandRedoBotTeamLineup is the console command of that name.
//
//sp:callback Command_RedoBotTeamLineup
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandRedoBotTeamLineup(client int32, args int32) Outcome { return 0 }

// CommandRemoveAllBots is the console command of that name.
//
//sp:callback Command_RemoveAllBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandRemoveAllBots(client int32, args int32) Outcome { return 0 }

// CommandRequestExtraBot is the console command of that name.
//
//sp:callback Command_RequestExtraBot
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandRequestExtraBot(client int32, args int32) Outcome { return 0 }

// CommandRerollNewBotTeamComposition is the console command of that name.
//
//sp:callback Command_RerollNewBotTeamComposition
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandRerollNewBotTeamComposition(client int32, args int32) Outcome { return 0 }

// CommandReseatBots is the console command of that name.
//
//sp:callback Command_ReseatBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandReseatBots(client int32, args int32) Outcome { return 0 }

// CommandShowBotChances is the console command of that name.
//
//sp:callback Command_ShowBotChances
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandShowBotChances(client int32, args int32) Outcome { return 0 }

// CommandShowNewBotTeamComposition is the console command of that name.
//
//sp:callback Command_ShowNewBotTeamComposition
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandShowNewBotTeamComposition(client int32, args int32) Outcome { return 0 }

// CommandStopManagingBots is the console command of that name.
//
//sp:callback Command_StopManagingBots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandStopManagingBots(client int32, args int32) Outcome { return 0 }

// CommandViewBotUpgrades is the console command of that name.
//
//sp:callback Command_ViewBotUpgrades
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandViewBotUpgrades(client int32, args int32) Outcome { return 0 }

// CommandVotebots is the console command of that name.
//
//sp:callback Command_Votebots
//nolint:revive // unused-parameter: a name handed to a registration, never called
func CommandVotebots(client int32, args int32) Outcome { return 0 }

// ConVarChangedBotLineupMode is the convar hook of that name.
//
//sp:callback ConVarChanged_BotLineupMode
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ConVarChangedBotLineupMode(convar ConVar, before Text, after Text) {}

// ConVarChangedDefenderTeamSize is the convar hook of that name.
//
//sp:callback ConVarChanged_DefenderTeamSize
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ConVarChangedDefenderTeamSize(convar ConVar, before Text, after Text) {}

// ConVarChangedManagerMode is the convar hook of that name.
//
//sp:callback ConVarChanged_ManagerMode
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ConVarChangedManagerMode(convar ConVar, before Text, after Text) {}

// ConVarChangedTeamComposition is the convar hook of that name.
//
//sp:callback ConVarChanged_TeamComposition
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ConVarChangedTeamComposition(convar ConVar, before Text, after Text) {}

// ListenerTournamentPlayerReadystate is the ready panel's listener.
//
//sp:callback Listener_TournamentPlayerReadystate
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ListenerTournamentPlayerReadystate(client int32, command Text, argc int32) Outcome { return 0 }

// ListenerVoiceMenu is the voice menu's listener. Still in the plugin.
//
//sp:callback Listener_VoiceMenu
//nolint:revive // unused-parameter: a name handed to a registration, never called
func ListenerVoiceMenu(client int32, command Text, argc int32) Outcome { return 0 }

// SoundHookGeneral is the sound hook.
//
//sp:callback SoundHook_General
//nolint:revive // unused-parameter: a name handed to a registration, never called
func SoundHookGeneral(clients [101]int32, numClients int32, sample Text, entity int32, channel int32, volume float32, level int32, pitch int32, flags int32, soundEntry Text, seed int32) Outcome {
	return 0
}
