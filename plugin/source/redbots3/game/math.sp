/* The scalar and vector questions SourceMod does not answer.

ClampFloat is stocksoup's too, down to the signature, but <stocksoup/math> also
declares NormalizeAngle and this plugin generates one of its own. Five lines
here cost less than renaming a generated function to borrow them. */

stock float MinFloat(float a, float b)
{
	return a < b ? a : b;
}

stock float MaxFloat(float a, float b)
{
	return a > b ? a : b;
}

stock float ClampFloat(float value, float min, float max)
{
	if (value < min)
	{
		return min;
	}

	if (value > max)
	{
		return max;
	}

	return value;
}

stock bool Vector_IsZero(const float vec[3], float tolerance = 0.01)
{
	return vec[0] > -tolerance && vec[0] < tolerance
		&& vec[1] > -tolerance && vec[1] < tolerance
		&& vec[2] > -tolerance && vec[2] < tolerance;
}
