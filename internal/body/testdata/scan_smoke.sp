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
stock TFTeam GetPlayerEnemyTeam(int client) { return TF2_GetClientTeam(client) == TFTeam_Red ? TFTeam_Blue : TFTeam_Red; }

/* stocklib's, which is a vendored include and not on the compiler's path here. */
stock bool TF2_IsMiniBoss(int client) { return client == 5; }
stock bool TF2_IsInvulnerable(int client) { return client == 6; }
stock bool TF2_IsStealthed(int client) { return client == 7; }

stock float[] WorldSpaceCenter(int entity)
{
	float centre[3];
	GetClientAbsOrigin(entity, centre);
	centre[2] += 41.0;
	return centre;
}

#include "scan.sp"

public void OnPluginStart()
{
	// The three argument form is what util.sp's callers write.
	PrintToServer("%d", Go_NearestEnemyCount(1, 400.0));
	PrintToServer("%d", Go_NearestEnemyCount(1, 400.0, true));
	// The two argument form, which is how util.sp's callers write it.
	PrintToServer("%d", Go_EnemyNearestToMe(1, 400.0));
	PrintToServer("%d", Go_EnemyNearestToMe(1, 400.0, true, true, true, TFClass_Sniper));
}
