/*
Package dhooks is source/redbots3/dhooks.sp: standing in front of the functions
the game calls.

A detour covers every call of a function; a virtual hook covers one object. Both
are set up once from the gamedata file, and the callbacks below decide, per call,
whether to answer for the game or let it through.
*/
package dhooks

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// InitDHooks sets up every detour and hook, and says whether all of them worked.
//
//sp:name InitDHooks
func InitDHooks(hGamedata engine.GameData) bool {
	iFailCount := int32(0)

	/* No address for g_MannVsMachineUpgrades, so this detour fetches it
	instead. It will not support a late load. */
	if engine.UpgradesAddress() == engine.NoAddress() {
		if !RegisterDetour(hGamedata, "CMannVsMachineUpgradeManager::LoadUpgradesFile", engine.InvalidFunction(), engine.DHookLoadUpgradesFilePost()) {
			iFailCount++
		}
	}

	if !RegisterDetour(hGamedata, "CTFPlayer::ManageRegularWeapons", engine.DHookManageRegularWeaponsPre(), engine.DHookManageRegularWeaponsPost()) {
		iFailCount++
	}

	if !RegisterDetour(hGamedata, "CTFPlayer::ManageBuilderWeapons", engine.DHookManageBuilderWeaponsPre(), engine.InvalidFunction()) {
		iFailCount++
	}

	if !RegisterHook(hGamedata, engine.HookMyTouch(), "CItem::MyTouch") {
		iFailCount++
	}

	if !RegisterHook(hGamedata, engine.HookIsBot(), "CBasePlayer::IsBot") {
		iFailCount++
	}

	if !RegisterHook(hGamedata, engine.HookEventKilled(), "CBaseEntity::Event_Killed") {
		iFailCount++
	}

	if !RegisterHook(hGamedata, engine.HookIsVisibleEntityNoticed(), "IVision::IsVisibleEntityNoticed") {
		iFailCount++
	}

	if !RegisterHook(hGamedata, engine.HookIsIgnored(), "IVision::IsIgnored") {
		iFailCount++
	}

	if iFailCount > 0 {
		engine.LogError("InitDHooks: found %d problems with gamedata!", iFailCount)
		return false
	}

	return true
}

/*
	RegisterDetour builds one detour and turns on the sides it was given

The handle is closed either way: the detour it installs outlives it, and
holding on to it would leak one per function.
*/
//
//sp:name RegisterDetour
//sp:default pre INVALID_FUNCTION
//sp:default post INVALID_FUNCTION
func RegisterDetour(gd engine.GameData, fnName string, pre engine.DHookCallback, post engine.DHookCallback) bool {
	//nolint:staticcheck // S1021: the shipped function declares it and then assigns, and the emitted declaration follows
	var hDetour engine.Detour
	hDetour = engine.DetourFromConf(gd, fnName)

	if hDetour != engine.NoDetour() {
		if pre != engine.InvalidFunction() {
			hDetour.Enable(engine.HookPre(), pre)
		}

		if post != engine.InvalidFunction() {
			hDetour.Enable(engine.HookPost(), post)
		}
	} else {
		hDetour.Close()
		engine.LogError("Failed to detour \"%s\"!", fnName)

		return false
	}

	hDetour.Close()

	return true
}

// RegisterHook builds one virtual hook, which is armed per object later.
//
//sp:name RegisterHook
//sp:byref hook
//nolint:staticcheck // SA4009: hook is by reference in SourcePawn, which //sp:byref says, so the write is what the caller reads
func RegisterHook(gd engine.GameData, hook engine.Hook, fnName string) bool {
	hook = engine.HookFromConf(gd, fnName)

	if hook == engine.NoHook() {
		engine.LogError("Failed to setup DynamicHook for \"%s\"!", fnName)
		return false
	}

	return true
}

