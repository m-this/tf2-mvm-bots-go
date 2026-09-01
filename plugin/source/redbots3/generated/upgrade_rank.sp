/* Generated from internal/upgrade/table.go. Do not edit.

The table it comes from is the only place these scores are written, and its
upstream test is what says they are the shipped ones. */

/* An upgrade no table below recognised

The mod's own answer for every upgrade, kept for the ones the tables do not name. It has to stay
random: a constant would tie every unknown upgrade, and a tie is broken by whichever the game
listed first, so a bot would buy the same wrong thing every wave of every mission. */
stock int UnrankedUpgradePriority()
{
	return GetRandomInt(50, 100);
}

/* What a resistance is worth, given whether the wave will actually deal that damage

Below the damage upgrades on purpose. A team that kills the wave faster takes less of everything,
and the guides all put resistances after the weapon is bought. Above the rest of the tail,
because the alternative is what this mod did before, which was never buying one. */
stock int ResistancePriority(bool wanted)
{
	if (!Feature(FEATURE_WAVE_RESISTANCES))
		return 35;
	
	return wanted ? 210 : 25;
}

/* The upgrade that is the reason to carry this exact weapon, by item definition index

Zero when the weapon in that slot has no opinion, which is most of them: this only names the few
where the loadout, not the class, decides what to buy first. */
stock int UpgradeRankLoadout(int itemDef, int attribute)
{
	switch (itemDef)
	{
		case 35:
		{
			switch (attribute)
			{
				case ATTRIBUTE_UBERCHARGE_RATE_BONUS:
				{
					return 330;
				}
			}
		}
		case 141:
		{
			switch (attribute)
			{
				case ATTRIBUTE_CLIP_SIZE_UPGRADE_ATOMIC:
				{
					return 260;
				}
				case ATTRIBUTE_CLIP_SIZE_BONUS_UPGRADE:
				{
					return 260;
				}
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 250;
				}
			}
		}
		case 312:
		{
			switch (attribute)
			{
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 320;
				}
			}
		}
		case 411:
		{
			switch (attribute)
			{
				case ATTRIBUTE_HEALING_MASTERY:
				{
					return 330;
				}
				case ATTRIBUTE_UBERCHARGE_RATE_BONUS:
				{
					return 300;
				}
			}
		}
		case 424:
		{
			switch (attribute)
			{
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 300;
				}
			}
		}
		case 526:
		{
			switch (attribute)
			{
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 300;
				}
			}
		}
		case 527:
		{
			switch (attribute)
			{
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 300;
				}
				case ATTRIBUTE_FIRE_RATE_BONUS:
				{
					return 250;
				}
			}
		}
		case 528:
		{
			switch (attribute)
			{
				case ATTRIBUTE_METAL_REGEN:
				{
					return 300;
				}
			}
		}
		case 730:
		{
			switch (attribute)
			{
				case ATTRIBUTE_CLIP_SIZE_UPGRADE_ATOMIC:
				{
					return 280;
				}
				case ATTRIBUTE_CLIP_SIZE_BONUS_UPGRADE:
				{
					return 280;
				}
				case ATTRIBUTE_FIRE_RATE_BONUS:
				{
					return 20;
				}
			}
		}
		case 752:
		{
			switch (attribute)
			{
				case ATTRIBUTE_SRIFLE_CHARGE_RATE_INCREASED:
				{
					return 300;
				}
			}
		}
		case 996:
		{
			switch (attribute)
			{
				case ATTRIBUTE_PROJECTILE_SPEED_INCREASED:
				{
					return 300;
				}
			}
		}
		case 997:
		{
			switch (attribute)
			{
				case ATTRIBUTE_METAL_REGEN:
				{
					return 300;
				}
				case ATTRIBUTE_MAXAMMO_METAL_INCREASED:
				{
					return 290;
				}
			}
		}
	}

	return 0;
}

/* The metal upgrades an engineer whose gun spends metal wants, which do not hang off the gun

Asked before the slot is looked at at all, because the attribute is on the player rather than on
the weapon that spends it. */
stock int UpgradeRankEngineerMetal(int attribute)
{
	switch (attribute)
	{
		case ATTRIBUTE_MAXAMMO_METAL_INCREASED:
		{
			return 310;
		}
		case ATTRIBUTE_METAL_REGEN:
		{
			return 305;
		}
	}

	return 0;
}

