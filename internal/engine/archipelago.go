package engine

/*
The seam to tf2-archipelago, which is optional and can come and go while this
plugin runs.
*/

// ArchipelagoCalls are the answers.
type ArchipelagoCalls struct {
	BundleCredits func(client int32) int32
	NativeStatus  func(name string) int32
	SetCurrency   func(client int32, amount int32)
}

var archipelago ArchipelagoCalls

// InstallArchipelago puts a set of answers behind them.
func InstallArchipelago(c ArchipelagoCalls) func() {
	previous := archipelago
	Fill(&c)
	archipelago = c
	return func() { archipelago = previous }
}

// BundleCredits is TF2AP_GetBundleCredits, everything Archipelago paid this
// client that the game never recorded.
//
//sp:native TF2AP_GetBundleCredits
func BundleCredits(client int32) int32 { return archipelago.BundleCredits(client) }

// NativeStatus asks whether an optional native exists right now, which is
// GetFeatureStatus with the question fixed to natives.
//
//sp:native GetFeatureStatus before FeatureType_Native
func NativeStatus(name string) int32 { return archipelago.NativeStatus(name) }

// FeatureAvailable is FeatureStatus_Available, the one answer that means yes.
//
//sp:global FeatureStatus_Available
func FeatureAvailable() int32 { return 0 }

// SetCurrency writes a player's balance.
//
//sp:native TF2_SetCurrency
func SetCurrency(client int32, amount int32) { archipelago.SetCurrency(client, amount) }
