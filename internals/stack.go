package internals

import (
	"fmt"
	"runtime"
)

func GetStackByLevel(level int) (string, error) {
	// We need to skip an additional frame because GetStackByLevel itself is on the stack.
	targetFrame := level + 1

	// Allocate a slice to hold the program counters.
	pcs := make([]uintptr, targetFrame+2) // Allocate enough space for desired level + extra frames
	n := runtime.Callers(targetFrame, pcs)
	if n == 0 {
		return "", fmt.Errorf("no stack trace information available")
	}

	frames := runtime.CallersFrames(pcs[:n])

	for more := true; more; level-- {
		frame, more := frames.Next()
		if level == 0 {
			if !more {
				return "", fmt.Errorf("level exceeds the available stack depth")
			}
			return fmt.Sprintf("%s:%d", frame.Function, frame.Line), nil
		}
	}
	return "", fmt.Errorf("invalid level: stack level does not exist")
}
