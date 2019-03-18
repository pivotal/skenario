package simulator

import "time"

type EventName string

type Event struct {
	OccursAt time.Time
	Name     EventName
	Subject  Process
}

type TransitionResult struct {
	FromState string
	ToState   string
	Note      string
}
