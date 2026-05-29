package internals

import (
	"math"
	"math/rand"
)

// ApplyRandomPercentageChange randomly adds or subtracts a given percentage from the input value.
// The percentage should be provided as a decimal (e.g., 0.05 for 5%).
func ApplyRandomPercentageChange(value float64, percentage float64) float64 {
	amountToChange := value * percentage

	// Generate a random boolean (true for add, false for subtract)
	if rand.Intn(2) == 0 { // rand.Intn(2) returns 0 or 1
		// Subtract the calculated amount
		return value - amountToChange
	} else {
		// Add the calculated amount
		return value + amountToChange
	}
}

// ApplyRandomPercentageChangeToInt randomly adds or subtracts a given percentage
// from the input value, where the percentage is applied to the
// absolute integer part of the value.
// The percentage should be provided as a decimal (e.g., 0.05 for 5%).
func ApplyRandomPercentageChangeToInt(value float64, percentage float64) float64 {
	// Get the absolute integer part of the value
	// math.Abs ensures it works correctly for negative numbers
	// math.Trunc removes the fractional part
	integerPart := math.Trunc(math.Abs(value))

	// Calculate the amount to change based on the integer part
	amountToChange := float64(int(integerPart * percentage))

	// Generate a random boolean (true for add, false for subtract)
	if rand.Intn(2) == 0 { // rand.Intn(2) returns 0 or 1
		// Subtract the calculated amount
		return value - amountToChange
	} else {
		// Add the calculated amount
		return value + amountToChange
	}
}

func ToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}
