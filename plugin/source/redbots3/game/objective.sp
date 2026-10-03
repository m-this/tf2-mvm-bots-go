/* The mission and the map, read off tf_objective_resource and the game rules.

The wave counts and the popfile name are the manager's own properties. The
control-point part is the linear-capture rule: TF2 refuses a point whose
previous point the team does not hold yet, and a sapper bot has to ask the same
question the game asks before it walks to a point it cannot take. */

stock bool TF2_IsMannVsMachineMode()
{
	return view_as<bool>(GameRules_GetProp("m_bPlayingMannVsMachine"));
}

stock bool TF2_IsInKothMode()
{
	return view_as<bool>(GameRules_GetProp("m_bPlayingKoth"));
}

stock bool TF2_IsInArenaMode()
{
	return GameRules_GetProp("m_nGameType") == TF_GAMETYPE_ARENA;
}

stock bool TeamplayRoundBasedRules_IsInWaitingForPlayers()
{
	return view_as<bool>(GameRules_GetProp("m_bInWaitingForPlayers"));
}

stock int TF2_GetMannVsMachineWaveCount(int resource)
{
	return GetEntProp(resource, Prop_Send, "m_nMannVsMachineWaveCount");
}

stock int TF2_GetMannVsMachineMaxWaveCount(int resource)
{
	return GetEntProp(resource, Prop_Send, "m_nMannVsMachineMaxWaveCount");
}

stock void TF2_GetMannVsMachineWaveClassName(int resource, int index, char[] buffer, int maxlen)
{
	GetEntPropString(resource, Prop_Data, "m_iszMannVsMachineWaveClassNames", buffer, maxlen, index);
}

stock void TF2_GetMvMPopfileName(int resource, char[] buffer, int maxlen)
{
	GetEntPropString(resource, Prop_Send, "m_iszMvMPopfileName", buffer, maxlen);
}

stock bool CaptureFlag_IsHome(int entity)
{
	return GetEntProp(entity, Prop_Send, "m_nFlagStatus") == TF_FLAGINFO_HOME;
}

/* m_iPreviousPoints is one flat array of MAX_PREVIOUS_POINTS entries per point
per team, so the index has to be built rather than passed. */
stock int ObjectiveResource_GetPreviousPointForPoint(int resource, int index, int team, int previous)
{
	int flat = previous + (index * MAX_PREVIOUS_POINTS) + (team * MAX_CONTROL_POINTS * MAX_PREVIOUS_POINTS);
	return GetEntProp(resource, Prop_Send, "m_iPreviousPoints", _, flat);
}

stock int ObjectiveResource_GetNumControlPoints(int resource)
{
	return GetEntProp(resource, Prop_Send, "m_iNumControlPoints");
}

stock int ObjectiveResource_GetOwningTeam(int resource, int index)
{
	if (index >= ObjectiveResource_GetNumControlPoints(resource))
	{
		return 0;
	}

	return GetEntProp(resource, Prop_Send, "m_iOwner", _, index);
}

stock int ObjectiveResource_GetBaseControlPointForTeam(int resource, int team)
{
	return GetEntProp(resource, Prop_Send, "m_iBaseControlPoints", _, team);
}

stock bool ObjectiveResource_GetCPLocked(int resource, int index)
{
	return view_as<bool>(GetEntProp(resource, Prop_Send, "m_bCPLocked", _, index));
}

stock bool ObjectiveResource_PlayingMiniRounds(int resource)
{
	return view_as<bool>(GetEntProp(resource, Prop_Send, "m_bPlayingMiniRounds"));
}

/* The last point of the team's own run of points, walking outward from its base
until a point somebody else owns. */
stock int TF2_GetFarthestOwnedControlPoint(TFTeam team)
{
	int resource = FindEntityByClassname(-1, "tf_objective_resource");

	int ownedEnd = ObjectiveResource_GetBaseControlPointForTeam(resource, view_as<int>(team));

	if (ownedEnd == -1)
	{
		return -1;
	}

	int walk = 1;
	int enemyEnd = ObjectiveResource_GetNumControlPoints(resource) - 1;

	if (ownedEnd != 0)
	{
		walk = -1;
		enemyEnd = 0;
	}

	int farthest = ownedEnd;

	for (int point = ownedEnd; point != enemyEnd; point += walk)
	{
		if (ObjectiveResource_GetOwningTeam(resource, point) != view_as<int>(team))
		{
			break;
		}

		farthest = point;
	}

	return farthest;
}

/* Whether the team is allowed to take this point yet.

Without tf_caplinear every point is open. With it, a point names the points that
have to fall first, and a point that names itself is the start of the run. */
stock bool TFGameRules_TeamMayCapturePoint(TFTeam team, int pointIndex)
{
	if (!FindConVar("tf_caplinear").BoolValue)
	{
		return true;
	}

	int resource = FindEntityByClassname(-1, "tf_objective_resource");
	int needed = ObjectiveResource_GetPreviousPointForPoint(resource, pointIndex, view_as<int>(team), 0);

	if (needed == pointIndex)
	{
		return true;
	}

	if (TF2_IsInKothMode() && TeamplayRoundBasedRules_IsInWaitingForPlayers())
	{
		return false;
	}

	if (ObjectiveResource_GetCPLocked(resource, pointIndex))
	{
		return false;
	}

	if (needed == -1)
	{
		if (TF2_IsInArenaMode())
		{
			return GameRules_GetPropFloat("m_flCapturePointEnableTime") <= GetGameTime()
				&& GameRules_GetRoundState() == RoundState_Stalemate;
		}

		if (ObjectiveResource_PlayingMiniRounds(resource))
		{
			return true;
		}

		return TF2_GetFarthestOwnedControlPoint(team) - pointIndex <= 1;
	}

	for (int previous = 0; previous < MAX_PREVIOUS_POINTS; previous++)
	{
		needed = ObjectiveResource_GetPreviousPointForPoint(resource, pointIndex, view_as<int>(team), previous);

		if (needed == -1)
		{
			continue;
		}

		if (ObjectiveResource_GetOwningTeam(resource, needed) != view_as<int>(team))
		{
			return false;
		}
	}

	return true;
}
