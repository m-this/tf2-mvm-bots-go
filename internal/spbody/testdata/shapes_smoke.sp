/* Compiles the emitted shapes body. Hand written, and it does nothing but call
   each generated function once so nothing is dropped as unused. */
#include "shapes.sp"

native void printnum(int n);

public void main()
{
	Go_Sample s;
	s.Client = 1;
	s.Score = 2.5;

	printnum(view_as<int>(Go_Rank(s, 1.0)));

	float average;
	printnum(Go_SumRecent(s, average));
	printnum(Go_Clamp(9, 0, 4));
	printnum(Go_Clamp(9));
	printnum(view_as<int>(Go_Note(1, Go_PriorityBusy)));
	printnum(view_as<int>(Go_Offset(s)));
	printnum(view_as<int>(Go_Reach(s)));
}
