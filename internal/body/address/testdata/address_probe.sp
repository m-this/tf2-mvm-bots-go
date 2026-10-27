/* The differential probe for the address crossing. Hand written: it is the
   control. address.sp is the generated file, included by name.

   Output, per case: whether ThrowError was reached, then the low and high
   halves of the result. */
typedef Address = int64;

native void printnum(int n);

int gThrown;

stock void ThrowError(const char[] format, any ...)
{
	gThrown = 1;
}

#include "address.sp"

static void EmitAddress(Address address)
{
	printnum(gThrown);
	printnum(view_as<int>(address));
	printnum(view_as<int>(address >> 32));
	gThrown = 0;
}

static void EmitCell(int cell)
{
	printnum(gThrown);
	printnum(cell);
	gThrown = 0;
}

public void main()
{
	int cells[5] = { 0, 1, -1, 0x7FFFFFFF, 0x80001000 };
	for (int i = 0; i < sizeof(cells); i++)
	{
		EmitAddress(AddressOfCell(cells[i]));
		EmitCell(CellOfAddress(AddressOfCell(cells[i])));
		EmitCell(CellOfAddress(view_as<Address>(cells[i])));
	}

	EmitCell(CellOfAddress(view_as<Address>(1) << 32));
	EmitCell(CellOfAddress((view_as<Address>(-1) << 32) | view_as<Address>(5)));
	EmitCell(CellOfAddress((view_as<Address>(2) << 32) | view_as<Address>(0x80001000) & ADDRESS_CELL_MASK));
}
