/* A compile-and-run check on the generated attribute lookup.

   Unlike the edge, this one runs: AttributeID is a string comparison and
   nothing else, so spshell can answer with it. Each name the Go table declares
   goes in, one id comes out, and the Go side asserts it is the id it declared.

   The names arrive in probe_names, written by the test, rather than being read
   out of the generated array: that array is static, which is right for the
   plugin because it is AttributeID's own business and not a second public name
   table. */
#include <smoke_env>
#include <probe_names>
#include "attributes.sp"

native void printnum(int n);

public void main()
{
	for (int i = 0; i < sizeof(gProbeNames); i++)
		printnum(AttributeID(gProbeNames[i]));
}
