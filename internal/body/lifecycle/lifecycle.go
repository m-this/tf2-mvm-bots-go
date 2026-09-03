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

/*
	TimerRefillDefenderTeam puts a bot back in the seat a player just left

Runs a tick after the disconnect, because the leaving player is still in the
game at the point the forward fires and would otherwise still be counted.
Nobody is left to play with if the last player leaves, so an empty defending
team is left empty rather than filled with six bots holding a hatch for no one.
*/
//
//sp:name Timer_RefillDefenderTeam
//nolint:revive // unused-parameter: the handle is the timer's own, and nothing here needs it
func TimerRefillDefenderTeam(timer engine.Timer) engine.Outcome {
	if !engine.BotsEnabled() {
		return engine.PluginStop()
	}

	if engine.RealPlayerCount() < 1 {
		return engine.PluginStop()
	}

	missing := engine.DefenderTeamSize().Int() - engine.HumanAndDefenderBotCount(engine.TeamRed())

	if missing > 0 {
		engine.AddBotsBasedOnLineupModeNow(missing, true)
	}

	return engine.PluginStop()
}

// OnMapStart puts the mod back to nothing and reads the new map's files.
//
//sp:name OnMapStart
//sp:public
func OnMapStart() {
	engine.SetBotsEnabled(false)
	engine.SetAddingBotTime(0.0)
	engine.SetNextReadyTime(0.0)
	engine.SetBotClassesLocked(false)
	engine.SetAllowBotRedo(false)
	engine.ReseatOnMapStart()

	engine.ResetMapHintNests()

	engine.ConfigLoadMap()
	engine.ConfigLoadBotNames()
	engine.ConfigLoadServerLoadout()

	engine.CreateBotPreferenceMenu()
}

/*
	OnClientDisconnect forgets everything that was true of that slot

The refill runs a tick later, because the leaving player is still in the game
here and would otherwise still be counted.
*/
//
//sp:name OnClientDisconnect
//sp:public
func OnClientDisconnect(client int32) {
	if client == engine.PlayerForcedPref() {
		engine.SetPlayerForcedPref(-1)
	}

	if !engine.IsFakeClient(client) {
		engine.CreateTimerFlags(0.1, TimerRefillDefenderTeam, engine.TimerNoMapChange())
	}

	engine.SetDefenderBotFlag(client, false)
	engine.ResetSpawnExitWatch(client)

	engine.SetChoosingBotClasses(client, false)

	engine.ResetLoadouts(client)
	engine.ForgetBotSeat(client)
	engine.ForgetBotCosmetics(client)
}

/*
	OnClientPutInServer forgets everything the previous occupant of the slot left

The name is set at tf_bot_add, so one of ours is known here, before it picks its
loadout.
*/
//
//sp:name OnClientPutInServer
//sp:public
func OnClientPutInServer(client int32) {
	if !engine.IsFakeClient(client) {
		engine.MakeRoomForHumanPlayer(client)
	}

	if engine.IsDefenderBot(client) {
		engine.TakeBotSeat(client)
	}

	engine.SetHasUpgraded(client, false)
	engine.SetShoppedThisBreak(client, false)
	// A slot is reused, and a call left on the clock by whoever had it is not
	// this player's call.
	engine.ForgetMedicCall(client)
	engine.ExtraButtons(client).Reset()
	engine.SetDeadRethinkTime(client, 0.0)
	engine.SetBuybackNumber(client, 0)
	engine.SetBuyUpgradesNumber(client, 0)

	engine.SetNextRollTime(client, 0.0)

	engine.SetEnableBotsCooldown(client, 0.0)

	engine.ResetCommandThrottle(client)
	engine.SetLastReadyInputTime(client, 0.0)

	engine.SetHasBoughtUpgrades(client, false)

	engine.ResetNextBot(client)
	engine.ResetSpawnExitWatch(client)
}

