/* Hats and unusual effects on the defender bots, for the look of the thing

None of this changes how a bot plays. What the team looks like now is six bare-headed mercenaries
with the same three weapons, and a run built out of a randomiser is more fun to watch when they
look like six strangers who met on the way in.

Both pools come from the game's own item schema through tf_econ_data, never from a table of
numbers in this file. A table of item definition indexes is a guess that goes stale on the next
update, and the schema is what the client renders from anyway. Each pool costs one walk of the
schema, once, and is then kept.

Attribute 134 attaches the particle, read out of scripts/items/items_game.txt rather than copied
from a forum post.

War paints on the weapons were here too and are gone: they are attributes on the entity the
upgrade station rewrites all wave, and they were not worth one more thing to blame a crash on.

Defender bots only, and never the invading robots: a wave is read by silhouette, and a robot in a
hat is a robot somebody shoots a moment later than they should. */


/* Half a second after the bot spawns, not the moment it does

The game gives its own items on spawn, and the custom loadout replaces them a tenth of a second
later, and a hat handed to a bot in the middle of that is a hat the game throws away.

/* Put the drawn hat back on

/* Wearables the game refused, standing in the world with nobody wearing them
 *
 * TF2Util_EquipPlayerWearable asserts that the wearable ended up attached, and throws when the
 * game declined it. A thrown native takes the rest of the callback with it, so the hat entity that
 * was created a few lines earlier is never cleaned up and never worn: it just stays. Seen in the
 * error log dozens of times across a mission, and a server that leaks an edict per bot per respawn
 * eventually runs out of them.
 *
 * The equip cannot be tested in advance, so the leak is swept instead of prevented. Anything of
 * ours whose owner is not a player in the game is nobody's hat.
 */
/* Take the hat off the way the game does it

/* The model this class wears this hat with

/* A bot that has left takes its hat with it

/* What every defender bot is actually wearing, entity by entity

A hat that does not show up is one of four things and they look the same from the outside: never
drawn, drawn and refused a model, worn by an entity the game threw away, or worn by an entity with
no model index on it. This says which. */
public Action Command_DumpHats(int client, int args)
{
	for (int playerClass = 1; playerClass < sizeof(g_adtHats); playerClass++)
	{
		ArrayList pool = HatPoolForClass(view_as<TFClassType>(playerClass));

		ReplyToCommand(client, "class %d: %d hats in the pool", playerClass, pool == null ? -1 : pool.Length);
	}

	for (int i = 1; i <= MaxClients; i++)
	{
		if (!IsClientInGame(i) || !g_bIsDefenderBot[i])
			continue;

		int hat = EntRefToEntIndex(g_iBotHat[i]);

		if (hat == INVALID_ENT_REFERENCE)
		{
			ReplyToCommand(client, "%N: class %d, drew item %d, effect %d, wearing nothing",
				i, g_wardrobe[i].PlayerClass, g_wardrobe[i].HatItem, g_wardrobe[i].HatEffect);

			continue;
		}

		char model[PLATFORM_MAX_PATH]; GetEntPropString(hat, Prop_Data, "m_ModelName", model, sizeof(model));

		ReplyToCommand(client, "%N: class %d, item %d, effect %d, entity %d, owner %d, modelindex %d, model %s",
			i, g_wardrobe[i].PlayerClass, g_wardrobe[i].HatItem, g_wardrobe[i].HatEffect, hat,
			GetEntPropEnt(hat, Prop_Send, "m_hOwnerEntity"),
			GetEntProp(hat, Prop_Send, "m_nModelIndex"), model);
	}

	return Plugin_Handled;
}

/* Every cosmetic this class may wear, asked of the schema once

Not the medals. What the bots actually drew, almost every time, was a UGC participation medal or
an ozfortress season badge: a postage stamp on the chest that reads in game as a bot wearing
nothing at all, while anything with a particle on it still drew the particle, which is why the
effects looked like the only part that worked. There are far more tournament medals in the schema
than there are cosmetics, so drawing uniformly from the slot is drawing a medal.

Filed by equip region rather than by slot, because the slot cannot tell them apart: the head slot
is the game's old single-hat one and no modern item reports it, so every cosmetic and every medal
alike comes back as misc. The medals are the ones the schema puts in the "medal" region, which is
one string off a prefab they all share.

