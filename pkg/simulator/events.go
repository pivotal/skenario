package simulator

import "time"

type AdvanceFunc func(t time.Time, eventName string)  (identifier, outcome string)

type Event struct {
	Time        time.Time
	EventName   string
	AdvanceFunc AdvanceFunc
}