// DHooksOnEntityCreated arms the credit-pickup hook on every money pack.
//
//sp:name DHooks_OnEntityCreated
//sp:public
//nolint:revive // exported: the SourcePawn name is DHooks_OnEntityCreated and the Go follows it
func DHooksOnEntityCreated(entity int32, classname engine.Text) {
	if engine.StrContains(classname, "item_currencypack_", true) != -1 {
		engine.HookMyTouch().HookEntity(engine.HookPre(), entity, engine.DHookMyTouchPre())
		engine.HookMyTouch().HookEntity(engine.HookPost(), entity, engine.DHookMyTouchPost())
	}
}

// DHooksDefenderBot arms everything that is per bot, including the two on its
// vision interface, which is not an entity and has to be hooked by address.
//
//sp:name DHooks_DefenderBot
//nolint:revive // exported: the SourcePawn name is DHooks_DefenderBot and the Go follows it
func DHooksDefenderBot(client int32) {
	engine.HookIsBot().HookEntity(engine.HookPre(), client, engine.DHookIsBotPre())
	engine.HookEventKilled().HookEntity(engine.HookPre(), client, engine.DHookEventKilledPre())
	engine.HookEventKilled().HookEntity(engine.HookPost(), client, engine.DHookEventKilledPost())

	bot := engine.NextBotOf(client)
	vision := engine.AddressOfVision(bot.Vision())

	if vision != engine.NoAddress() {
		engine.HookIsVisibleEntityNoticed().HookRaw(engine.HookPre(), vision, engine.DHookIsVisibleEntityNoticedPre())
		engine.HookIsVisibleEntityNoticed().HookRaw(engine.HookPost(), vision, engine.DHookIsVisibleEntityNoticedPost())
		engine.HookIsIgnored().HookRaw(engine.HookPre(), vision, engine.DHookIsIgnoredPre())
	} else {
		engine.LogError("DHooks_DefenderBot: IVision is NULL! Bot vision will not be hooked.")
	}
}

// DHookCallbackLoadUpgradesFilePost catches the upgrade table's address on the
// one call that has it, for a server whose gamedata could not name it.
//
//sp:name DHookCallback_LoadUpgradesFile_Post
func DHookCallbackLoadUpgradesFilePost(pThis engine.Address) engine.Mres {
	if engine.UpgradesAddress() == engine.NoAddress() {
		engine.SetUpgradesAddress(pThis)
	}

	return engine.MresIgnored()
}

/*
	DHookCallbackManageRegularWeaponsPre stops the game handing a bot its stock weapons

A bot loses its items when it buys upgrades, so the function is blocked, but
only while it is actually at the upgrade station: blocking it otherwise leaves
every bot with broken items when it spawns.

	CUpgrades::PlayerPurchasingUpgrade
	  CTFPlayer::Regenerate
	    CTFPlayer::InitClass
	      CTFPlayer::GiveDefaultItems
	        CTFPlayer::ManageRegularWeapons
*/
//
//sp:name DHookCallback_ManageRegularWeapons_Pre
func DHookCallbackManageRegularWeaponsPre(pThis int32) engine.Mres {
	if engine.UseCustomLoadouts().Bool() {
		if engine.DefenderBotFlag(pThis) && engine.IsPlayerAlive(pThis) && engine.IsInUpgradeZone(pThis) {
			return engine.MresSupercede()
		}
	}

	return engine.MresIgnored()
}

// DHookCallbackManageRegularWeaponsPost hands the bot its own loadout instead,
// a tick later: the delay has to be in frames or the broken-items problem
// comes back.
//
//sp:name DHookCallback_ManageRegularWeapons_Post
func DHookCallbackManageRegularWeaponsPost(pThis int32) engine.Mres {
	if engine.UseCustomLoadouts().Bool() {
		if engine.DefenderBotFlag(pThis) && engine.IsPlayerAlive(pThis) && !engine.IsInUpgradeZone(pThis) {
			if engine.HasCustomLoadout(pThis) {
				engine.CreateTimerWith(0.1, engine.TimerGiveCustomLoadout, pThis, engine.TimerNoMapChange())
			} else {
				engine.PrepareCustomLoadout(pThis)
				engine.CreateTimerWith(0.1, engine.TimerGiveCustomLoadout, pThis, engine.TimerNoMapChange())
			}
		}
	}

	return engine.MresIgnored()
}

