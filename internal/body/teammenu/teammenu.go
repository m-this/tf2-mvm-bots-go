/*
Package teammenu is the team-picking menu out of source/redbots3/menu.sp: the
panel a player builds a lineup in, and the handler that hears each choice.

The two halves belong together because the handler redraws the menu, and the
builder names the handler. SourceMod takes the handler by name, which is what
the emitter writes.
*/
package teammenu

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// ChooseBotClassesTime is how long the picking menu stays up.
//
//sp:name CHOOSE_BOT_CLASSES_TIME
const ChooseBotClassesTime = 30

//sp:name m_iBotsLeftToChoose
var botsLeftToChoose int32

/*
ShowDefenderBotTeamSetupMenu puts the class list up, once per seat left to
fill.

Initialising is a separate flag rather than a count of zero: the menu is redrawn
after every choice with the same function, and a redraw must not clear what has
been chosen so far.
*/
//
//sp:name ShowDefenderBotTeamSetupMenu
//sp:default itemPosition 0
//sp:default bInitialize false
//sp:default numBotsToAdd 0
func ShowDefenderBotTeamSetupMenu(client int32, itemPosition int32, bInitialize bool, numBotsToAdd int32) {
	if bInitialize {
		engine.ChosenBotClasses().Clear()
		botsLeftToChoose = numBotsToAdd
	}

	hMenu := engine.NewMenu(MenuHandlerDefenderBotTeamSetup)
	hMenu.SetTitle("Create Your Team (%d)", botsLeftToChoose)
	hMenu.AddItem("0", "Scout")
	hMenu.AddItem("1", "Soldier")
	hMenu.AddItem("2", "Pyro")
	hMenu.AddItem("3", "Demoman")
	hMenu.AddItem("4", "Heavy")
	hMenu.AddItem("5", "Engineer")
	hMenu.AddItem("6", "Medic")
	hMenu.AddItem("7", "Sniper")
	hMenu.AddItem("8", "Spy")
	hMenu.DisplayAt(client, itemPosition, ChooseBotClassesTime)

	if bInitialize {
		engine.SetChoosingBotClasses(client, true)
	}
}

/*
MenuHandlerDefenderBotTeamSetup hears one choice, then either asks for the next
seat or moves to the confirmation.

The menu deletes itself on End rather than in the builder: SourceMod keeps it
alive until the player is done with it, so the builder does not own it.
*/
//
//sp:name MenuHandler_DefenderBotTeamSetup
func MenuHandlerDefenderBotTeamSetup(menu engine.Menu, action engine.MenuChoice, param1 int32, param2 int32) int32 {
	switch action {
	case engine.MenuSelect():
		switch param2 {
		case 0:
			engine.ChosenBotClasses().PushString("scout")
			engine.PrintToChat(param1, "You have chosen Scout")
		case 1:
			engine.ChosenBotClasses().PushString("soldier")
			engine.PrintToChat(param1, "You have chosen Soldier")
		case 2:
			engine.ChosenBotClasses().PushString("pyro")
			engine.PrintToChat(param1, "You have chosen Pyro")
		case 3:
			engine.ChosenBotClasses().PushString("demoman")
			engine.PrintToChat(param1, "You have chosen Demoman")
		case 4:
			engine.ChosenBotClasses().PushString("heavyweapons")
			engine.PrintToChat(param1, "You have chosen Heavy")
		case 5:
			engine.ChosenBotClasses().PushString("engineer")
			engine.PrintToChat(param1, "You have chosen Engineer")
		case 6:
			engine.ChosenBotClasses().PushString("medic")
			engine.PrintToChat(param1, "You have chosen Medic")
		case 7:
			engine.ChosenBotClasses().PushString("sniper")
			engine.PrintToChat(param1, "You have chosen Sniper")
		case 8:
			engine.ChosenBotClasses().PushString("spy")
			engine.PrintToChat(param1, "You have chosen Spy")
		}

		botsLeftToChoose--

		if botsLeftToChoose <= 0 {
			engine.ShowDefenderBotTeamConfirmationMenu(param1)
			return 0
		}

		ShowDefenderBotTeamSetupMenu(param1, engine.MenuSelectionPosition(), false, 0)
	case engine.MenuCancel():
		engine.SetChoosingBotClasses(param1, false)
		engine.DefenderBotTeamSetupCancelled()
		engine.PrintToChatAll("%s %N is no longer selecting the bot team.", engine.PluginPrefix(), param1)
	case engine.MenuEnd():
		menu.Close()
	}

	return 0
}
