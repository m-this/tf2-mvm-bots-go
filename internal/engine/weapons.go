package engine

/*
The weapon ids the tank scoring reads.

One //sp:global each, and the numbers come from the generated tf2_stocks binding
rather than from anybody's memory, so the Go side of a differential answers what
the game answers.
*/

// WeaponBat is TF_WEAPON_BAT.
//
//sp:global TF_WEAPON_BAT
func WeaponBat() Weapon { return 1 }

// WeaponBatFish is TF_WEAPON_BAT_FISH.
//
//sp:global TF_WEAPON_BAT_FISH
func WeaponBatFish() Weapon { return 72 }

// WeaponBatGiftwrap is TF_WEAPON_BAT_GIFTWRAP.
//
//sp:global TF_WEAPON_BAT_GIFTWRAP
func WeaponBatGiftwrap() Weapon { return 82 }

// WeaponBatWood is TF_WEAPON_BAT_WOOD.
//
//sp:global TF_WEAPON_BAT_WOOD
func WeaponBatWood() Weapon { return 2 }

// WeaponBonesaw is TF_WEAPON_BONESAW.
//
//sp:global TF_WEAPON_BONESAW
func WeaponBonesaw() Weapon { return 11 }

// WeaponBottle is TF_WEAPON_BOTTLE.
//
//sp:global TF_WEAPON_BOTTLE
func WeaponBottle() Weapon { return 3 }

// WeaponBuffItem is TF_WEAPON_BUFF_ITEM.
//
//sp:global TF_WEAPON_BUFF_ITEM
func WeaponBuffItem() Weapon { return 62 }

// WeaponCannon is TF_WEAPON_CANNON.
//
//sp:global TF_WEAPON_CANNON
func WeaponCannon() Weapon { return 91 }

// WeaponChargedSmg is TF_WEAPON_CHARGED_SMG.
//
//sp:global TF_WEAPON_CHARGED_SMG
func WeaponChargedSmg() Weapon { return 103 }

// WeaponCleaver is TF_WEAPON_CLEAVER.
//
//sp:global TF_WEAPON_CLEAVER
func WeaponCleaver() Weapon { return 86 }

// WeaponClub is TF_WEAPON_CLUB.
//
//sp:global TF_WEAPON_CLUB
func WeaponClub() Weapon { return 5 }

// WeaponCompoundBow is TF_WEAPON_COMPOUND_BOW.
//
//sp:global TF_WEAPON_COMPOUND_BOW
func WeaponCompoundBow() Weapon { return 61 }

// WeaponCrossbow is TF_WEAPON_CROSSBOW.
//
//sp:global TF_WEAPON_CROSSBOW
func WeaponCrossbow() Weapon { return 73 }

// WeaponDirecthit is TF_WEAPON_DIRECTHIT.
//
//sp:global TF_WEAPON_DIRECTHIT
func WeaponDirecthit() Weapon { return 65 }

// WeaponDispenser is TF_WEAPON_DISPENSER.
//
//sp:global TF_WEAPON_DISPENSER
func WeaponDispenser() Weapon { return 56 }

// WeaponDispenserGun is TF_WEAPON_DISPENSER_GUN.
//
//sp:global TF_WEAPON_DISPENSER_GUN
func WeaponDispenserGun() Weapon { return 68 }

// WeaponDrgPomson is TF_WEAPON_DRG_POMSON.
//
//sp:global TF_WEAPON_DRG_POMSON
func WeaponDrgPomson() Weapon { return 81 }

// WeaponFireaxe is TF_WEAPON_FIREAXE.
//
//sp:global TF_WEAPON_FIREAXE
func WeaponFireaxe() Weapon { return 4 }

// WeaponFists is TF_WEAPON_FISTS.
//
//sp:global TF_WEAPON_FISTS
func WeaponFists() Weapon { return 8 }

// WeaponFlamethrowerRocket is TF_WEAPON_FLAMETHROWER_ROCKET.
//
//sp:global TF_WEAPON_FLAMETHROWER_ROCKET
func WeaponFlamethrowerRocket() Weapon { return 52 }

// WeaponFlaregun is TF_WEAPON_FLAREGUN.
//
//sp:global TF_WEAPON_FLAREGUN
func WeaponFlaregun() Weapon { return 58 }

// WeaponGrenadelauncher is TF_WEAPON_GRENADELAUNCHER.
//
//sp:global TF_WEAPON_GRENADELAUNCHER
func WeaponGrenadelauncher() Weapon { return 23 }

