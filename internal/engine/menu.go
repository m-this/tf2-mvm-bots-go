package engine

/*
SourceMod's menus: the panel a player is shown, and the callback that hears
what they pressed.

A menu is a handle the plugin makes, shows once and lets go of: the End action
is where it is deleted, so the lifetime belongs to the callback rather than to
the function that built it. That is what //sp:new Menu borrowed says, and why
Close is only ever called from the handler.
*/

// MenuCalls are the answers.
type MenuCalls struct {
	NewMenu             func(handler string) Menu
	SetTitle            func(m Menu, format string, args []any)
	AddItem             func(m Menu, info string, display string)
	Display             func(m Menu, client int32, time int32)
	DisplayAt           func(m Menu, client int32, position int32, time int32)
	DeleteMenu          func(m Menu)
	SelectionPos        func() int32
	SetChoosing         func(client int32, choosing bool)
	PrintToChat         func(client int32, format string, args []any)
	ChosenClasses       func() List
	ShowConfirmation    func(client int32)
	SetupCancelled      func()
	SetBotClassesLocked func(locked bool)
}

var menus MenuCalls

// InstallMenus puts a set of answers behind them.
func InstallMenus(c MenuCalls) func() {
	previous := menus
	menus = c
	return func() { menus = previous }
}

// Menu is SourceMod's Menu.
//
//sp:tag Menu
type Menu int32

// MenuChoice is SourceMod's MenuAction, which of the menu's events fired.
//
//sp:tag MenuAction
type MenuChoice int32

// MenuSelect is MenuAction_Select, a player pressed an item.
//
//sp:global MenuAction_Select
func MenuSelect() MenuChoice { return 4 }

// MenuCancel is MenuAction_Cancel, they closed it instead.
//
//sp:global MenuAction_Cancel
func MenuCancel() MenuChoice { return 8 }

// MenuEnd is MenuAction_End, the menu is finished and owns nothing more.
//
//sp:global MenuAction_End
func MenuEnd() MenuChoice { return 16 }

// NewMenu makes one behind a handler named by the emitter.
//
//sp:new Menu borrowed
//nolint:revive // unused-parameter: the handler is a name the emitter writes
func NewMenu(handler func(menu Menu, action MenuChoice, param1 int32, param2 int32) int32) Menu {
	if menus.NewMenu == nil {
		missing("new Menu")
	}
	return menus.NewMenu("")
}

// SetTitle writes the line above the items.
//
//sp:method SetTitle
func (m Menu) SetTitle(format string, args ...any) {
	if menus.SetTitle == nil {
		missing("Menu.SetTitle")
	}
	menus.SetTitle(m, format, args)
}

// AddItem adds one row: what comes back on select, and what is shown.
//
//sp:method AddItem
func (m Menu) AddItem(info string, display string) {
	if menus.AddItem == nil {
		missing("Menu.AddItem")
	}
	menus.AddItem(m, info, display)
}

// Display shows it for that many seconds.
//
//sp:method Display
func (m Menu) Display(client int32, time int32) {
	if menus.Display == nil {
		missing("Menu.Display")
	}
	menus.Display(m, client, time)
}

// DisplayAt shows it scrolled to an item, which is how a menu redrawn after a
// selection keeps its place.
//
//sp:method DisplayAt
func (m Menu) DisplayAt(client int32, position int32, time int32) {
	if menus.DisplayAt == nil {
		missing("Menu.DisplayAt")
	}
	menus.DisplayAt(m, client, position, time)
}

// Close deletes it, which only the End action should do.
//
//sp:delete Close
func (m Menu) Close() {
	if menus.DeleteMenu == nil {
		missing("delete Menu")
	}
	menus.DeleteMenu(m)
}

// MenuSelectionPosition is the row the player was scrolled to.
//
//sp:native GetMenuSelectionPosition
func MenuSelectionPosition() int32 {
	if menus.SelectionPos == nil {
		missing("GetMenuSelectionPosition")
	}
	return menus.SelectionPos()
}

// SetChoosingBotClasses writes g_bChoosingBotClasses.
//
//sp:slotset g_bChoosingBotClasses
func SetChoosingBotClasses(client int32, choosing bool) {
	if menus.SetChoosing == nil {
		missing("g_bChoosingBotClasses")
	}
	menus.SetChoosing(client, choosing)
}

// PrintToChat says one line to one player.
//
//sp:native PrintToChat
func PrintToChat(client int32, format string, args ...any) {
	if menus.PrintToChat == nil {
		missing("PrintToChat")
	}
	menus.PrintToChat(client, format, args)
}

// ChosenBotClasses is g_adtChosenBotClasses, the lineup a player is building.
//
//sp:global g_adtChosenBotClasses
func ChosenBotClasses() List {
	if menus.ChosenClasses == nil {
		missing("g_adtChosenBotClasses")
	}
	return menus.ChosenClasses()
}

// DefenderBotTeamSetupCancelled puts the vote back when nobody finished
// picking. Still in menu.sp.
//
//sp:plugin DefenderBotTeamSetupCancelled
func DefenderBotTeamSetupCancelled() {
	if menus.SetupCancelled == nil {
		missing("DefenderBotTeamSetupCancelled")
	}
	menus.SetupCancelled()
}

// SetBotClassesLocked writes g_bBotClassesLocked: the lineup a player accepted
// is held until the bots that use it are seated.
//
//sp:globalset g_bBotClassesLocked
func SetBotClassesLocked(locked bool) {
	if menus.SetBotClassesLocked == nil {
		missing("g_bBotClassesLocked")
	}
	menus.SetBotClassesLocked(locked)
}
