package simulator

import "time"

type AdvanceFunc func(time time.Time, description string)

type Event struct {
	Time        time.Time
	Description string
	AdvanceFunc AdvanceFunc
}
