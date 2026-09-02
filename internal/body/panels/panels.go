/*
Package panels is the two readouts out of source/redbots3/menu.sp: what each
class's chance of being drawn is, and what the current lineup is.

A panel asks nothing, so both handlers are empty and live in prefmenu with the
rest of the menu callbacks. The panel itself is the caller's: it is shown and
deleted on the spot, unlike a Menu, which SourceMod keeps.
*/
package panels

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// MenuHandlerShowBotChances hears nothing: the panel it belongs to is a
// readout, and there is nothing to press.
//
//sp:name MenuHandler_ShowBotChances
//nolint:revive // unused-parameter: SourceMod calls this with the full set and the panel answers none of it
func MenuHandlerShowBotChances(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	// Do nothing.
	return 0
}

// MenuHandlerShowBotTeamComposition is the same for the lineup readout.
//
//sp:name MenuHandler_ShowBotTeamComposition
//nolint:revive // unused-parameter: SourceMod calls this with the full set and the panel answers none of it
func MenuHandlerShowBotTeamComposition(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	// Do nothing.
	return 0
}

// The class names the percentages panel labels its rows with, in the order the
// shares arrive.
//
//sp:name PANEL_CLASS_NAMES
var panelClassNames = [9]string{
	"Scout", "Soldier", "Pyro", "Demoman", "Heavy", "Engineer", "Medic", "Sniper", "Spy",
}

/*
CreateDisplayPanelBotPercentages shows each class's share of the draw.

A class with no share is left off rather than shown as zero: the panel is meant
to say what could turn up, and a list of nine with six zeroes in it says that
worse than a list of three.

The shipped file wrote this as nine blocks of the same four lines. The names are
a table here and the shares are already indexed by class, so it is one loop over
both.
*/
//
//sp:name CreateDisplayPanelBotPercentages
//sp:const classPercents
//sp:const duration
//sp:default duration 30
func CreateDisplayPanelBotPercentages(client int32, classPercents [9]float32, duration int32) {
	if engine.IsFakeClient(client) {
		return
	}

	hPanel := engine.NewPanel()
	defer hPanel.Close()

	hPanel.SetTitle("Defender Bot Class Chances")

	for i := int32(0); i < 9; i++ {
		if classPercents[i] > 0.0 {
			itemText := engine.FormatPercent("%s: %.0f%%", panelClassNames[i], classPercents[i])
			hPanel.DrawItem(itemText)
		}
	}

	hPanel.Send(client, MenuHandlerShowBotChances, duration)
}

// CreateDisplayPanelBotTeamComposition shows the lineup as it stands, and says
// whether there was one to show.
//
//sp:name CreateDisplayPanelBotTeamComposition
//sp:const duration
//sp:default duration 30
func CreateDisplayPanelBotTeamComposition(client int32, duration int32) bool {
	if engine.ChosenBotClasses().Length() == 0 {
		return false
	}

	hPanel := engine.NewPanel()
	defer hPanel.Close()

	hPanel.SetTitle("Current Bot Lineup")

	for i := int32(0); i < engine.ChosenBotClasses().Length(); i++ {
		itemText := engine.ChosenBotClasses().GetString(i)
		hPanel.DrawItem(itemText)
	}

	bSuccess := hPanel.Send(client, MenuHandlerShowBotTeamComposition, duration)

	return bSuccess
}
