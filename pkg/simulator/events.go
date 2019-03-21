package simulator

import "time"

const (
	EventGeneral  EventKind = iota
	EventMovement EventKind = iota
)

type EventName string
type EventKind int

func (ek EventKind) String() string {
	return []string{"EventGeneral", "EventMovement"}[ek]
}

func NewGeneralEvent(name EventName, occursAt time.Time, subject Process) Event {
	return &event{
		kind:     EventGeneral,
		name:     name,
		occursAt: occursAt,
		subject:  subject,
	}
}

// General Events

type Event interface {
	Kind() EventKind
	Name() EventName
	OccursAt() time.Time
	Subject() Process
}

type event struct {
	kind     EventKind
	name     EventName
	occursAt time.Time
	subject  Process
}

func (e *event) Kind() EventKind {
	return e.kind
}
func (e *event) Name() EventName {
	return e.name
}
func (e *event) OccursAt() time.Time {
	return e.occursAt
}
func (e *event) Subject() Process {
	return e.subject
}

type TransitionResult struct {
	FromState string
	ToState   string
	Note      string
}
