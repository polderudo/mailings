package internals

import (
	"fmt"
	"math/rand"
)

// AddUnique adds a elemet to a slice if it does not exist yet
func AddUnique[T comparable](slice []T, element T) []T {
	for _, existing := range slice {
		if existing == element {
			return slice // Element already exists, return original slice
		}
	}
	return append(slice, element) // Add new element
}

// AddUniqueSlice adds elements from `toAdd` to the `existing` slice,
// ensuring that each element remains unique in the resulting slice.
func AddUniqueSlice[T comparable](existing []T, toAdd []T) []T {
	uniqueSet := make(map[T]bool) // A map to track the added elements for uniqueness.

	// Add all existing elements to the set.
	for _, value := range existing {
		uniqueSet[value] = true
	}

	// Add elements from `toAdd` only if they're not already in the set.
	for _, value := range toAdd {
		if !uniqueSet[value] {
			existing = append(existing, value)
			uniqueSet[value] = true
		}
	}

	return existing
}

// RemoveElementByPointerPreservingOrder removes the first occurrence of an element
// from a slice of pointers, identified by its memory address (pointer).
// It preserves the order of the remaining elements.
// 'slice' is the slice of pointers (e.g., []*EntryType).
// 'elementToRemove' is the specific pointer to the element to be removed (e.g., *EntryType).
func RemoveElementByPointerPreservingOrder[T any](slice []*T, elementToRemove *T) []*T {
	foundIndex := -1
	for i, ptr := range slice {
		// Compare memory addresses
		if ptr == elementToRemove {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		// Create a new slice by appending the parts before and after the found index
		return append(slice[:foundIndex], slice[foundIndex+1:]...)
	}

	// If the element wasn't found or the slice was empty, return the original slice
	return slice
}

// RemoveElementByPointerNoOrder removes the first occurrence of an element
// from a slice of pointers, identified by its memory address (pointer).
// This version is more efficient but does NOT preserve the order of remaining elements.
// 'slice' is the slice of pointers (e.g., []*EntryType).
// 'elementToRemove' is the specific pointer to the element to be removed (e.g., *EntryType).
func RemoveElementByPointerNoOrder[T any](slice []*T, elementToRemove *T) []*T {
	foundIndex := -1
	for i, ptr := range slice {
		// Compare memory addresses
		if ptr == elementToRemove {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		// Replace the element to be removed with the last element in the slice
		slice[foundIndex] = slice[len(slice)-1]
		// Nil out the last element in the original slice's backing array
		// to prevent memory leaks and help the garbage collector.
		slice[len(slice)-1] = nil
		// Return a new slice header that is one element shorter
		return slice[:len(slice)-1]
	}

	// If the element wasn't found or the slice was empty, return the original slice
	return slice
}

// GetRandomElement returns a random element from the provided slice.
// It uses generics, so it can work with a slice of any type T.
// If the slice is empty, it returns the zero value for type T and an error.
func GetRandomElement[T any](slice []T) (T, error) {
	// Get the length of the slice
	n := len(slice)

	// Handle the case of an empty slice
	if n == 0 {
		var zeroValue T // Get the zero value for the type T
		return zeroValue, fmt.Errorf("cannot get a random element from an empty slice")
	}

	// Generate a random index within the bounds of the slice [0, n-1]
	randomIndex := rand.Intn(n)

	// Return the element at the random index and no error
	return slice[randomIndex], nil
}
