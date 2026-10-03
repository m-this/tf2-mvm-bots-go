/* Calling the bot's own script functions.

CTFBot exposes PressFireButton and the rest to VScript, and nothing exposes them
to SourceMod. The RunScriptCode input is the way in: a line of Squirrel with the
entity as self. */

stock bool RunScriptCode(int entity, int activator = -1, int caller = -1, const char[] fmt, any ...)
{
	char code[PLATFORM_MAX_PATH];
	VFormat(code, sizeof(code), fmt, 5);

	SetVariantString(code);

	return AcceptEntityInput(entity, "RunScriptCode", activator, caller);
}

stock void VS_PressFireButton(int bot, float duration = -1.0)
{
	RunScriptCode(bot, _, _, "self.PressFireButton(%f)", duration);
}

stock void VS_PressAltFireButton(int bot, float duration = -1.0)
{
	RunScriptCode(bot, _, _, "self.PressAltFireButton(%f)", duration);
}

stock void VS_PressSpecialFireButton(int bot, float duration = -1.0)
{
	RunScriptCode(bot, _, _, "self.PressSpecialFireButton(%f)", duration);
}

stock void VS_AddBotAttribute(int bot, int attributeFlags)
{
	RunScriptCode(bot, _, _, "self.AddBotAttribute(%d)", attributeFlags);
}

stock void VS_GrantOrRemoveAllUpgrades(int client, bool remove, bool refund)
{
	RunScriptCode(client, _, _, "self.GrantOrRemoveAllUpgrades(%s, %s)",
		remove ? "true" : "false", refund ? "true" : "false");
}
