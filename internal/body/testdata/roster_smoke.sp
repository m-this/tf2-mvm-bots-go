/* The generated DHook callbacks, compiled against the real DHooks include.

   They cannot run here: DHookParam and DHookReturn are handles the engine hands
   the plugin, and nothing outside a game server makes one. Compiling is the
   check, and it is the one that bites: an argument read as the wrong type, a
   supercede that answers a value the return does not take and a callback whose
   shape DHooks will not accept are all errors here.

   Hand written, like the probe. The plugin state the bodies reach through the
   extern package is stubbed, because what is being compiled is the generated
   file and not the plugin. */
#include <sourcemod>
#include <sdktools>
#include <dhooks>

Handle m_hHasAmmo;
Handle m_hClip1;

stock bool IsDefenderBot(int client) { return client > 0; }
stock void SetTouchCredits(bool touching) { }

#include "roster.sp"
#include "roster_dhooks.sp"

public void OnPluginStart()
{
	PrintToServer("%d", Go_AliveOnTeam(MaxClients, 2));
	PrintToServer("%d", Go_LoadedRounds(0));
}