// DHookCallbackManageBuilderWeaponsPre keeps the sapper on a spy that is
// upgrading, for the same reason.
//
//sp:name DHookCallback_ManageBuilderWeapons_Pre
func DHookCallbackManageBuilderWeaponsPre(pThis int32) engine.Mres {
	if engine.UseCustomLoadouts().Bool() {
		if engine.DefenderBotFlag(pThis) && engine.PlayerClass(pThis) == engine.ClassSpy() && engine.IsPlayerAlive(pThis) {
			if engine.IsInUpgradeZone(pThis) {
				return engine.MresSupercede()
			}
		}
	}

	return engine.MresIgnored()
}

/*
	The virtual hooks, armed per object.

Only on our own bots and on the vision interfaces they own, so the game's own
robots go through the code the game wrote for them.
*/

// DHookCallbackMyTouchPre notices a defender bot picking credits up.
//
//sp:name DHookCallback_MyTouch_Pre
//nolint:revive // unused-parameter: a DHook is handed the object, the return and the arguments
func DHookCallbackMyTouchPre(pThis int32, hReturn engine.DHookReturn, hParams engine.DHookParam) engine.Mres {
	player := hParams.Get(1)

	if engine.DefenderBotFlag(player) {
		engine.SetTouchCredits(true)
	}

	return engine.MresIgnored()
}

// DHookCallbackMyTouchPost puts the flag back. The pair is the whole point: a
// hook that sets a flag and never clears it leaves every defender bot looking
// like a person to the money code for the rest of the map.
//
//sp:name DHookCallback_MyTouch_Post
//nolint:revive // unused-parameter: a DHook is handed the object, the return and the arguments
func DHookCallbackMyTouchPost(pThis int32, hReturn engine.DHookReturn, hParams engine.DHookParam) engine.Mres {
	player := hParams.Get(1)

	if engine.DefenderBotFlag(player) {
		engine.SetTouchCredits(false)
	}

	return engine.MresIgnored()
}

// DHookCallbackIsBotPre answers false for a defender bot in the middle of a
// credit touch or a death, so the game's own money and death code treats it as
// a player.
//
//sp:name DHookCallback_IsBot_Pre
//nolint:revive // unused-parameter: a DHook is handed the object and the return
func DHookCallbackIsBotPre(pThis int32, hReturn engine.DHookReturn) engine.Mres {
	if engine.IsClientInGame(pThis) && engine.DefenderBotFlag(pThis) {
		if engine.TouchCredits() || engine.PlayerKilled() {
			hReturn.SetBool(false)

			return engine.MresSupercede()
		}
	}

	return engine.MresIgnored()
}

/*
	DHookCallbackEventKilledPre moves a dying engineer off his class first

CTFBot::Event_Killed disbands an engineer bot's buildings when it dies in Mann
vs Machine mode, so it dies as a soldier and is put back afterwards. That messes
with class-specific achievement data, which does not matter here.
*/
//
//sp:name DHookCallback_EventKilled_Pre
//nolint:revive // unused-parameter: a DHook is handed the object and the arguments
func DHookCallbackEventKilledPre(pThis int32, hParams engine.DHookParam) engine.Mres {
	if engine.DefenderBotFlag(pThis) {
		engine.SetPlayerKilled(true)

		if engine.PlayerClass(pThis) == engine.ClassEngineer() {
			engine.SetPlayerClass(pThis, engine.ClassSoldier(), true, false)
			engine.SetEngineerKilled(true)
		} else if engine.PlayerClass(pThis) == engine.ClassSpy() {
			engine.SetSpyKilled(true)
		}
	}

	return engine.MresIgnored()
}

// DHookCallbackEventKilledPost puts the engineer back.
//
//sp:name DHookCallback_EventKilled_Post
//nolint:revive // unused-parameter: a DHook is handed the object and the arguments
func DHookCallbackEventKilledPost(pThis int32, hParams engine.DHookParam) engine.Mres {
	if engine.DefenderBotFlag(pThis) {
		engine.SetPlayerKilled(false)

		if engine.EngineerKilled() {
			engine.SetPlayerClass(pThis, engine.ClassEngineer(), true, false)
			engine.SetEngineerKilled(false)
		}

		if engine.SpyKilled() {
			engine.SetSpyKilled(false)
		}
	}

	return engine.MresIgnored()
}

