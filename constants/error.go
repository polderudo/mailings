package constants

type ErrorCode int

const (
	ErrGeneral ErrorCode = iota
	ErrValidation
	ErrUserDeactivated
	ErrUserNoPassword
	ErrUserInvalidPassword
	ErrMissingInput
	ErrWrongInput
	ErrTokenNotExists
	ErrToShort
	errorCodeEnd // Internal marker for ranging
)

// String returns the name of the constant as a string
func (e ErrorCode) String() string {
	names := [...]string{
		"ErrGeneral",
		"ErrValidation",
		"ErrUserDeactivated",
		"ErrUserNoPassword",
		"ErrUserInvalidPassword",
		"ErrMissingInput",
		"ErrWrongInput",
		"ErrTokenNotExists",
		"ErrToShort",
	}

	if len(names) < int(errorCodeEnd) {
		panic("constants.ErrorCode: names array is too short")
	}

	if e < 0 || int(e) >= len(names) {
		return "UnknownErrorCode"
	}
	return names[e]
}

// AllErrorCodes returns a slice of all defined error codes for iteration
func AllErrorCodes() []ErrorCode {
	var codes []ErrorCode
	for i := ErrorCode(0); i < errorCodeEnd; i++ {
		codes = append(codes, i)
	}
	return codes
}

// Map returns a map of constant names to their associated integer values
// Use this to generate JSON or export to frontend files
func Map() map[string]int {
	m := make(map[string]int)
	for _, code := range AllErrorCodes() {
		m[code.String()] = int(code)
	}
	return m
}