// WeaponHandgunScoutPrimary is TF_WEAPON_HANDGUN_SCOUT_PRIMARY.
//
//sp:global TF_WEAPON_HANDGUN_SCOUT_PRIMARY
func WeaponHandgunScoutPrimary() Weapon { return 71 }

// WeaponHandgunScoutSec is TF_WEAPON_HANDGUN_SCOUT_SEC.
//
//sp:global TF_WEAPON_HANDGUN_SCOUT_SEC
func WeaponHandgunScoutSec() Weapon { return 75 }

// WeaponHarvesterSaw is TF_WEAPON_HARVESTER_SAW.
//
//sp:global TF_WEAPON_HARVESTER_SAW
func WeaponHarvesterSaw() Weapon { return 96 }

// WeaponInvis is TF_WEAPON_INVIS.
//
//sp:global TF_WEAPON_INVIS
func WeaponInvis() Weapon { return 57 }

// WeaponJar is TF_WEAPON_JAR.
//
//sp:global TF_WEAPON_JAR
func WeaponJar() Weapon { return 60 }

// WeaponJarMilk is TF_WEAPON_JAR_MILK.
//
//sp:global TF_WEAPON_JAR_MILK
func WeaponJarMilk() Weapon { return 70 }

// WeaponKnife is TF_WEAPON_KNIFE.
//
//sp:global TF_WEAPON_KNIFE
func WeaponKnife() Weapon { return 7 }

// WeaponLaserPointer is TF_WEAPON_LASER_POINTER.
//
//sp:global TF_WEAPON_LASER_POINTER
func WeaponLaserPointer() Weapon { return 67 }

// WeaponLunchbox is TF_WEAPON_LUNCHBOX.
//
//sp:global TF_WEAPON_LUNCHBOX
func WeaponLunchbox() Weapon { return 59 }

// WeaponMechanicalArm is TF_WEAPON_MECHANICAL_ARM.
//
//sp:global TF_WEAPON_MECHANICAL_ARM
func WeaponMechanicalArm() Weapon { return 80 }

// WeaponMinigun is TF_WEAPON_MINIGUN.
//
//sp:global TF_WEAPON_MINIGUN
func WeaponMinigun() Weapon { return 18 }

// WeaponNailgun is TF_WEAPON_NAILGUN.
//
//sp:global TF_WEAPON_NAILGUN
func WeaponNailgun() Weapon { return 44 }

// WeaponParachute is TF_WEAPON_PARACHUTE.
//
//sp:global TF_WEAPON_PARACHUTE
func WeaponParachute() Weapon { return 100 }

// WeaponParticleCannon is TF_WEAPON_PARTICLE_CANNON.
//
//sp:global TF_WEAPON_PARTICLE_CANNON
func WeaponParticleCannon() Weapon { return 79 }

// WeaponPda is TF_WEAPON_PDA.
//
//sp:global TF_WEAPON_PDA
func WeaponPda() Weapon { return 45 }

// WeaponPdaEngineerBuild is TF_WEAPON_PDA_ENGINEER_BUILD.
//
//sp:global TF_WEAPON_PDA_ENGINEER_BUILD
func WeaponPdaEngineerBuild() Weapon { return 46 }

// WeaponPdaEngineerDestroy is TF_WEAPON_PDA_ENGINEER_DESTROY.
//
//sp:global TF_WEAPON_PDA_ENGINEER_DESTROY
func WeaponPdaEngineerDestroy() Weapon { return 47 }

// WeaponPdaSpy is TF_WEAPON_PDA_SPY.
//
//sp:global TF_WEAPON_PDA_SPY
func WeaponPdaSpy() Weapon { return 48 }

// WeaponPdaSpyBuild is TF_WEAPON_PDA_SPY_BUILD.
//
//sp:global TF_WEAPON_PDA_SPY_BUILD
func WeaponPdaSpyBuild() Weapon { return 94 }

// WeaponPepBrawlerBlaster is TF_WEAPON_PEP_BRAWLER_BLASTER.
//
//sp:global TF_WEAPON_PEP_BRAWLER_BLASTER
func WeaponPepBrawlerBlaster() Weapon { return 85 }

// WeaponPistol is TF_WEAPON_PISTOL.
//
//sp:global TF_WEAPON_PISTOL
func WeaponPistol() Weapon { return 41 }

// WeaponPistolScout is TF_WEAPON_PISTOL_SCOUT.
//
//sp:global TF_WEAPON_PISTOL_SCOUT
func WeaponPistolScout() Weapon { return 42 }

// WeaponRaygun is TF_WEAPON_RAYGUN.
//
//sp:global TF_WEAPON_RAYGUN
func WeaponRaygun() Weapon { return 78 }

