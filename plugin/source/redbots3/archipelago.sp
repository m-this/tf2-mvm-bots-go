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

