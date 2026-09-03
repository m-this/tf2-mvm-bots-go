/*
Package declarations is what tf2_defenderbots.sp used to declare: the two mode
enums, the mod's globals and every convar handle it owns.

Nothing here is read from Go. It exists so that adding a convar or a per-client
field is a Go edit rather than a SourcePawn one, which is what "SourcePawn is a
build artifact" has to mean to be worth anything.

Three things stayed behind, because the generator has no form for them: the
three enum structs, which carry methods, the plugin's own myinfo, and the
include list, which is its build order.
*/
package declarations

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is MAXPLAYERS + 1, the client array size.
const Slots = 65

// ManagerMode is which of the three ways the mod runs.
const (
	//sp:name MANAGER_MODE_MANUAL_BOTS
	ManagerModeManualBots = 0
	//sp:name MANAGER_MODE_READY_BOTS
	ManagerModeReadyBots = 1
	//sp:name MANAGER_MODE_AUTO_BOTS
	ManagerModeAutoBots = 2
)

// BotLineupMode is how the bot team's classes are decided.
const (
	//sp:name BOT_LINEUP_MODE_RANDOM
	BotLineupModeRandom = 0
	//sp:name BOT_LINEUP_MODE_PREFERENCE
	BotLineupModePreference = 1
	//sp:name BOT_LINEUP_MODE_CHOOSE
	BotLineupModeChoose = 2
	//sp:name BOT_LINEUP_MODE_PREFERENCE_CHOOSE
	BotLineupModePreferenceChoose = 3
)

/*
The mod's own state, none of it per client.
*/

//sp:name g_bLateLoad
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var lateLoad bool

//sp:name g_bBotsEnabled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botsEnabled bool

//sp:name g_flAddingBotTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var addingBotTime float32

//sp:name g_flNextReadyTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextReadyTime float32

//sp:name g_iDetonatingPlayer
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var detonatingPlayer int32 = -1

//sp:name g_adtChosenBotClasses
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var chosenBotClasses engine.List

/*
chosenBotSeats is the seat of the team composition each of those classes was
named in, index for index.

Only the composition names seats, and the lineup mode's classes are appended
after its own, so this list is the shorter of the two and an index past its end
is a bot sitting in no seat.
*/
//
//sp:name g_adtChosenBotSeats
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var chosenBotSeats engine.List

//sp:name g_bBotClassesLocked
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botClassesLocked bool

//sp:name g_iUIDBotSummoner
//nolint:unused,revive // emitted, not read from Go, and var-declaration: the shipped declaration writes the zero out and the emitter follows it
var uidBotSummoner int32 = 0

//sp:name g_bAllowBotTeamRedo
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var allowBotTeamRedo bool

/*
One entry per client, for the bots.
*/

//sp:name g_bIsDefenderBot
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var isDefenderBot [Slots]bool

//sp:name g_bIsBeingRevived
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var isBeingRevived [Slots]bool

//sp:name g_bHasUpgraded
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hasUpgraded [Slots]bool

/*
shoppedThisBreak is whether this bot has done its shopping since the last wave
started.

Readiness used to stand in for this, and it stopped being able to: with a person
on RED every bot is held ready from the first frame of the break so the person
alone decides when the wave starts. Every "has he finished preparing" test that
read the ready flag then answered yes before he had bought anything, which
skipped the shopping trip and, from the second wave on, skipped it for the rest
of the mission.
*/
//
//sp:name g_bShoppedThisBreak
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var shoppedThisBreak [Slots]bool

//sp:name g_arrExtraButtons
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var extraButtons [Slots]engine.ButtonInput

//sp:name m_flDeadRethinkTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var deadRethinkTime [Slots]float32

//sp:name g_iBuybackNumber
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var buybackNumber [Slots]int32

//sp:name g_iBuyUpgradesNumber
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var buyUpgradesNumber [Slots]int32

//sp:name m_flNextSnipeFireTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextSnipeFireTime [Slots]float32

//sp:name m_flNextRollTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nextRollTime [Slots]float32

/*
One entry per client, for the people.
*/

//sp:name g_bChoosingBotClasses
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var choosingBotClasses [Slots]bool

//sp:name g_flEnableBotsCooldown
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var enableBotsCooldown [Slots]float32

//sp:name m_flLastReadyInputTime
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var lastReadyInputTime [Slots]float32

/*
The map's config, and the world.
*/

//sp:name g_arrMapConfig
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var mapConfig engine.MapConfigRecord

//sp:name m_adtBotNames
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botNames engine.List

//sp:name g_iPopulationManager
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var populationManager int32 = -1

//sp:name g_pMannVsMachineUpgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var mannVsMachineUpgrades engine.Address

