package internals

func AllSame[T comparable](a []T) bool {
	for _, v := range a {
		if v != a[0] {
			return false
		}
	}
	return true
}

func GetValueOrDefault[T comparable](val1, val2 T) T {
	var zero T // This creates the zero value for type T (nil for pointers)
	if val1 == zero {
		return val2
	}
	return val1
}
