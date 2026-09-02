/*
Package addmenu is the manual add-bots menu out of source/redbots3/menu.sp: an
admin picking classes one at a time.

Every choice is logged with the admin log rather than the plugin log, because
adding a bot changes the round for everybody and the server owner is who has to
be able to see who did it.
*/
package addmenu

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// CreateDisplayMenuAddDefenderBots shows the nine classes, redrawn at the same
// row after each pick so an admin can add several.
//
//sp:name CreateDisplayMenuAddDefenderBots
//sp:default itemPosition 0
func CreateDisplayMenuAddDefenderBots(client int32, itemPosition int32) {
	if engine.IsFakeClient(client) {
		return
	}

	hMenu := engine.NewMenu(MenuHandlerAddDefenderBots)
	hMenu.SetTitle("Manually add bots")
	hMenu.AddItem("0", "Scout")
	hMenu.AddItem("1", "Soldier")
	hMenu.AddItem("2", "Pyro")
	hMenu.AddItem("3", "Demoman")
	hMenu.AddItem("4", "Heavy")
	hMenu.AddItem("5", "Engineer")
	hMenu.AddItem("6", "Medic")
	hMenu.AddItem("7", "Sniper")
	hMenu.AddItem("8", "Spy")
	hMenu.DisplayAt(client, itemPosition, engine.MenuTimeForever())
}

// MenuHandlerAddDefenderBots adds one bot of the class pressed.
//
//sp:name MenuHandler_AddDefenderBots
func MenuHandlerAddDefenderBots(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		switch param2 {
		case 0:
			engine.AddDefenderTFBotNamed(1, "scout", "red", "expert")
		case 1:
			engine.AddDefenderTFBotNamed(1, "soldier", "red", "expert")
		case 2:
			engine.AddDefenderTFBotNamed(1, "pyro", "red", "expert")
		case 3:
			engine.AddDefenderTFBotNamed(1, "demoman", "red", "expert")
		case 4:
			engine.AddDefenderTFBotNamed(1, "heavyweapons", "red", "expert")
		case 5:
			engine.AddDefenderTFBotNamed(1, "engineer", "red", "expert")
		case 6:
			engine.AddDefenderTFBotNamed(1, "medic", "red", "expert")
		case 7:
			engine.AddDefenderTFBotNamed(1, "sniper", "red", "expert")
		case 8:
			engine.AddDefenderTFBotNamed(1, "spy", "red", "expert")
		}

		CreateDisplayMenuAddDefenderBots(param1, engine.MenuSelectionPosition())
		engine.LogAction(param1, -1, "MenuHandler_AddDefenderBots: %L, select %d", param1, param2)
		return 0
	case engine.MenuEnd():
		menu.Close()
		return 0
	}

	return 0
}
