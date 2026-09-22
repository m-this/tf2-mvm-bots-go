// External plugins may give an individual defender a short lived rally order.
// The bot mod owns pathing; the caller owns when and where the order applies.
//
// An order lapses on its own so a caller that stops talking, or a plugin that
// unloads mid-wave, cannot leave a bot walking to a point nobody still wants.
// The caller is expected to republish well inside this window: tf2-archipelago
// does it every two seconds. Widening the gap past this lifetime makes orders
// flicker rather than fail, which is the harder thing to diagnose, so the two
// numbers belong in the same sentence.
#define DirectiveLifetime 5.0

static bool g_DirectiveRally[MAXPLAYERS + 1];
static bool g_DirectiveSeek[MAXPLAYERS + 1];
static bool g_DirectiveBuyAnywhere[MAXPLAYERS + 1];
static float g_DirectivePoint[MAXPLAYERS + 1][3];
static float g_DirectiveUntil[MAXPLAYERS + 1];
// A client index is reused as soon as its occupant leaves. The userid is not,
// so an order outlives the bot it was given to only until the next one checks.
static int g_DirectiveUserId[MAXPLAYERS + 1];

void RegisterDirectiveNatives()
{
    CreateNative("Defenderbots_SetDirective", Native_SetDefenderDirective);
}

public any Native_SetDefenderDirective(Handle plugin, int numParams)
{
    int client = GetNativeCell(1);
    if (client < 1 || client > MaxClients || !IsClientInGame(client) || !IsFakeClient(client))
    {
        return false;
    }
    g_DirectiveRally[client] = view_as<bool>(GetNativeCell(2));
    GetNativeArray(3, g_DirectivePoint[client], 3);
    g_DirectiveSeek[client] = view_as<bool>(GetNativeCell(4));
    g_DirectiveBuyAnywhere[client] = view_as<bool>(GetNativeCell(5));
    g_DirectiveUserId[client] = GetClientUserId(client);
    g_DirectiveUntil[client] = GetEngineTime() + DirectiveLifetime;
    return true;
}

stock bool DefenderDirectiveCurrent(int client)
{
    return client > 0 && client <= MaxClients
        && g_DirectiveUntil[client] > GetEngineTime()
        && IsClientInGame(client)
        && GetClientUserId(client) == g_DirectiveUserId[client];
}

stock bool DefenderRallyActive(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveRally[client];
}

// The three readers below gate on the same freshness as every other accessor.
// They used to hand back the last point written however long ago it lapsed,
// which was only ever safe because each caller happened to ask RallyActive
// first.
stock float DefenderRallyX(int client)
{
    return DefenderRallyActive(client) ? g_DirectivePoint[client][0] : 0.0;
}

stock float DefenderRallyY(int client)
{
    return DefenderRallyActive(client) ? g_DirectivePoint[client][1] : 0.0;
}

stock float DefenderRallyZ(int client)
{
    return DefenderRallyActive(client) ? g_DirectivePoint[client][2] : 0.0;
}

stock bool DefenderSeekEnemies(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveSeek[client];
}

stock bool DefenderBuyAnywhere(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveBuyAnywhere[client];
}