/* What this class contributes with, which is not always the weapon in its hands */
stock int UpgradeRankClass(TFClassType pclass, int slot, int attribute)
{
	switch (pclass)
	{
		case TFClass_Scout:
		{
			switch (attribute)
			{
				case ATTRIBUTE_APPLIES_SNARE_EFFECT:
				{
					return 250;
				}
				case ATTRIBUTE_MAD_MILK_SYRINGES:
				{
					return 200;
				}
				case ATTRIBUTE_MOVE_SPEED_BONUS:
				{
					return 190;
				}
			}
		}
		case TFClass_Sniper:
		{
			switch (attribute)
			{
				case ATTRIBUTE_EXPLOSIVE_SNIPER_SHOT:
				{
					return 330;
				}
				case ATTRIBUTE_FASTER_RELOAD_RATE:
				{
					return 300;
				}
				case ATTRIBUTE_SRIFLE_CHARGE_RATE_INCREASED:
				{
					return 60;
				}
			}
		}
		case TFClass_Soldier:
		{
			switch (attribute)
			{
				case ATTRIBUTE_FASTER_RELOAD_RATE:
				{
					return 310;
				}
				case ATTRIBUTE_ROCKET_SPECIALIST:
				{
					return 290;
				}
				case ATTRIBUTE_HEAL_ON_KILL:
				{
					return 250;
				}
			}
		}
		case TFClass_DemoMan:
		{
			switch (attribute)
			{
				case ATTRIBUTE_FASTER_RELOAD_RATE:
				{
					return 310;
				}
				case ATTRIBUTE_FIRE_RATE_BONUS:
				{
					return 290;
				}
				case ATTRIBUTE_PROJECTILE_SPEED_INCREASED:
				{
					return 200;
				}
			}
		}
		case TFClass_Medic:
		{
			switch (attribute)
			{
				case ATTRIBUTE_GENERATE_RAGE_ON_HEAL:
				{
					return 320;
				}
				case ATTRIBUTE_UBERCHARGE_RATE_BONUS:
				{
					return 300;
				}
				case ATTRIBUTE_HEALING_MASTERY:
				{
					return 280;
				}
				case ATTRIBUTE_UBER_DURATION_BONUS:
				{
					return 230;
				}
				case ATTRIBUTE_OVERHEAL_EXPERT:
				{
					return 210;
				}
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 40;
				}
				case ATTRIBUTE_FIRE_RATE_BONUS:
				{
					return 40;
				}
			}
		}
		case TFClass_Heavy:
		{
			switch (attribute)
			{
				case ATTRIBUTE_HEAL_ON_KILL:
				{
					return 320;
				}
				case ATTRIBUTE_ATTACK_PROJECTILES:
				{
					return 230;
				}
			}
		}
		case TFClass_Pyro:
		{
			switch (attribute)
			{
				case ATTRIBUTE_DAMAGE_BONUS:
				{
					return 320;
				}
				case ATTRIBUTE_ATTACK_PROJECTILES:
				{
					return 250;
				}
			}
		}
		case TFClass_Spy:
		{
			if (slot != TF_LOADOUT_SLOT_MELEE)
				return 0;

			switch (attribute)
			{
				case ATTRIBUTE_ARMOR_PIERCING:
				{
					return 330;
				}
				case ATTRIBUTE_MELEE_ATTACK_RATE_BONUS:
				{
					return 280;
				}
				case ATTRIBUTE_ROBO_SAPPER:
				{
					return 70;
				}
			}
		}
		case TFClass_Engineer:
		{
			if (slot == TF_LOADOUT_SLOT_PRIMARY || slot == TF_LOADOUT_SLOT_SECONDARY)
			{
				switch (attribute)
				{
					case ATTRIBUTE_DAMAGE_BONUS:
					{
						return 200;
					}
					case ATTRIBUTE_FIRE_RATE_BONUS:
					{
						return 190;
					}
					case ATTRIBUTE_CLIP_SIZE_UPGRADE_ATOMIC:
					{
						return 150;
					}
					case ATTRIBUTE_CLIP_SIZE_BONUS_UPGRADE:
					{
						return 150;
					}
					case ATTRIBUTE_FASTER_RELOAD_RATE:
					{
						return 140;
					}
					case ATTRIBUTE_MAXAMMO_PRIMARY_INCREASED:
					{
						return 130;
					}
					case ATTRIBUTE_MAXAMMO_SECONDARY_INCREASED:
					{
						return 120;
					}
				}

				//Anything else on the gun is worth less than the cheapest thing the nest wants
				return 50;
			}

			switch (attribute)
			{
				case ATTRIBUTE_ENGY_DISPENSER_RADIUS_INCREASED:
				{
					return 330;
				}
				case ATTRIBUTE_ENGY_SENTRY_FIRE_RATE_INCREASED:
				{
					return 320;
				}
				case ATTRIBUTE_ENGY_DISPOSABLE_SENTRIES:
				{
					return Feature(FEATURE_ENGINEER_DISPOSABLE) ? 310 : -10;
				}
				case ATTRIBUTE_ENGY_BUILDING_HEALTH_BONUS:
				{
					return 260;
				}
				case ATTRIBUTE_METAL_REGEN:
				{
					return 220;
				}
				case ATTRIBUTE_MAXAMMO_METAL_INCREASED:
				{
					return 210;
				}
				case ATTRIBUTE_MELEE_ATTACK_RATE_BONUS:
				{
					return 200;
				}
			}
		}
	}

	return 0;
}