/*
Every convar this mod owns, and the seven of the game's own it looks up.
*/

//sp:name redbots_manager_debug
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDebug engine.ConVar

//sp:name redbots_manager_debug_actions
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDebugActions engine.ConVar

//sp:name redbots_manager_mode
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerMode engine.ConVar

//sp:name redbots_manager_bot_lineup_mode
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotLineupMode engine.ConVar

//sp:name redbots_manager_use_custom_loadouts
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerUseCustomLoadouts engine.ConVar

//sp:name redbots_manager_class_blacklist
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerClassBlacklist engine.ConVar

//sp:name redbots_manager_team_composition
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerTeamComposition engine.ConVar

//sp:name redbots_manager_kick_bots
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerKickBots engine.ConVar

//sp:name redbots_manager_min_players
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerMinPlayers engine.ConVar

//sp:name redbots_manager_defender_team_size
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerDefenderTeamSize engine.ConVar

//sp:name redbots_manager_ready_cooldown
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerReadyCooldown engine.ConVar

//sp:name redbots_manager_keep_bot_upgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerKeepBotUpgrades engine.ConVar

//sp:name redbots_manager_bot_upgrade_interval
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotUpgradeInterval engine.ConVar

//sp:name redbots_manager_engineer_nest_depth
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestDepth engine.ConVar

//sp:name redbots_manager_engineer_nest_relocate
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestRelocate engine.ConVar

//sp:name redbots_manager_engineer_nest_relocate_score_gain_min
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerEngineerNestRelocateScoreGainMin engine.ConVar

//sp:name redbots_manager_bot_use_upgrades
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotUseUpgrades engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecovery engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery_radius
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecoveryRadius engine.ConVar

//sp:name redbots_manager_spawn_nav_recovery_time
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerSpawnNavRecoveryTime engine.ConVar

//sp:name redbots_manager_bot_hats
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHats engine.ConVar

//sp:name redbots_manager_bot_hat_effects
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHatEffects engine.ConVar

//sp:name redbots_manager_bot_buyback_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBuybackChance engine.ConVar

//sp:name redbots_manager_bot_buy_upgrades_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBuyUpgradesChance engine.ConVar

//sp:name redbots_manager_bot_max_tank_attackers
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotMaxTankAttackers engine.ConVar

//sp:name redbots_manager_bot_aim_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotAimSkill engine.ConVar

//sp:name redbots_manager_bot_reflect_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotReflectSkill engine.ConVar

//sp:name redbots_manager_bot_reflect_chance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotReflectChance engine.ConVar

//sp:name redbots_manager_bot_backstab_skill
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotBackstabSkill engine.ConVar

//sp:name redbots_manager_bot_hear_spy_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotHearSpyRange engine.ConVar

//sp:name redbots_manager_bot_notice_spy_time
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotNoticeSpyTime engine.ConVar

//sp:name redbots_manager_extra_bots
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerExtraBots engine.ConVar

//sp:name redbots_manager_bot_request_credits
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotRequestCredits engine.ConVar

//sp:name redbots_manager_bot_rtd_variance
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var redbotsManagerBotRtdVariance engine.ConVar

//sp:name nb_blind
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var nbBlind engine.ConVar

//sp:name tf_bot_path_lookahead_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotPathLookaheadRange engine.ConVar

//sp:name tf_bot_health_critical_ratio
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthCriticalRatio engine.ConVar

//sp:name tf_bot_health_ok_ratio
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthOkRatio engine.ConVar

//sp:name tf_bot_ammo_search_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotAmmoSearchRange engine.ConVar

//sp:name tf_bot_health_search_far_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthSearchFarRange engine.ConVar

//sp:name tf_bot_health_search_near_range
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var tfBotHealthSearchNearRange engine.ConVar

/*
botTeamCompositions is the three preset lineups AddBotsWithPresetTeamComp draws
from.

Both are dead: nothing calls that function and nothing else reads this table,
which is mvm-z83.80. They stayed because a port does not delete what it does not
understand.
*/
//
//sp:name g_sBotTeamCompositions
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var botTeamCompositions = [3][6]string{
	{"scout", "soldier", "demoman", "heavyweapons", "engineer", "medic"},
	{"scout", "heavyweapons", "heavyweapons", "heavyweapons", "engineer", "sniper"},
	{"scout", "heavyweapons", "heavyweapons", "pyro", "engineer", "demoman"},
}

// NestZoneLength is how long a zone name may be.
//
//sp:name NEST_ZONE_LENGTH
const NestZoneLength = 24

// CompositionLength is how long the lineup a map names may be.
//
//sp:name Go_CompositionLength
const CompositionLength = 128