/*
	DHookCallbackIsVisibleEntityNoticedPre tells the vision code this is not MvM

Which has a few consequences worth knowing: a disguised spy is never forgotten
unless he redisguises, sapping anything nearby calls him out, and changing
disguise in front of a bot calls him out.
*/
//
//sp:name DHookCallback_IsVisibleEntityNoticed_Pre
//nolint:revive // unused-parameter: a DHook is handed the object, the return and the arguments
func DHookCallbackIsVisibleEntityNoticedPre(pThis engine.Address, hReturn engine.DHookReturn, hParams engine.DHookParam) engine.Mres {
	engine.SetGameRulesProp("m_bPlayingMannVsMachine", 0)

	return engine.MresIgnored()
}

// DHookCallbackIsVisibleEntityNoticedPost puts it back, because at the end of
// the day this is still Mann vs Machine.
//
//sp:name DHookCallback_IsVisibleEntityNoticed_Post
//nolint:revive // unused-parameter: a DHook is handed the object, the return and the arguments
func DHookCallbackIsVisibleEntityNoticedPost(pThis engine.Address, hReturn engine.DHookReturn, hParams engine.DHookParam) engine.Mres {
	engine.SetGameRulesProp("m_bPlayingMannVsMachine", 1)

	return engine.MresIgnored()
}

/*
	DHookCallbackIsIgnoredPre decides what a bot does not need to look at

Three things: a sentry buster, which nothing can be done about; an invulnerable
enemy, unless the weapon in hand has knockback worth using on one; and a sapped
building, which the game only ignores outside Mann vs Machine.
*/
//
//sp:name DHookCallback_IsIgnored_Pre
//nolint:revive // unused-parameter: a DHook is handed the object, the return and the arguments
func DHookCallbackIsIgnoredPre(pThis engine.Address, hReturn engine.DHookReturn, hParams engine.DHookParam) engine.Mres {
	subject := hParams.Get(1)
	myself := engine.VisionBot(pThis).Bot().Entity()
	myTeam := engine.GetClientTeam(myself)

	if engine.IsPlayer(subject) && engine.GetClientTeam(subject) != myTeam {
		if engine.IsSentryBusterRobot(subject) {
			// A sentry buster is not something to shoot at.
			hReturn.SetBool(true)

			return engine.MresSupercede()
		}

		if engine.IsInvulnerable(subject) {
			if engine.IsPlayerInCondition(subject, engine.ConditionImmuneToPushback()) {
				// Always ignored, since nothing can be done about them.
				hReturn.SetBool(true)

				return engine.MresSupercede()
			}

			myWeapon := engine.ActiveWeapon(myself)

			if myWeapon != -1 {
				switch engine.WeaponID(myWeapon) {
				case engine.WeaponRocketLauncher(), engine.WeaponGrenadeLauncher(), engine.WeaponPipebombLauncher(), engine.WeaponDirecthit(), engine.WeaponParticleCannon(), engine.WeaponFlameBall():
					// Not ignored with these, which have knockback.
				case engine.WeaponFlamethrower():
					if !engine.CanWeaponAirblast(myWeapon) {
						// Nothing can be done about that.
						hReturn.SetBool(true)
						return engine.MresSupercede()
					}
				default:
					// An ubered enemy is ignored for threat selection,
					// because shooting one only wastes ammo.
					hReturn.SetBool(true)

					return engine.MresSupercede()
				}
			}
		}
	} else if engine.IsBaseObject(subject) && engine.EntityTeamNumber(subject) != myTeam {
		if engine.HasSapper(subject) {
			// The game ignores these outside Mann vs Machine.
			hReturn.SetBool(true)

			return engine.MresSupercede()
		}
	}

	return engine.MresIgnored()
}
