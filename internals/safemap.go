package internals

import (
	"sync"
)

/*
usage:
	// String to int map

	intMap := NewSafeMap[string, int]()
	intMap.Set("count", 42)

	if value, ok := intMap.Get("count"); ok {
	    slog.Info("Count: %d\n", value)
	}

	// String to component pointer map (like in your code)
	cvMap := NewSafeMap[string, *components.Conveyor]()
	// cvMap.Set("CV001", someConveyorInstance)

	// String to slice map
	hutMap := NewSafeMap[string, []string]()
	hutMap.Set("equipment1", []string{"hut1", "hut2", "hut3"})

	// Iterate over all entries
	hutMap.Range(func(equipID string, huts []string) bool {
	    slog.Info("Equipment %s has huts: %v\n", equipID, huts)
	    return true // continue iteration
	})

	// Get all keys
	keys := intMap.Keys()
	slog.Info("All keys: %v\n", keys)

	// Check length
	slog.Info("Map size: %d\n", intMap.Len())

	// Load or store
	actual, wasLoaded := intMap.LoadOrStore("newKey", 100)
	slog.Info("Value: %d, Was loaded: %t\n", actual, wasLoaded)
*/

type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
}

func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.data[key]
	return val, ok
}

func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.data, key)
}

func (sm *SafeMap[K, V]) Has(key K) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.data[key]
	return exists
}

func (sm *SafeMap[K, V]) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.data)
}

func (sm *SafeMap[K, V]) Keys() []K {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	keys := make([]K, 0, len(sm.data))
	for k := range sm.data {
		keys = append(keys, k)
	}
	return keys
}

func (sm *SafeMap[K, V]) Values() []V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	values := make([]V, 0, len(sm.data))
	for _, v := range sm.data {
		values = append(values, v)
	}
	return values
}

func (sm *SafeMap[K, V]) Range(fn func(key K, value V) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for k, v := range sm.data {
		if !fn(k, v) {
			break
		}
	}
}

func (sm *SafeMap[K, V]) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data = make(map[K]V)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded apiResult is true if the value was loaded, false if stored.
func (sm *SafeMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if actual, loaded = sm.data[key]; loaded {
		return actual, true
	}
	sm.data[key] = value
	return value, false
}
