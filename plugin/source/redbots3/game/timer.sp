/* CCountdownTimer, read where the game keeps it.

The game's own timers are plain structs inside an action or an entity, so there
is no property to read: the two floats sit at a fixed offset from the struct's
address and are written through it. Starting one is how the plugin tells a game
behaviour to leave a bot alone for a while.

LoadFromAddress and StoreToAddress take a number type rather than a float type,
so the float goes through as its four bytes. */

#define COUNTDOWNTIMER_OFFSET_DURATION	0x4
#define COUNTDOWNTIMER_OFFSET_TIMESTAMP	0x8

methodmap CountdownTimer
{
	public CountdownTimer(Address pointer)
	{
		return view_as<CountdownTimer>(pointer);
	}

	property Address Address
	{
		public get() { return view_as<Address>(this); }
	}

	property float m_duration
	{
		public get()
		{
			return LoadFromAddress(this.Address + view_as<Address>(COUNTDOWNTIMER_OFFSET_DURATION), NumberType_Int32);
		}
		public set(float value)
		{
			StoreToAddress(this.Address + view_as<Address>(COUNTDOWNTIMER_OFFSET_DURATION), value, NumberType_Int32, false);
		}
	}

	property float m_timestamp
	{
		public get()
		{
			return LoadFromAddress(this.Address + view_as<Address>(COUNTDOWNTIMER_OFFSET_TIMESTAMP), NumberType_Int32);
		}
		public set(float value)
		{
			StoreToAddress(this.Address + view_as<Address>(COUNTDOWNTIMER_OFFSET_TIMESTAMP), value, NumberType_Int32, false);
		}
	}

	public void Start(float duration)
	{
		this.m_timestamp = GetGameTime() + duration;
		this.m_duration = duration;
	}
}