/* Damage first, then what keeps it firing. What a bot buys when nothing above had an opinion */
stock int UpgradeRankGeneral(int attribute)
{
	switch (attribute)
	{
		case ATTRIBUTE_DAMAGE_BONUS:
		{
			return 260;
		}
		case ATTRIBUTE_FIRE_RATE_BONUS:
		{
			return 250;
		}
		case ATTRIBUTE_DMG_TAKEN_FROM_BLAST_REDUCED:
		{
			return ResistancePriority(WaveHasExplosiveRobots());
		}
		case ATTRIBUTE_DMG_TAKEN_FROM_BULLETS_REDUCED:
		{
			return ResistancePriority(WaveHasBulletRobots());
		}
		case ATTRIBUTE_DMG_TAKEN_FROM_FIRE_REDUCED:
		{
			return ResistancePriority(WaveHasFireRobots());
		}
		case ATTRIBUTE_MELEE_ATTACK_RATE_BONUS:
		{
			return 200;
		}
		case ATTRIBUTE_PROJECTILE_PENETRATION:
		{
			return 190;
		}
		case ATTRIBUTE_PROJECTILE_PENETRATION_HEAVY:
		{
			return 190;
		}
		case ATTRIBUTE_CRITBOOST_ON_KILL:
		{
			return 180;
		}
		case ATTRIBUTE_CLIP_SIZE_UPGRADE_ATOMIC:
		{
			return 170;
		}
		case ATTRIBUTE_CLIP_SIZE_BONUS_UPGRADE:
		{
			return 170;
		}
		case ATTRIBUTE_FASTER_RELOAD_RATE:
		{
			return 160;
		}
		case ATTRIBUTE_MAXAMMO_PRIMARY_INCREASED:
		{
			return 150;
		}
		case ATTRIBUTE_PROJECTILE_SPEED_INCREASED:
		{
			return 130;
		}
		case ATTRIBUTE_MAXAMMO_SECONDARY_INCREASED:
		{
			return 120;
		}
		case ATTRIBUTE_HEAL_ON_KILL:
		{
			return 110;
		}
		case ATTRIBUTE_MARK_FOR_DEATH:
		{
			return 90;
		}
		case ATTRIBUTE_ARMOR_PIERCING:
		{
			return 85;
		}
		case ATTRIBUTE_ATTACK_PROJECTILES:
		{
			return 80;
		}
		case ATTRIBUTE_INCREASE_BUFF_DURATION:
		{
			return 75;
		}
		case ATTRIBUTE_EFFECT_BAR_RECHARGE_RATE_INCREASED:
		{
			return 70;
		}
		case ATTRIBUTE_CHARGE_RECHARGE_RATE_INCREASED:
		{
			return 70;
		}
		case ATTRIBUTE_GENERATE_RAGE_ON_DAMAGE:
		{
			return 60;
		}
		case ATTRIBUTE_BLEEDING_DURATION:
		{
			return 55;
		}
		case ATTRIBUTE_MOVE_SPEED_BONUS:
		{
			return 45;
		}
		case ATTRIBUTE_HEALTH_REGEN:
		{
			return 40;
		}
		case ATTRIBUTE_DMG_TAKEN_FROM_CRIT_REDUCED:
		{
			return 30;
		}
		case ATTRIBUTE_DAMAGE_FORCE_REDUCTION:
		{
			return 25;
		}
		case ATTRIBUTE_INCREASED_JUMP_HEIGHT:
		{
			return 10;
		}
	}

	return UnrankedUpgradePriority();
}