// OnPlayerRunCmd is the whole of it.
//
//sp:name OnPlayerRunCmd
//sp:public
//sp:byref buttons
//sp:byref impulse
//sp:byref weapon
//sp:byref subtype
//sp:byref cmdnum
//sp:byref tickcount
//sp:byref seed
//sp:mutates vel
//sp:mutates angles
//nolint:revive // unused-parameter: a run-cmd hook is handed the whole command and reads three of it
func OnPlayerRunCmd(client int32, buttons int32, impulse int32, vel [3]float32, angles [3]float32, weapon int32, subtype int32, cmdnum int32, tickcount int32, seed int32, mouse [2]int32) engine.Outcome {
	if !engine.DefenderBotFlag(client) {
		return engine.PluginContinue()
	}

	if engine.IsPlayerAlive(client) {
		engine.WatchDefenderSpawnExit(client)

		if engine.ExtraButtons(client).Press() != 0 {
			if engine.ExtraButtons(client).Press()&engine.ButtonBack() != 0 {
				vel[0] -= engine.PlayerSideSpeed()
			}

			if engine.ExtraButtons(client).Press()&engine.InForward() != 0 {
				vel[0] += engine.PlayerSideSpeed()
			}

			if engine.ExtraButtons(client).Press()&engine.InMoveLeft() != 0 {
				vel[1] -= engine.PlayerSideSpeed()
			}

			if engine.ExtraButtons(client).Press()&engine.InMoveRight() != 0 {
				vel[1] += engine.PlayerSideSpeed()
			}

			if engine.ExtraButtons(client).Press()&engine.ButtonTurnLeft() != 0 {
				angles[1] -= engine.ExtraButtons(client).KeySpeed()
			}

			if engine.ExtraButtons(client).Press()&engine.ButtonTurnRight() != 0 {
				angles[1] += engine.ExtraButtons(client).KeySpeed()
			}

			buttons |= engine.ExtraButtons(client).Press()

			// Held down for a time somebody asked for, so it is not cleared
			// until that time is up.
			if engine.ExtraButtons(client).PressTime() <= engine.GameTime() {
				engine.ExtraButtons(client).SetPress(0)
			}
		}

		if engine.ExtraButtons(client).Release() != 0 {
			buttons &= ^engine.ExtraButtons(client).Release()

			if engine.ExtraButtons(client).ReleaseTime() <= engine.GameTime() {
				engine.ExtraButtons(client).SetRelease(0)
			}
		}

		engine.PluginBotSimulateFrame(client)

		if engine.RoundState() != engine.RoundStateBetweenRounds() {
			myWeapon := engine.ActiveWeapon(client)
			weaponID := engine.ChooseWeapon(myWeapon != -1, engine.WeaponID(myWeapon), -1)

			if buttons&engine.InAttack() != 0 {
				switch weaponID {
				case engine.WeaponMinigun():
					// Do not keep spinning the minigun once it is out of ammo.
					if !engine.HasAmmo(myWeapon) {
						buttons &= ^engine.InAttack()
					}
				case engine.WeaponSniperrifleClassic():
					// The classic fires on release, so let go at a full charge.
					if engine.EntPropFloat(myWeapon, engine.PropSend(), "m_flChargedDamage") >= 150.0 {
						buttons &= ^engine.InAttack()
					}
				case engine.WeaponBuffItem():
					// Once the horn is blowing, stop pressing fire.
					if engine.IsPlayingHorn(myWeapon) {
						buttons &= ^engine.InAttack()
					}
				case engine.WeaponRevolver():
					if engine.CanRevolverHeadshot(myWeapon) {
						// Do not fire while the shot would be inaccurate.
						if !(engine.GameTime()-engine.LastAccuracyCheck(myWeapon) > 1.0) {
							buttons &= ^engine.InAttack()
						}
					}
				}
			}

			myBot := engine.NextBotOf(client)
			myVision := myBot.Vision()

			engine.MonitorKnownEntities(client, myVision)

			threat := myVision.PrimaryKnownThreat(false)

			engine.UseWeaponAbilities(client, myWeapon, myBot, threat)
			engine.UsePowerupBottle(client, myWeapon, myBot, threat)

			if (weaponID == engine.WeaponFlamethrower() || weaponID == engine.WeaponFlameBall()) && engine.CanWeaponAirblast(myWeapon) {
				engine.UtilizeCompressionBlast(client, myBot, threat, 1)
			}

			if engine.WeaponIDIsSniperRifle(weaponID) {
				if engine.IsPlayerInCondition(client, engine.ConditionZoomed()) {
					if engine.AimSkill().Int() >= 1 {
						if threat != engine.NoKnownEntity() && engine.IsLineOfFireClearEntity(client, engine.EyePosition(client), threat.Entity()) {
							// Help aim towards the desired target point.
							aimPos := myBot.Intention().SelectTargetPointOf(threat.Entity())
							engine.SnapViewToPosition(client, aimPos)

							if engine.NextSnipeFireTime(client) <= engine.GameTime() {
								engine.PressFireButton(client)
							}
						} else {
							// A reaction time before the next shot, so a threat
							// that reappears is not hit instantly.
							engine.SetNextSnipeFireTime(client, engine.GameTime()+engine.SniperReactionTime())
						}
					} else {
						if threat != engine.NoKnownEntity() && threat.VisibleInFOVNow() && myBot.Body().IsHeadAimingOnTarget() {
							if engine.NextSnipeFireTime(client) <= engine.GameTime() {
								engine.PressFireButton(client)
							}
						} else {
							engine.SetNextSnipeFireTime(client, engine.GameTime()+engine.SniperReactionTime())
						}
					}
				} else {
					// A reaction time while not scoped in.
					engine.SetNextSnipeFireTime(client, engine.GameTime()+engine.SniperReactionTime())
				}
			} else {
				if threat != engine.NoKnownEntity() {
					// Some scenarios where the aim must not be altered.
					if engine.IsCombatWeapon(client, myWeapon) && weaponID != engine.WeaponKnife() && engine.PlayerClass(client) != engine.ClassEngineer() && weaponID != engine.WeaponBonesaw() {
						iThreat := threat.Entity()

						//nolint:gocritic // ifElseChain: the shipped aim ladder is this chain, and a switch cannot be compared against it
						if engine.AimSkill().Int() >= 2 {
							/* This used to be handled in CTFBotMainAction_SelectTargetPoint, but
							that function does not always get called when the bot is up close to a
							tank: the bot looks up, then starts looking towards the centre again and
							stops firing, then looks up and fires again, over and over until it gets
							away from the tank */
							if weaponID == engine.WeaponFlamethrower() && engine.IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, engine.FlamethrowerReachRange()) {
								aimPos := engine.FlameThrowerAimForTank(iThreat)
								engine.SnapViewToPosition(client, aimPos)
								//nolint:ineffassign,wastedassign // the caller sees this: buttons is a by-reference parameter in SourcePawn and //sp:byref says so
								buttons |= engine.InAttack()
							} else if !threat.VisibleInFOVNow() && engine.IsLineOfFireClearEntity(client, engine.EyePosition(client), iThreat) {
								// Not facing the threat, so turn towards it quickly.
								aimPos := myBot.Intention().SelectTargetPointOf(iThreat)
								engine.SnapViewToPosition(client, aimPos)
							}
						} else if engine.AimSkill().Int() == 1 {
							if weaponID == engine.WeaponFlamethrower() && engine.IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, engine.FlamethrowerReachRange()) {
								aimPos := engine.FlameThrowerAimForTank(iThreat)
								engine.SnapViewToPosition(client, aimPos)
								//nolint:ineffassign,wastedassign // the caller sees this: buttons is a by-reference parameter in SourcePawn and //sp:byref says so
								buttons |= engine.InAttack()
							} else if !threat.VisibleRecently() && engine.IsLineOfFireClearEntity(client, engine.EyePosition(client), iThreat) {
								aimPos := myBot.Intention().SelectTargetPointOf(iThreat)
								engine.SnapViewToPosition(client, aimPos)
							}
						} else {
							if weaponID == engine.WeaponFlamethrower() && engine.IsBaseBoss(iThreat) && myBot.IsRangeLessThan(iThreat, engine.FlamethrowerReachRange()) {
								aimPos := engine.FlameThrowerAimForTank(iThreat)
								engine.SnapViewToPosition(client, aimPos)
								//nolint:ineffassign,wastedassign // the caller sees this: buttons is a by-reference parameter in SourcePawn and //sp:byref says so
								buttons |= engine.InAttack()
							}
						}
					}
				}
			}

			if engine.RtdVariance().Float() >= engine.CommandMaxRate() {
				if threat != engine.NoKnownEntity() && threat.VisibleInFOVNow() && engine.NextRollTime(client) <= engine.GameTime() {
					engine.SetNextRollTime(client, engine.GameTime()+engine.RandomFloat(engine.CommandMaxRate(), engine.RtdVariance().Float()))
					engine.FakeClientCommand(client, "sm_rtd")
				}
			}
		}

		if engine.IsInUpgradeZone(client) && engine.LookupEntityActionByName(client, "DefenderUpgrade") != engine.InvalidAction() {
			// Because of CTFBot::AvoidPlayers, do not let the bot move away from
			// other players while it is upgrading.
			//nolint:ineffassign,wastedassign // the caller sees this: vel is a by-reference parameter in SourcePawn and //sp:mutates says so
			vel = engine.NullVector()
		}
	} else { //nolint:gocritic // elseif: the shipped function nests these, and flattening them would not compare
		if engine.DeadRethinkTime(client) <= engine.GameTime() {
			// Think once a second while dead.
			engine.SetDeadRethinkTime(client, engine.GameTime()+1.0)

			iObsMode := engine.ObserverMode(client)

			if iObsMode == engine.ObserverModeFreezecam() || iObsMode == engine.ObserverModeDeathcam() {
				// Buying back is not possible right now, so do not think about it.
				engine.SetBuybackNumber(client, 0)
			} else {
				// Randomly think about buying back.
				engine.SetBuybackNumber(client, engine.RandomInt(1, 100))
			}

			if engine.ShouldBuybackIntoGame(client) {
				engine.PlayerBuyback(client)
			}

			if engine.ManagerDebug().Bool() {
				engine.PrintToChatAll("[OnPlayerRunCmd] g_iBuybackNumber[%d] = %d", client, engine.BuybackNumber(client))
			}
		}
	}

	return engine.PluginContinue()
}
