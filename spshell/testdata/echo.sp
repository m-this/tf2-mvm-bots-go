/* The float literal control: every value in gInputs printed straight back as
   its bits.

   Nothing is computed, so a difference is the compiler reading the literal
   differently from Go writing it, which is the only thing under test here. */
#include <golden_inputs>

native void printnum(int n);

public void main()
{
	for (int i = 0; i < sizeof(gInputs); i++) {
		printnum(view_as<int>(gInputs[i][0]));
		printnum(view_as<int>(gInputs[i][1]));
		printnum(view_as<int>(gInputs[i][2]));
	}
}
