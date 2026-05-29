package time_tracker

import (
	"fmt"
	"log"
	"time"
)

const TimeTrackesUnitMS = "ms"
const TimeTrackesUnitNS = "ns"
const TimeTrackesUnitMic = "mic"
const TimeTrackesUnitS = "s"
const TimeTrackesUnitM = "m"

type TimeTracker struct {
	StartTime    time.Time
	LastTime     time.Time
	LastTempTime *time.Time
	IsDisabled   bool
}

func NewTimeTracker() *TimeTracker {
	t := &TimeTracker{
		StartTime:  time.Now(),
		LastTime:   time.Now(),
		IsDisabled: false,
	}
	return t
}

func (t *TimeTracker) Start() {
	t.LastTime = time.Now()
}

func (t *TimeTracker) Reset() {
	t.LastTime = time.Now()
	t.LastTempTime = nil
}

func (t *TimeTracker) TrackStep(msg string, withReset bool, timeUnit string, opts ...string) {
	if t.IsDisabled {
		return
	}
	n := time.Now()
	dur := n.Sub(t.LastTime)

	if !withReset && t.LastTempTime != nil {
		dur = n.Sub(*t.LastTempTime)
	}

	t.Print(msg, dur, timeUnit, opts)
	if withReset {
		t.LastTime = n
		t.LastTempTime = nil
	} else {
		t.LastTempTime = &n
	}
}

func (t *TimeTracker) TrackFromStart(msg string, timeUnit string, opts ...string) {
	n := time.Now()
	dur := n.Sub(t.StartTime)
	t.Print(msg, dur, timeUnit, opts)
}

func (t *TimeTracker) Print(msg string, dur time.Duration, timeUnit string, opts []string) {
	var s string
	switch timeUnit {
	case TimeTrackesUnitMS:
		s = fmt.Sprintf("%s took: %d ms", msg, dur.Milliseconds())
	case TimeTrackesUnitNS:
		s = fmt.Sprintf("%s took: %d ns", msg, dur.Nanoseconds())
	case TimeTrackesUnitS:
		s = fmt.Sprintf("%s took: %f s", msg, dur.Seconds())
	case TimeTrackesUnitM:
		s = fmt.Sprintf("%s took: %f min", msg, dur.Minutes())
	case TimeTrackesUnitMic:
		s = fmt.Sprintf("%s took: %d min", msg, dur.Microseconds())
	}

	if len(opts) > 0 {
		for _, v := range opts {
			s = fmt.Sprintf("%s, %s", s, v)
		}
	}

	log.Println(s)
}
