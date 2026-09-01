/* The money the game's own record does not know about
 *
 * tf2-archipelago hands out Cash Bundles by writing m_nCurrency straight onto a
 * player, because a currency pack on this server cannot be told what it is
 * worth. That money is real on the screen and absent from the record the game
 * keeps of the wave.
 *
 * This mod sets a bot's currency from that record whenever one joins, which is
 * right for everything the game paid and wrong for everything Archipelago did:
 * a bot that rejoins, or changes class mid-mission, comes back without a single
 * credit of it. On a six seat team that is five players' worth of bundles
 * thrown away every time somebody reseats.
 *
 * So this asks. The plugin is optional and this mod runs on servers that have
 * never heard of Archipelago, so the native is declared optional and its
 * absence is zero rather than an error: a server without the plugin has no
 * bundles, which is the same answer.
 */

native int TF2AP_GetBundleCredits(int client);

//Whether the Archipelago plugin is loaded and its native is callable
static bool g_bArchipelago;

void Archipelago_Init()
{
	Archipelago_Recheck();
}

/* Asked again on every library change, because the plugin can be loaded, unloaded or reloaded
while this one is running, and a native that was there at map start is not always there now. */
void Archipelago_Recheck()
{
	g_bArchipelago = GetFeatureStatus(FeatureType_Native, "TF2AP_GetBundleCredits") == FeatureStatus_Available;
}

/* Everything Archipelago has paid this client that the game never recorded, or zero

Zero covers three cases that all mean the same thing here: no plugin, a plugin too old to have
the native, and a run that has been given no bundles yet. */
int GetArchipelagoCredits(int client)
{
	if (!g_bArchipelago)
		return 0;

	return TF2AP_GetBundleCredits(client);
}

/* The currency a defender bot should hold, bundles included

Every place this mod decides a bot's balance from the game's accounting goes through here rather
than through TF2_SetCurrency, so there is one place that knows Archipelago exists and no way to
add a second call site that forgets. */
void SetCurrencyWithBundles(int client, int earned)
{
	TF2_SetCurrency(client, earned + GetArchipelagoCredits(client));
}
