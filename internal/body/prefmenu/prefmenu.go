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
			engine.DisplayMenu(botPreferenceMenu, param1, engine.MenuTimeForever())
		}
	}

	return 0
}

/*
MenuHandlerBotVote hears the round's vote on whether to have bots at all.

Who called it is remembered on a yes and forgotten on anything else: the caller
is the one player allowed to send the bots away again, and a vote that failed
gives nobody that.
*/
//
//sp:name MenuHandler_BotVote
//
//nolint:revive,staticcheck // unused-parameter, QF1003: the menu and param2 are SourceMod's, and the if-else is the shipped shape
func MenuHandlerBotVote(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuVoteEnd():
		if param1 == 0 {
			// They said yes.
			engine.ManageDefenderBotsOn(true)
		} else if param1 == 1 {
			// They said no. Forget who called the vote, as they were not
			// able to summon bots.
			engine.SetBotSummoner(0)

			engine.PrintToChatAll("%s Bot vote was unsuccessful!", engine.PluginPrefix())
		}
	case engine.MenuVoteCancel():
		engine.SetBotSummoner(0)
	}

	return 0
}

/*
DefenderBotTeamSetupCancelled puts things back when nobody finished picking.

What back means depends on the mode: with preferences behind it the lineup is
recomputed from what the players asked for, and without them the half-built
lineup is cleared, or the manager would think one had been chosen.
*/
//
//sp:name DefenderBotTeamSetupCancelled
func DefenderBotTeamSetupCancelled() {
	switch engine.BotLineupMode().Int() {
	case engine.LineupModePreferenceChoose():
		engine.UpdateChosenBotTeamComposition()
	case engine.LineupModeChoose():
		engine.ChosenBotClasses().Clear()
	}
}

// MenuHandlerBotPreferenceMain is the root of the preference menus: classes,
// or weapons when the server allows custom loadouts at all.
//
//sp:name MenuHandler_BotPreferenceMain
//nolint:revive,gocritic // unused-parameter, singleCaseSwitch: the menu is SourceMod's, and the switch is the shipped shape
func MenuHandlerBotPreferenceMain(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		switch param2 {
		case 0:
			DisplayClassPreferenceMenu(param1, 0)
		case 1:
			if !engine.UseCustomLoadouts().Bool() {
				engine.PrintToChat(param1, "%s Custom loadouts are not enabled.", engine.PluginPrefix())
				return 0
			}

			engine.DisplayMenu(weaponPrefClassMenu, param1, engine.MenuTimeForever())
		}
	}

	return 0
}

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

// The class a player is picking a loadout for, so every menu below the class
// list knows which one it is talking about.
//
//sp:name m_sSelectedClass
var selectedClass [65][16]byte

/*
The two menus that are kept rather than shown and forgotten.

Every submenu below them backs out to one of the two, so they have to outlive
the call that built them: that is what makes them globals and why
CreateBotPreferenceMenu deletes the previous pair before making a new one.
*/

//sp:name g_hBotPreferenceMenu
var botPreferenceMenu engine.Menu

//sp:name m_hWeaponPrefClassMenu
var weaponPrefClassMenu engine.Menu

// CreateBotPreferenceMenu builds the two kept menus, replacing whatever pair
// was there.
//
//sp:name CreateBotPreferenceMenu
func CreateBotPreferenceMenu() {
	botPreferenceMenu.Close()

	botPreferenceMenu = engine.CreateMenu(MenuHandlerBotPreferenceMain)
	engine.SetMenuTitle(botPreferenceMenu, "Teammate Bot Preferences")
	engine.AddMenuItem(botPreferenceMenu, "0", "Class")
	engine.AddMenuItem(botPreferenceMenu, "1", "Weapons")

	weaponPrefClassMenu.Close()

	weaponPrefClassMenu = engine.CreateMenu(MenuHandlerWeaponPreferenceClassList)
	engine.SetMenuExitBackButton(weaponPrefClassMenu, true)
	engine.AddMenuItem(weaponPrefClassMenu, "0", "Scout")
	engine.AddMenuItem(weaponPrefClassMenu, "1", "Soldier")
	engine.AddMenuItem(weaponPrefClassMenu, "2", "Pyro")
	engine.AddMenuItem(weaponPrefClassMenu, "3", "Demoman")
	engine.AddMenuItem(weaponPrefClassMenu, "4", "Heavy")
	engine.AddMenuItem(weaponPrefClassMenu, "5", "Engineer")
	engine.AddMenuItem(weaponPrefClassMenu, "6", "Medic")
	engine.AddMenuItem(weaponPrefClassMenu, "7", "Sniper")
	engine.AddMenuItem(weaponPrefClassMenu, "8", "Spy")
}

