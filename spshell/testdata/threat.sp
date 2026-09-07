#include <golden_inputs>

// Hand written, deliberately: this file is the spike's control. No sourcemod.inc,
// no engine natives. printnum is spshell's own builtin, and the score goes out as
// raw IEEE-754 bits so the comparison against Go is exact, not printf rounded.
native void printnum(int n);

float ThreatScore(float distance, float health, float classID)
{
	float range = 1.0 - distance / 2048.0;
	if (range < 0.0)
		range = 0.0;

	float hurt = 1.0 - health / 300.0;
	if (hurt < 0.0)
		hurt = 0.0;

	return range * 60.0 + hurt * 30.0 + classID * 1.5;
}

public void main()
{
	for (int i = 0; i < sizeof(gInputs); i++) {
		float score = ThreatScore(gInputs[i][0], gInputs[i][1], gInputs[i][2]);
		printnum(view_as<int>(score));
	}
}
