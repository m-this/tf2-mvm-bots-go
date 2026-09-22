// External plugins may give an individual defender a short lived rally order.
// The bot mod owns pathing; the caller owns when and where the order applies.
static bool g_DirectiveRally[MAXPLAYERS + 1];
static bool g_DirectiveSeek[MAXPLAYERS + 1];
static bool g_DirectiveBuyAnywhere[MAXPLAYERS + 1];
static float g_DirectivePoint[MAXPLAYERS + 1][3];
static float g_DirectiveUntil[MAXPLAYERS + 1];

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
    g_DirectiveUntil[client] = GetEngineTime() + 5.0;
    return true;
}

stock bool DefenderDirectiveCurrent(int client)
{
    return client > 0 && client <= MaxClients && g_DirectiveUntil[client] > GetEngineTime();
}

stock bool DefenderRallyActive(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveRally[client];
}

stock float DefenderRallyX(int client)
{
    return g_DirectivePoint[client][0];
}

stock float DefenderRallyY(int client)
{
    return g_DirectivePoint[client][1];
}

stock float DefenderRallyZ(int client)
{
    return g_DirectivePoint[client][2];
}

stock bool DefenderSeekEnemies(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveSeek[client];
}

stock bool DefenderBuyAnywhere(int client)
{
    return DefenderDirectiveCurrent(client) && g_DirectiveBuyAnywhere[client];
}
