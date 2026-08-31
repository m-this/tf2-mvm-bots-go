/*
Package archipelago is source/redbots3/archipelago.sp: the money the game's own
record does not know about.

tf2-archipelago hands out Cash Bundles by writing m_nCurrency straight onto a
player, so that money is real on the screen and absent from the record the game
keeps of the wave. This mod sets a bot's currency from that record whenever one
joins, which is right for everything the game paid and wrong for everything
Archipelago did. So this asks. The plugin is optional, and a server without it
has no bundles, which is the same answer as zero.
*/
package archipelago

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Whether the Archipelago plugin is loaded and its native is callable.
//
//sp:name g_bArchipelago
var archipelagoLoaded bool

// Init is the map start half: one first look.
//
//sp:name Archipelago_Init
func Init() {
	Recheck()
}

// Recheck is asked again on every library change, because the plugin can be
// loaded, unloaded or reloaded while this one is running, and a native that was
// there at map start is not always there now.
//
//sp:name Archipelago_Recheck
func Recheck() {
	archipelagoLoaded = engine.NativeStatus("TF2AP_GetBundleCredits") == engine.FeatureAvailable()
}

/*
Credits is everything Archipelago has paid this client that the game never
recorded, or zero.

Zero covers three cases that all mean the same thing here: no plugin, a plugin
too old to have the native, and a run that has been given no bundles yet.
*/
//
//sp:name GetArchipelagoCredits
func Credits(client int32) int32 {
	if !archipelagoLoaded {
		return 0
	}

	return engine.BundleCredits(client)
}

/*
SetCurrencyWithBundles is the currency a defender bot should hold, bundles
included.

Every place this mod decides a bot's balance from the game's accounting goes
through here rather than through TF2_SetCurrency, so there is one place that
knows Archipelago exists and no way to add a second call site that forgets.
*/
//
//sp:name SetCurrencyWithBundles
func SetCurrencyWithBundles(client int32, earned int32) {
	engine.SetCurrency(client, earned+Credits(client))
}