// WeaponRaygunRevenge is TF_WEAPON_RAYGUN_REVENGE.
//
//sp:global TF_WEAPON_RAYGUN_REVENGE
func WeaponRaygunRevenge() Weapon { return 84 }

// WeaponRevolver is TF_WEAPON_REVOLVER.
//
//sp:global TF_WEAPON_REVOLVER
func WeaponRevolver() Weapon { return 43 }

// WeaponRocketlauncher is TF_WEAPON_ROCKETLAUNCHER.
//
//sp:global TF_WEAPON_ROCKETLAUNCHER
func WeaponRocketlauncher() Weapon { return 22 }

// WeaponScattergun is TF_WEAPON_SCATTERGUN.
//
//sp:global TF_WEAPON_SCATTERGUN
func WeaponScattergun() Weapon { return 16 }

// WeaponSentryRevenge is TF_WEAPON_SENTRY_REVENGE.
//
//sp:global TF_WEAPON_SENTRY_REVENGE
func WeaponSentryRevenge() Weapon { return 69 }

// WeaponShotgunBuildingRescue is TF_WEAPON_SHOTGUN_BUILDING_RESCUE.
//
//sp:global TF_WEAPON_SHOTGUN_BUILDING_RESCUE
func WeaponShotgunBuildingRescue() Weapon { return 90 }

// WeaponShotgunHwg is TF_WEAPON_SHOTGUN_HWG.
//
//sp:global TF_WEAPON_SHOTGUN_HWG
func WeaponShotgunHwg() Weapon { return 14 }

// WeaponShotgunPrimary is TF_WEAPON_SHOTGUN_PRIMARY.
//
//sp:global TF_WEAPON_SHOTGUN_PRIMARY
func WeaponShotgunPrimary() Weapon { return 12 }

// WeaponShotgunPyro is TF_WEAPON_SHOTGUN_PYRO.
//
//sp:global TF_WEAPON_SHOTGUN_PYRO
func WeaponShotgunPyro() Weapon { return 15 }

// WeaponShotgunSoldier is TF_WEAPON_SHOTGUN_SOLDIER.
//
//sp:global TF_WEAPON_SHOTGUN_SOLDIER
func WeaponShotgunSoldier() Weapon { return 13 }

// WeaponShovel is TF_WEAPON_SHOVEL.
//
//sp:global TF_WEAPON_SHOVEL
func WeaponShovel() Weapon { return 9 }

// WeaponSmg is TF_WEAPON_SMG.
//
//sp:global TF_WEAPON_SMG
func WeaponSmg() Weapon { return 19 }

// WeaponSniperrifle is TF_WEAPON_SNIPERRIFLE.
//
//sp:global TF_WEAPON_SNIPERRIFLE
func WeaponSniperrifle() Weapon { return 17 }

// WeaponSniperrifleClassic is TF_WEAPON_SNIPERRIFLE_CLASSIC.
//
//sp:global TF_WEAPON_SNIPERRIFLE_CLASSIC
func WeaponSniperrifleClassic() Weapon { return 99 }

// WeaponSniperrifleDecap is TF_WEAPON_SNIPERRIFLE_DECAP.
//
//sp:global TF_WEAPON_SNIPERRIFLE_DECAP
func WeaponSniperrifleDecap() Weapon { return 77 }

// WeaponSodaPopper is TF_WEAPON_SODA_POPPER.
//
//sp:global TF_WEAPON_SODA_POPPER
func WeaponSodaPopper() Weapon { return 76 }

// WeaponStickbomb is TF_WEAPON_STICKBOMB.
//
//sp:global TF_WEAPON_STICKBOMB
func WeaponStickbomb() Weapon { return 74 }

// WeaponStickyBallLauncher is TF_WEAPON_STICKY_BALL_LAUNCHER.
//
//sp:global TF_WEAPON_STICKY_BALL_LAUNCHER
func WeaponStickyBallLauncher() Weapon { return 88 }

// WeaponSword is TF_WEAPON_SWORD.
//
//sp:global TF_WEAPON_SWORD
func WeaponSword() Weapon { return 64 }

// WeaponSyringegunMedic is TF_WEAPON_SYRINGEGUN_MEDIC.
//
//sp:global TF_WEAPON_SYRINGEGUN_MEDIC
func WeaponSyringegunMedic() Weapon { return 20 }

// WeaponWrench is TF_WEAPON_WRENCH.
//
//sp:global TF_WEAPON_WRENCH
func WeaponWrench() Weapon { return 10 }