/*
MapConfiguration is everything one map's config file says.

It is one record shared by everything that reads it, which is why it keeps the
plugin's own names for the type and for every field.
*/
//
//sp:name esMapConfiguration
type MapConfiguration struct {
	SniperSpot       engine.List `sp:"adtSniperSpot"`
	EngineerNestSpot engine.List `sp:"adtEngineerNestLocation"`
	// One zone name per nest spot, same order. Empty when the map does not name one.
	EngineerNestZone engine.List `sp:"adtEngineerNestZone"`
	TeleporterIn     engine.List `sp:"adtTeleporterEntranceLocation"`
	TeleporterOut    engine.List `sp:"adtTeleporterExitLocation"`
	DispenserSpot    engine.List `sp:"adtDispenserLocation"`
	// One zone name per dispenser spot, same order, so a nest in a zone takes
	// the dispenser in it.
	DispenserZone engine.List `sp:"adtDispenserZone"`
	// Nests that only apply to a wave with a tank in it, and nests that only
	// apply to one without.
	NestTankOnly engine.List `sp:"adtNestTankOnlyLocation"`
	NestNoTank   engine.List `sp:"adtNestNoTankLocation"`
	// The lineup this map wants, comma separated, empty when it does not care.
	Composition [CompositionLength]byte `sp:"strComposition"`
	/* MovingNests is whether the engineers are expected to pick the nest up and
	move it between waves.

	Mannhattan's gates move the front, and Rottenburg wants a different nest for
	a tank wave than for one without. On a map like that a disposable sentry
	covers the ground while the real one is in a toolbox, and is worth buying. On
	every other map it is a hundred and fifty credits for a second sentry nobody
	moves, which is what the guides mean when they say never. */
	MovingNests bool `sp:"bMovingNests"`
}

// Initialize makes every list the record holds.
//
//sp:name Initialize
func (c *MapConfiguration) Initialize() {
	c.SniperSpot = engine.NewListSized(3)
	c.EngineerNestSpot = engine.NewListSized(3)
	c.EngineerNestZone = engine.NewListSized(engine.ByteCountToCells(NestZoneLength))
	c.TeleporterIn = engine.NewListSized(3)
	c.TeleporterOut = engine.NewListSized(3)
	c.DispenserSpot = engine.NewListSized(3)
	c.DispenserZone = engine.NewListSized(engine.ByteCountToCells(NestZoneLength))
	c.NestTankOnly = engine.NewListSized(3)
	c.NestNoTank = engine.NewListSized(3)
}

// Reset empties them, and clears the two plain fields.
//
//sp:name Reset
func (c *MapConfiguration) Reset() {
	c.SniperSpot.Clear()
	c.EngineerNestSpot.Clear()
	c.EngineerNestZone.Clear()
	c.TeleporterIn.Clear()
	c.TeleporterOut.Clear()
	c.DispenserSpot.Clear()
	c.DispenserZone.Clear()
	c.NestTankOnly.Clear()
	c.NestNoTank.Clear()
	c.Composition[0] = 0
	c.MovingNests = false
}

/*
ButtonInputRecord is esButtonInput: the buttons a behaviour asked to be held
down or held off, and until when.

The two times are what makes it a record rather than two ints. A behaviour that
wants a key held for a moment says so once, and OnPlayerRunCmd holds it until
the clock says otherwise.
*/
//
//sp:name esButtonInput
type ButtonInputRecord struct {
	Press       int32   `sp:"iPress"`
	PressTime   float32 `sp:"flPressTime"`
	Release     int32   `sp:"iRelease"`
	ReleaseTime float32 `sp:"flReleaseTime"`
	KeySpeed    float32 `sp:"flKeySpeed"`
}

// Reset puts every field back to its starting value.
//
//sp:name Reset
func (b *ButtonInputRecord) Reset() {
	b.Press = 0
	b.PressTime = 0.0
	b.Release = 0
	b.ReleaseTime = 0.0
	b.KeySpeed = 0.0
}

// PressButtons holds those buttons down, for a time or until told otherwise.
//
//sp:name PressButtons
//sp:default duration -1.0
func (b *ButtonInputRecord) PressButtons(buttons int32, duration float32) {
	b.Press = buttons
	b.PressTime = engine.ChooseFloat(duration > 0.0, engine.GameTime()+duration, 0.0)
}

// ReleaseButtons holds those buttons off the same way.
//
//sp:name ReleaseButtons
//sp:default duration -1.0
func (b *ButtonInputRecord) ReleaseButtons(buttons int32, duration float32) {
	b.Release = buttons
	b.ReleaseTime = engine.ChooseFloat(duration > 0.0, engine.GameTime()+duration, 0.0)
}

