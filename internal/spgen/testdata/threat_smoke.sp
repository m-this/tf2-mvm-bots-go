/* The threat priority sweep: every combination, through the generated code.

   Hand written on purpose, like the action selection sweep. It is the control
   for the generator, so nothing in it is generated: the record is unpacked here
   rather than by anything the same Go wrote.

   The ranges come in as float32 bits so the two comparisons in ThreatBand are
   made on the exact distances the Go compared, and not on a printed decimal. */
#include <smoke_env>
#include <probe_ranges>
#include "threat_priority.sp"

native void printnum(int n);

public void main()
{
	for (int r = 0; r < sizeof(gProbeRanges); r++)
	{
		float rangeSq = view_as<float>(gProbeRanges[r]);
		
		for (int isPlayer = 0; isPlayer < 2; isPlayer++)
		for (int inGame = 0; inGame < 2; inGame++)
		for (int pclass = 0; pclass < THREAT_CLASSES; pclass++)
		for (int giant = 0; giant < 2; giant++)
		for (int carrier = 0; carrier < 2; carrier++)
		{
			printnum(ThreatPriorityOf(rangeSq, isPlayer != 0, inGame != 0,
				view_as<TFClassType>(pclass), giant != 0, carrier != 0));
		}
	}
}
