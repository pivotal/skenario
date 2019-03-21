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

// General Events

type Event interface {
	Kind() EventKind
	Name() EventName
	OccursAt() time.Time
	SubjectIdentity() ProcessIdentity
}

type GeneralEvent interface {
	Event
	Subject() Process
}

type generalEvent struct {
	kind     EventKind
	name     EventName
	occursAt time.Time
	subject  Process
}

func (e *generalEvent) Kind() EventKind {
	return e.kind
}

func (e *generalEvent) Name() EventName {
	return e.name
}

func (e *generalEvent) OccursAt() time.Time {
	return e.occursAt
}

func (e *generalEvent) Subject() Process {
	return e.subject
}

func (e *generalEvent) SubjectIdentity() ProcessIdentity {
	return e.subject.Identity()
}

// TODO: add the event so that we can collect results and print them all at once, instead of incrementally.
type StateTransitionResult struct {
	FromState string
	ToState   string
	Note      string
}

func NewGeneralEvent(name EventName, occursAt time.Time, subject Process) GeneralEvent {
	return &generalEvent{
		kind:     EventGeneral,
		name:     name,
		occursAt: occursAt,
		subject:  subject,
	}
}

// Stock Movement Events

type StockMovementEvent interface {
	Event
	Subject() Stockable
	From() Stock
	To() Stock
}

type movement struct {
	kind     EventKind
	name     EventName
	occursAt time.Time
	subject  Stockable
	from     Stock
	to       Stock
}

func (m *movement) Kind() EventKind {
	return m.kind
}

func (m *movement) Name() EventName {
	return m.name
}

func (m *movement) OccursAt() time.Time {
	return m.occursAt
}

func (m *movement) From() Stock {
	return m.from
}

func (m *movement) To() Stock {
	return m.to
}

func (m *movement) Subject() Stockable {
	return m.subject
}

func (m *movement) SubjectIdentity() ProcessIdentity {
	return m.subject.Identity()
}

type MovementResult struct {
	FromStock Stock
	ToStock   Stock
	Note      string
}

func NewMovementEvent(name EventName, occursAt time.Time, subject Stockable, from Stock, to Stock) StockMovementEvent {
	return &movement{
		kind:     EventMovement,
		name:     name,
		occursAt: occursAt,
		subject:  subject,
		from:     from,
		to:       to,
	}
}
