package simulator

import "time"

type Event struct {
	OccursAt  time.Time
	EventName string
	Subject   Process
}

type TransitionResult struct {
	FromState string
	ToState   string
	Note      string
}
