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
	AddDefenderTFBotNamed  func(count int32, class string, team string, difficulty string)
	LogAction              func(client int32, target int32, format string, args []any)
	SetMenuTitleFor        func(m Menu, format string, class string)
	CopyShort              func(from string) [16]byte
	AddMenuItemFrom        func(m Menu, info string, display Text)
	ShowWeaponItemList     func(client int32, class [16]byte, slot string)
	NewMenu                func(handler string) Menu
	SetTitle               func(m Menu, format string, args []any)
	AddItem                func(m Menu, info string, display string)
	Display                func(m Menu, client int32, time int32)
	DisplayAt              func(m Menu, client int32, position int32, time int32)
	DeleteMenu             func(m Menu)
	SelectionPos           func() int32
	SetChoosing            func(client int32, choosing bool)
	PrintToChat            func(client int32, format string, args []any)
	ChosenClasses          func() List
	ShowConfirmation       func(client int32)
	SetupCancelled         func()
	SetBotClassesLocked    func(locked bool)
	CreateMenu             func(handler string) Menu
	SetMenuTitle           func(m Menu, title string)
	AddMenuItem            func(m Menu, info string, display string)
	AddMenuItemText        func(m Menu, info string, display string)
	SetMenuExitBackButton  func(m Menu, on bool)
	DisplayMenu            func(m Menu, client int32, time int32)
	DisplayMenuAtItem      func(m Menu, client int32, item int32, time int32)
	ClassPreferencesFlags  func(client int32) int32
	SetClassPreferences    func(client int32, class string, value int32)
	SetBotSummoner         func(userid int32)
	WeaponPrefMenuItemText func(client int32, class string, slot int32) Text
	ManageDefenderBots     func(manage bool)
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

// DefenderBotTeamSetupCancelled puts things back when nobody finished picking.
// Ported, prefmenu.
//
//sp:body DefenderBotTeamSetupCancelled
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

/*
The function-style menu API.

SourceMod has both: the methodmap the team menu uses, and these, which the
preference menus were written against. They are the same objects, so a menu made
by one can be shown by the other; the port keeps whichever the shipped file
wrote.
*/

// CreateMenu makes one behind a handler named by the emitter.
//
//sp:native CreateMenu borrowed
//nolint:revive // unused-parameter: the handler is a name the emitter writes
func CreateMenu(handler func(menu Menu, action MenuChoice, param1 int32, param2 int32) int32) Menu {
	if menus.CreateMenu == nil {
		missing("CreateMenu")
	}
	return menus.CreateMenu("")
}

// SetMenuTitle writes the line above the items.
//
//sp:native SetMenuTitle
func SetMenuTitle(m Menu, title string) {
	if menus.SetMenuTitle == nil {
		missing("SetMenuTitle")
	}
	menus.SetMenuTitle(m, title)
}

// AddMenuItem adds one row.
//
//sp:native AddMenuItem
func AddMenuItem(m Menu, info string, display string) {
	if menus.AddMenuItem == nil {
		missing("AddMenuItem")
	}
	menus.AddMenuItem(m, info, display)
}

// AddMenuItemText is the same row where the label was built rather than
// written out, which a toggle needs.
//
//sp:native AddMenuItem
func AddMenuItemText(m Menu, info string, display string) {
	if menus.AddMenuItemText == nil {
		missing("AddMenuItem")
	}
	menus.AddMenuItemText(m, info, display)
}

// SetMenuExitBackButton puts a back button on it, which is what makes a
// submenu a submenu.
//
//sp:native SetMenuExitBackButton
func SetMenuExitBackButton(m Menu, on bool) {
	if menus.SetMenuExitBackButton == nil {
		missing("SetMenuExitBackButton")
	}
	menus.SetMenuExitBackButton(m, on)
}

// DisplayMenu shows it.
//
//sp:native DisplayMenu
func DisplayMenu(m Menu, client int32, time int32) {
	if menus.DisplayMenu == nil {
		missing("DisplayMenu")
	}
	menus.DisplayMenu(m, client, time)
}

// DisplayMenuAtItem shows it scrolled to a row, so a menu redrawn after a
// toggle keeps its place.
//
//sp:native DisplayMenuAtItem
func DisplayMenuAtItem(m Menu, client int32, item int32, time int32) {
	if menus.DisplayMenuAtItem == nil {
		missing("DisplayMenuAtItem")
	}
	menus.DisplayMenuAtItem(m, client, item, time)
}

// MenuTimeForever is MENU_TIME_FOREVER: up until it is answered.
//
//sp:global MENU_TIME_FOREVER
func MenuTimeForever() int32 { return 0 }