/*
DisplayWeaponPreferenceMenu shows the slots of one class, each row saying what
is picked for it.

The spy gets a fourth row: his watch is a loadout slot the other classes have
nothing in.
*/
//
//sp:name DisplayWeaponPreferenceMenu
//sp:writable class
//sp:default item 0
func DisplayWeaponPreferenceMenu(client int32, class string, item int32) {
	// Tell us the class we just chose so everything else will get the correct
	// data for this class.
	selectedClass[client] = engine.CopyShort(class)

	hWeaponPrefMenu := engine.CreateMenu(MenuHandlerWeaponPreference)
	engine.SetMenuTitleFor(hWeaponPrefMenu, "Bot Weapon Preferences: %s", class)
	engine.SetMenuExitBackButton(hWeaponPrefMenu, true)
	engine.AddMenuItemFrom(hWeaponPrefMenu, "0", engine.WeaponPrefMenuItemText(client, class, engine.WeaponSlotPrimary()))
	engine.AddMenuItemFrom(hWeaponPrefMenu, "1", engine.WeaponPrefMenuItemText(client, class, engine.WeaponSlotSecondary()))
	engine.AddMenuItemFrom(hWeaponPrefMenu, "2", engine.WeaponPrefMenuItemText(client, class, engine.WeaponSlotMelee()))

	if engine.StrEqualLiteral(class, "spy", false) {
		engine.AddMenuItemFrom(hWeaponPrefMenu, "3", engine.WeaponPrefMenuItemText(client, class, engine.WeaponSlotItem1()))
	}

	engine.DisplayMenuAtItem(hWeaponPrefMenu, client, item, engine.MenuTimeForever())
}

// MenuHandlerWeaponPreferenceClassList picks which class the loadout menus are
// about.
//
//sp:name MenuHandler_WeaponPreferenceClassList
//nolint:revive // unused-parameter: the menu is SourceMod's
func MenuHandlerWeaponPreferenceClassList(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		switch param2 {
		case 0:
			DisplayWeaponPreferenceMenu(param1, "scout", 0)
		case 1:
			DisplayWeaponPreferenceMenu(param1, "soldier", 0)
		case 2:
			DisplayWeaponPreferenceMenu(param1, "pyro", 0)
		case 3:
			DisplayWeaponPreferenceMenu(param1, "demoman", 0)
		case 4:
			DisplayWeaponPreferenceMenu(param1, "heavyweapons", 0)
		case 5:
			DisplayWeaponPreferenceMenu(param1, "engineer", 0)
		case 6:
			DisplayWeaponPreferenceMenu(param1, "medic", 0)
		case 7:
			DisplayWeaponPreferenceMenu(param1, "sniper", 0)
		case 8:
			DisplayWeaponPreferenceMenu(param1, "spy", 0)
		}
	case engine.MenuCancel():
		if param2 == engine.MenuCancelExitBack() {
			engine.DisplayMenu(botPreferenceMenu, param1, engine.MenuTimeForever())
		}
	}

	return 0
}

/*
MenuHandlerWeaponPreference picks which slot of the chosen class to list items
for.

Row three is only ever pressed by a spy: the builder does not add it for anyone
else, so the branch cannot be reached from another class's menu.
*/
//
//sp:name MenuHandler_WeaponPreference
func MenuHandlerWeaponPreference(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		switch param2 {
		case 0:
			engine.ShowWeaponPreferenceItemListMenu(param1, selectedClass[param1], "primary")
		case 1:
			engine.ShowWeaponPreferenceItemListMenu(param1, selectedClass[param1], "secondary")
		case 2:
			engine.ShowWeaponPreferenceItemListMenu(param1, selectedClass[param1], "melee")
		case 3:
			engine.ShowWeaponPreferenceItemListMenu(param1, selectedClass[param1], "pda2")
		}
	case engine.MenuEnd():
		menu.Close()
	case engine.MenuCancel():
		if param2 == engine.MenuCancelExitBack() {
			engine.DisplayMenu(weaponPrefClassMenu, param1, engine.MenuTimeForever())
		}
	}

	return 0
}
