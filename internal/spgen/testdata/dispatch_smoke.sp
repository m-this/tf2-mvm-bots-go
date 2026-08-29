/* A compile-only smoke test for the generated edge.

   The edge is the one generated file that calls into the engine, so it cannot
   be compiled against the real plugin here and it cannot be run under spshell
   at all. What it can do is compile against stubs with the same shapes, which
   is enough to catch a reserved word used as a parameter name, a typo in a
   behaviour or predicate name, and a switch over an outcome that is not in the
   enum.

   Everything below is a stand-in. Nothing here is generated and nothing here
   ships. */
#pragma semicolon 1
#pragma newdecls required

enum Action
{
	Plugin_Continue = 0,
	Plugin_Handled = 3
};

enum RoundState
{
	RoundState_Init = 0
};

enum TFClassType
{
	TFClass_Unknown = 0
};

#define INVALID_ACTION 0
#define MAXPLAYERS 65

methodmap BehaviorAction
{
	public BehaviorAction(int address) { return view_as<BehaviorAction>(address); }
	public Action SuspendFor(BehaviorAction next, const char[] reason)
	{
		if (next == view_as<BehaviorAction>(0) || reason[0] == 0)
			return Plugin_Continue;

		return Plugin_Handled;
	}
}

methodmap ConVar
{
	property bool BoolValue
	{
		public get() { return true; }
	}
}

methodmap ActionsManagerStub
{
	public int LookupEntityActionByName(int client, const char[] name)
	{
		return client + name[0];
	}
}

ActionsManagerStub ActionsManager;
ConVar redbots_manager_bot_use_upgrades;
bool g_bShoppedThisBreak[MAXPLAYERS + 1];
bool g_bHasUpgraded[MAXPLAYERS + 1];
int g_iBuyUpgradesNumber[MAXPLAYERS + 1];

stock RoundState GameRules_GetRoundState() { return RoundState_Init; }
stock TFClassType TF2_GetPlayerClass(int client) { return view_as<TFClassType>(client); }
stock void SetPlayerReady(int client, bool ready) { g_iBuyUpgradesNumber[client] = ready ? 1 : 0; }

stock bool CTFBotCollectMoney_IsPossible(int client) { return client > 0; }
stock bool TF2_IsInUpgradeZone(int client) { return client > 0; }
stock bool ShouldUpgradeMidRound(int client) { return client > 0; }
stock bool HasSniperRifle(int client) { return client > 0; }
stock bool IsSniperStalled(int client) { return client > 0; }
stock bool CTFBotDefenderAttack_SelectTarget(int client) { return client > 0; }
stock bool CTFBotAttackTank_SelectTarget(int client) { return client > 0; }
stock bool CTFBotMarkGiant_IsPossible(int client) { return client > 0; }
stock bool CTFBotCollectNearMoney_SelectTarget(int client) { return client > 0; }
stock bool CTFBotStickyTrap_IsPossible(int client) { return client > 0; }

stock BehaviorAction CTFBotCollectMoney() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotGotoUpgrade() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotMoveToFront() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotMarkGiant() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotAttackTank() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotDefenderAttack() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotMvMEngineerIdle() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotSpyLurkMvM() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotStickyTrap() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotCollectNearMoney() { return view_as<BehaviorAction>(1); }
stock BehaviorAction CTFBotGuardPoint() { return view_as<BehaviorAction>(1); }

#include "actionsel.sp"
#include "actionsel_dispatch.sp"

native void printnum(int n);

public void main()
{
	printnum(view_as<int>(ActionSel_GetDesiredBotAction(1, view_as<BehaviorAction>(1))));
}