// MenuCancelExitBack is MenuCancel_ExitBack, the back button rather than a
// close.
//
//sp:global MenuCancel_ExitBack
func MenuCancelExitBack() int32 { return -6 }

// ClassPreferencesFlags is what classes this player will accept as bots.
// Ported, playerpref.
//
//sp:body GetClassPreferencesFlags
func ClassPreferencesFlags(client int32) int32 {
	if menus.ClassPreferencesFlags == nil {
		missing("GetClassPreferencesFlags")
	}
	return menus.ClassPreferencesFlags(client)
}

// SetClassPreferences writes one class's answer. Still in player_pref.sp.
//
//sp:body SetClassPreferences
func SetClassPreferences(client int32, class string, value int32) {
	if menus.SetClassPreferences == nil {
		missing("SetClassPreferences")
	}
	menus.SetClassPreferences(client, class, value)
}

// MenuVoteEnd is MenuAction_VoteEnd, the vote finished and param1 is the item
// that won.
//
//sp:global MenuAction_VoteEnd
func MenuVoteEnd() MenuChoice { return 1024 }

// MenuVoteCancel is MenuAction_VoteCancel, nobody answered or it was stopped.
//
//sp:global MenuAction_VoteCancel
func MenuVoteCancel() MenuChoice { return 2048 }

// SetBotSummoner writes g_iUIDBotSummoner, the userid of whoever called the
// vote, kept so the bots can be taken away again by the same player.
//
//sp:globalset g_iUIDBotSummoner
func SetBotSummoner(userid int32) {
	if menus.SetBotSummoner == nil {
		missing("g_iUIDBotSummoner")
	}
	menus.SetBotSummoner(userid)
}

// ManageDefenderBotsOn turns the manager on or off. Ported, manage.
//
//sp:body ManageDefenderBots
func ManageDefenderBotsOn(manage bool) {
	if menus.ManageDefenderBots == nil {
		missing("ManageDefenderBots")
	}
	menus.ManageDefenderBots(manage)
}

// WeaponPrefMenuItemText is one row of the weapon menu: the slot's name and
// what this player has picked for it. Still in menu.sp.
//
//sp:plugin GetWeaponPrefMenuItemText returns
func WeaponPrefMenuItemText(client int32, class string, slot int32) Text {
	if menus.WeaponPrefMenuItemText == nil {
		missing("GetWeaponPrefMenuItemText")
	}
	return menus.WeaponPrefMenuItemText(client, class, slot)
}

// WeaponSlotItem1 is TFWeaponSlot_Item1, where the spy keeps his watch.
//
//sp:global TFWeaponSlot_Item1
func WeaponSlotItem1() int32 { return 4 }

// SetMenuTitleFor writes the title from a format and one substitution, which is
// what a menu naming the class it is about needs.
//
//sp:native SetMenuTitle
func SetMenuTitleFor(m Menu, format string, class string) {
	if menus.SetMenuTitleFor == nil {
		missing("SetMenuTitle")
	}
	menus.SetMenuTitleFor(m, format, class)
}

// CopyShort is strcopy into a buffer shorter than a Text, which is what a
// per-client class name is.
//
//sp:native strcopy fills
func CopyShort(from string) (out [16]byte) {
	if menus.CopyShort == nil {
		missing("strcopy")
	}
	return menus.CopyShort(from)
}

// AddMenuItemFrom is AddMenuItem where the label came out of a buffer rather
// than being written out.
//
//sp:native AddMenuItem
func AddMenuItemFrom(m Menu, info string, display Text) {
	if menus.AddMenuItemFrom == nil {
		missing("AddMenuItem")
	}
	menus.AddMenuItemFrom(m, info, display)
}

// ShowWeaponPreferenceItemListMenu lists the items for one slot of one class.
// Still in menu.sp.
//
//sp:plugin ShowWeaponPreferenceItemListMenu
func ShowWeaponPreferenceItemListMenu(client int32, class [16]byte, slot string) {
	if menus.ShowWeaponItemList == nil {
		missing("ShowWeaponPreferenceItemListMenu")
	}
	menus.ShowWeaponItemList(client, class, slot)
}

// AddDefenderTFBotNamed is AddDefenderTFBot called the way the menus call it:
// one bot, a class written out, and the two defaults left off. Ported, manage.
//
//sp:body AddDefenderTFBot
func AddDefenderTFBotNamed(count int32, class string, team string, difficulty string) {
	if menus.AddDefenderTFBotNamed == nil {
		missing("AddDefenderTFBot")
	}
	menus.AddDefenderTFBotNamed(count, class, team, difficulty)
}
