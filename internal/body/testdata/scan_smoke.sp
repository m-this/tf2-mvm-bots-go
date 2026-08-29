/* The ported scan, compiled against the compiler and the includes the plugin
   ships with. What it catches is the part the standalone SourcePawn cannot: the
   real GetVectorDistance, the real GetClientAbsOrigin, and the default the
   plugin's own call sites rely on.

   The plugin functions the port has not reached yet are declared here, and so
   are the ones from the vendored stocklib include, because what is being
   compiled is the generated file and not the plugin. */
#include <sourcemod>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>

stock bool IsSentryBusterRobot(int client) { return client == 1; }
stock bool IsCloakedPlayerExposed(int client) { return client == 4; }


/* stocklib's, which is a vendored include and not on the compiler's path here. */
stock bool TF2_IsMiniBoss(int client) { return client == 5; }
stock TFTeam TF2_GetEnemyTeam(TFTeam team) { return team == TFTeam_Red ? TFTeam_Blue : TFTeam_Red; }
stock bool TF2_IsInvulnerable(int client) { return client == 6; }
stock bool TF2_IsStealthed(int client) { return client == 7; }
stock bool TF2_IsPlacing(int entity) { return entity == 8; }
stock bool TF2_IsCarried(int entity) { return entity == 9; }
stock bool TF2_HasSapper(int entity) { return entity == 10; }
stock int BaseEntity_GetTeamNumber(int entity) { return GetEntProp(entity, Prop_Send, "m_iTeamNum"); }
stock bool BaseEntity_IsPlayer(int entity) { return entity > 0 && entity <= MaxClients; }
stock int TF2_GetNumHealers(int client) { return 0; }
stock int TF2Util_GetPlayerHealer(int client, int index) { return -1; }
stock int BaseCombatCharacter_GetActiveWeapon(int client) { return GetEntPropEnt(client, Prop_Send, "m_hActiveWeapon"); }
/* tf2utils' own. The include is not on the path here; the constant it answers
   with comes from tf2_stocks, which is. */
enum TFWeaponType
{
	TFWeaponTypeUnused = -1
};

stock TFWeaponType TF2Util_GetWeaponID(int weapon) { return view_as<TFWeaponType>(TF_WEAPON_MEDIGUN); }

/* stocklib's, which is a vendored include and not on the path here. Both fill
   the array they are given; the float[] forms the plugin used to have are now
   generated. */
stock void BaseEntity_GetAbsOrigin(int entity, float origin[3])
{
	GetEntPropVector(entity, Prop_Send, "m_vecOrigin", origin);
}

stock void BaseEntity_WorldSpaceCenter(int entity, float centre[3])
{
	GetClientAbsOrigin(entity, centre);
	centre[2] += 41.0;
}

#include "scan.sp"

public void OnPluginStart()
{
	// The three argument form is what util.sp's callers write.
	PrintToServer("%d", GetNearestEnemyCount(1, 400.0));
	PrintToServer("%d", GetNearestEnemyCount(1, 400.0, true));
	// The two argument form, which is how util.sp's callers write it.
	PrintToServer("%d", FindEnemyNearestToMe(1, 400.0));
	PrintToServer("%d", FindEnemyNearestToMe(1, 400.0, true, true, true, TFClass_Sniper));
	// Both building scans with the default range their callers omit.
	PrintToServer("%d", GetNearestSappableObject(1));
	PrintToServer("%d", GetNearestEnemyTeleporter(1));
	PrintToServer("%d", GetBestTargetForSpy(1, 400.0));
	// The spy's four, with the defaults their callers omit.
	PrintToServer("%d", GetNearestSappablePlayer(1, 400.0));
	PrintToServer("%d", GetFarthestSappablePlayer(1, 400.0, true));
	PrintToServer("%d", GetNearestSappablePlayerHealingSomeone(1, 400.0, false, TFClass_Medic, 300.0));

	float here[3];
	PrintToServer("%d", GetEnemyPlayerNearestToPosition(1, here, 400.0));
}
