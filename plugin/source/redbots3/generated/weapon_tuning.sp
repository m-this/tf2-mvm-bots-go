/* Generated from internal/tables/tuning.go. Do not edit.

The table it comes from is the only place these names are written. */

//No opinion: the caller keeps the range its weapon ID produced.
#define RANGE_TUNING_NONE 0.0

#define SOLDIER_ROCKET_SETTLE	750.0
#define DEMO_PIPE_SETTLE		600.0
#define DEMO_PIPE_FIRE_ANYWAY	1400.0

stock float DemoPipeMaxRange()
{
	return DEMO_PIPE_FIRE_ANYWAY;
}

/* Ranges for one weapon. False when the table says nothing about it, and neither output is
touched, so a caller can pass values it already computed */
stock bool GetTunedWeaponRanges(int weapon, float &desired, float &maxRange)
{
	if (!IsValidEntity(weapon) || !HasEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
		return false;

	switch (GetEntProp(weapon, Prop_Send, "m_iItemDefinitionIndex"))
	{
		//--- Scatterguns. Knockback and damage both want the bot in the target's face
		case 45: //Force-A-Nature
		{
			desired = 180.0;
			maxRange = 600.0;
		}
		case 448: //Soda Popper
		{
			desired = 250.0;
			maxRange = 650.0;
		}

		//--- Shotguns. Past a few hundred units the pellets are worth nothing
		case 425: //Family Business
		{
			desired = 280.0;
			maxRange = 700.0;
		}
		case 1153: //Panic Attack
		{
			desired = 250.0;
			maxRange = 650.0;
		}
		/* The stock shotgun, which was absent and took the five hundred unit default

		Five hundred is minigun ground. Every other shotgun in this table sits between two and
		three hundred, because that is where the pellets are worth anything, and the one four
		classes actually carry was the one nobody had written down. */
		case 199: //Shotgun
		{
			desired = 280.0;
			maxRange = 700.0;
		}
		case 527: //Widowmaker
		{
			desired = 300.0;
			maxRange = 750.0;
		}

		//--- Miniguns. The difference is whether the bot can afford to be caught out of position
		case 312: //Brass Beast
		{
			desired = 350.0;
			maxRange = RANGE_TUNING_NONE;
		}
		case 424: //Tomislav
		{
			desired = 500.0;
			maxRange = RANGE_TUNING_NONE;
		}

		//--- Explosives. Far enough out that the blast does not catch the bot
		case 996: //Loose Cannon
		{
			desired = 650.0;
			maxRange = 1500.0;
		}
		case 1151: //Iron Bomber
		{
			desired = DEMO_PIPE_SETTLE;
			maxRange = DemoPipeMaxRange();
		}
		case 730: //Beggar's Bazooka
		{
			//Shorter than a stock rocket launcher: the rockets spread as they load
			desired = 600.0;
			maxRange = 1500.0;
		}

		//--- Weapons that want the bot to hold its distance
		case 997: //Rescue Ranger
		{
			desired = 800.0;
			maxRange = RANGE_TUNING_NONE;
		}
		case 305: //Crusader's Crossbow
		{
			desired = 900.0;
			maxRange = RANGE_TUNING_NONE;
		}
		case 61: //Ambassador
		{
			desired = 700.0;
			maxRange = RANGE_TUNING_NONE;
		}
		case 412: //Overdose
		{
			//A syringe gun is a close weapon on a class that should not be there
			desired = 300.0;
			maxRange = 800.0;
		}
		default:
		{
			return false;
		}
	}

	return true;
}
