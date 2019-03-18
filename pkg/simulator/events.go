package simulator

import "time"

type Event struct {
	Time        time.Time
	EventName   string
	Subject     Process
}
