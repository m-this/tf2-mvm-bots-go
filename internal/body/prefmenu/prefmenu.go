/*
Package prefmenu is the class-preference menu out of source/redbots3/menu.sp:
nine toggles, one per class, saying which classes this player will accept as
teammates.

Every row is a toggle, so the handler writes the opposite of what it reads and
redraws the menu at the row the player was on.
*/
package prefmenu

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// The preference bits, one per class, as playerpref emits them.
const (
	prefScout    = 1
	prefSoldier  = 2
	prefPyro     = 4
	prefDemo     = 8
	prefHeavy    = 16
	prefEngineer = 32
	prefMedic    = 64
	prefSniper   = 128
	prefSpy      = 256
)

// DisplayClassPreferenceMenu shows the nine toggles with their current answers.
//
//sp:name DisplayClassPreferenceMenu
//sp:default item 0
func DisplayClassPreferenceMenu(client int32, item int32) {
	flags := engine.ClassPreferencesFlags(client)

	hClassPrefMenu := engine.CreateMenu(MenuHandlerClassPreference)
	engine.SetMenuTitle(hClassPrefMenu, "Bot Class Preferences")
	engine.SetMenuExitBackButton(hClassPrefMenu, true)
	engine.AddMenuItemText(hClassPrefMenu, "0", engine.ChooseString(flags&prefScout != 0, "Scout: Yes", "Scout: No"))
	engine.AddMenuItemText(hClassPrefMenu, "1", engine.ChooseString(flags&prefSoldier != 0, "Soldier: Yes", "Soldier: No"))
	engine.AddMenuItemText(hClassPrefMenu, "2", engine.ChooseString(flags&prefPyro != 0, "Pyro: Yes", "Pyro: No"))
	engine.AddMenuItemText(hClassPrefMenu, "3", engine.ChooseString(flags&prefDemo != 0, "Demoman: Yes", "Demoman: No"))
	engine.AddMenuItemText(hClassPrefMenu, "4", engine.ChooseString(flags&prefHeavy != 0, "Heavy: Yes", "Heavy: No"))
	engine.AddMenuItemText(hClassPrefMenu, "5", engine.ChooseString(flags&prefEngineer != 0, "Engineer: Yes", "Engineer: No"))
	engine.AddMenuItemText(hClassPrefMenu, "6", engine.ChooseString(flags&prefMedic != 0, "Medic: Yes", "Medic: No"))
	engine.AddMenuItemText(hClassPrefMenu, "7", engine.ChooseString(flags&prefSniper != 0, "Sniper: Yes", "Sniper: No"))
	engine.AddMenuItemText(hClassPrefMenu, "8", engine.ChooseString(flags&prefSpy != 0, "Spy: Yes", "Spy: No"))
	engine.DisplayMenuAtItem(hClassPrefMenu, client, item, engine.MenuTimeForever())
}

/*
MenuHandlerClassPreference flips whichever class was pressed.

It writes the opposite of what it reads, which is what a toggle is, and redraws
at the same row so the player can flip several without hunting for their place.
*/
//
//sp:name MenuHandler_ClassPreference
func MenuHandlerClassPreference(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		flags := engine.ClassPreferencesFlags(param1)

		switch param2 {
		case 0:
			engine.SetClassPreferences(param1, "scout", engine.ChooseInt(flags&prefScout != 0, 0, 1))
		case 1:
			engine.SetClassPreferences(param1, "soldier", engine.ChooseInt(flags&prefSoldier != 0, 0, 1))
		case 2:
			engine.SetClassPreferences(param1, "pyro", engine.ChooseInt(flags&prefPyro != 0, 0, 1))
		case 3:
			engine.SetClassPreferences(param1, "demoman", engine.ChooseInt(flags&prefDemo != 0, 0, 1))
		case 4:
			engine.SetClassPreferences(param1, "heavyweapons", engine.ChooseInt(flags&prefHeavy != 0, 0, 1))
		case 5:
			engine.SetClassPreferences(param1, "engineer", engine.ChooseInt(flags&prefEngineer != 0, 0, 1))
		case 6:
			engine.SetClassPreferences(param1, "medic", engine.ChooseInt(flags&prefMedic != 0, 0, 1))
		case 7:
			engine.SetClassPreferences(param1, "sniper", engine.ChooseInt(flags&prefSniper != 0, 0, 1))
		case 8:
			engine.SetClassPreferences(param1, "spy", engine.ChooseInt(flags&prefSpy != 0, 0, 1))
		}

		DisplayClassPreferenceMenu(param1, engine.MenuSelectionPosition())
	case engine.MenuEnd():
		menu.Close()
	case engine.MenuCancel():
		if param2 == engine.MenuCancelExitBack() {
			engine.DisplayMenu(engine.BotPreferenceMenu(), param1, engine.MenuTimeForever())
		}
	}

	return 0
}