/*
PluginBotRecord is esPluginBot: where a plugin-driven bot is walking to.

A goal is either a place or an entity and never both, which is what the two
setters enforce between them.
*/
//
//sp:name esPluginBot
type PluginBotRecord struct {
	Pathing        bool       `sp:"bPathing"`
	PathGoal       [3]float32 `sp:"vecPathGoal"`
	PathGoalEntity int32      `sp:"iPathGoalEntity"`
}

// Reset forgets where it was going.
//
//sp:name Reset
func (p *PluginBotRecord) Reset() {
	p.Pathing = false
	p.PathGoal = engine.NullVector()
	p.PathGoalEntity = -1
}

// HasPathGoalVector says the goal is a place.
//
//sp:name HasPathGoalVector
func (p *PluginBotRecord) HasPathGoalVector() bool {
	return !engine.VectorIsZero(p.PathGoal)
}

// HasPathGoalEntity says the goal is an entity.
//
//sp:name HasPathGoalEntity
func (p *PluginBotRecord) HasPathGoalEntity() bool {
	return p.PathGoalEntity != -1
}

// SetPathGoalVector aims it at a place.
//
//sp:name SetPathGoalVector
//sp:const vec
func (p *PluginBotRecord) SetPathGoalVector(vec [3]float32) {
	// You can only set one or the other, not both.
	p.PathGoalEntity = -1
	p.PathGoal = vec
}

// SetPathGoalEntity aims it at an entity.
//
//sp:name SetPathGoalEntity
func (p *PluginBotRecord) SetPathGoalEntity(entity int32) {
	p.PathGoal = engine.NullVector()
	p.PathGoalEntity = entity
}

//sp:name g_arrPluginBot
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var pluginBot [Slots]PluginBotRecord

// PluginPrefix is what the mod puts in front of every line it says in chat.
//
//sp:name PLUGIN_PREFIX
const PluginPrefix = "[BotManager]"

// BotIdentityName is the name a defender bot is created under, before it is
// given a real one. Nothing a person could type, so a person cannot be mistaken
// for one of ours.
//
//sp:name TFBOT_IDENTITY_NAME
const BotIdentityName = "TFBOT_SEX_HAVER"

/*
The prepared SDKCall handles, one per function the game has and SourceMod does
not offer.

They were file-static in sdkcalls.sp and the setter beside each is what fills
them, because a generated file cannot see a file-static in another one.
*/

//sp:name m_hPostInventoryApplication
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var postInventoryApplication engine.Call

//sp:name m_hSetMission
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var setMission engine.Call

//sp:name m_hLookupBone
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var lookupBone engine.Call

//sp:name m_hGetBonePosition
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var getBonePosition engine.Call

//sp:name m_hHasAmmo
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hasAmmo engine.Call

//sp:name m_hClip1
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var clip1 engine.Call

//sp:name m_hGetProjectileSpeed
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var getProjectileSpeed engine.Call

//sp:name m_hAimHeadTowards
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var aimHeadTowards engine.Call

//sp:name m_hGEconItemSchema
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var gEconItemSchema engine.Call

//sp:name m_hGetAttributeDefinitionByName
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var getAttributeDefinitionByName engine.Call

//sp:name m_hCanUpgradeWithAttrib
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var canUpgradeWithAttrib engine.Call

//sp:name m_hGetCostForUpgrade
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var getCostForUpgrade engine.Call

//sp:name m_hGetUpgradeTier
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var getUpgradeTier engine.Call

//sp:name m_hIsUpgradeTierEnabled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var isUpgradeTierEnabled engine.Call

//sp:name m_hShouldCollide
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var shouldCollide engine.Call

/*
The hooks and the flags they set, out of dhooks.sp.

Each hook is armed per object rather than once, so the handle is kept and
HookEntity or HookRaw is what points it at something.
*/

//sp:name m_hMyTouch
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hookMyTouch engine.Hook

//sp:name m_hIsBot
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hookIsBot engine.Hook

//sp:name m_hEventKilled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hookEventKilled engine.Hook

//sp:name m_hIsVisibleEntityNoticed
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hookIsVisibleEntityNoticed engine.Hook

//sp:name m_hIsIgnored
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var hookIsIgnored engine.Hook

/*
spyKilled is read outside dhooks: a spy dying is what tells the rest of the mod
to stop treating him as a threat.
*/
//
//sp:name g_bSpyKilled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var spyKilled bool

//sp:name m_bTouchCredits
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var touchCredits bool

//sp:name m_bPlayerKilled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var playerKilled bool

//sp:name m_bEngineerKilled
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var engineerKilled bool
