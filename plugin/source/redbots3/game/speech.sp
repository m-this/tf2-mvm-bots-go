/* Making a bot say something.

The SpeakResponseConcept input takes a response-rules concept token, and the
game decides from the rules whether this player says anything at all. The tokens
are TF2's own, out of scripts/talker. The plugin passes the constants below
rather than the strings, so a typo is a compile error. */

enum
{
	MP_CONCEPT_PLAYER_MEDIC = 0,
	MP_CONCEPT_PLAYER_HELP,
	MP_CONCEPT_PLAYER_INCOMING,
	MP_CONCEPT_PLAYER_CLOAKEDSPY,
	MP_CONCEPT_PLAYER_DISPENSERHERE,
	MP_CONCEPT_PLAYER_SENTRYHERE,
	MP_CONCEPT_PLAYER_JEERS
};

stock bool BaseMultiplayerPlayer_SpeakConceptIfAllowed(int client, int concept)
{
	char token[32];

	switch (concept)
	{
		case MP_CONCEPT_PLAYER_MEDIC: token = "TLK_PLAYER_MEDIC";
		case MP_CONCEPT_PLAYER_HELP: token = "TLK_PLAYER_HELP";
		case MP_CONCEPT_PLAYER_INCOMING: token = "TLK_PLAYER_INCOMING";
		case MP_CONCEPT_PLAYER_CLOAKEDSPY: token = "TLK_PLAYER_CLOAKEDSPY";
		case MP_CONCEPT_PLAYER_DISPENSERHERE: token = "TLK_PLAYER_DISPENSERHERE";
		case MP_CONCEPT_PLAYER_SENTRYHERE: token = "TLK_PLAYER_SENTRYHERE";
		case MP_CONCEPT_PLAYER_JEERS: token = "TLK_PLAYER_JEERS";
		default: return false;
	}

	SetVariantString(token);

	return AcceptEntityInput(client, "SpeakResponseConcept");
}
