package newsimulator

import "time"

type MovementKind string

type Movement interface {
	Kind() MovementKind
	OccursAt() time.Time
	From() SourceStock
	To() SinkStock
	Notes() []string
	AddNote(note string)
}

type move struct {
	kind     MovementKind
	from     SourceStock
	to       SinkStock
	occursAt time.Time
	notes    []string
}

func (mv *move) Kind() MovementKind {
	return mv.kind
}

func (mv *move) OccursAt() time.Time {
	return mv.occursAt
}

func (mv *move) From() SourceStock {
	return mv.from
}

func (mv *move) To() SinkStock {
	return mv.to
}

func (mv *move) Notes() []string {
	return mv.notes
}

func (mv *move) AddNote(note string) {
	mv.notes = append(mv.notes, note)
}

func NewMovement(kind MovementKind, occursAt time.Time, from SourceStock, to SinkStock, note string) Movement {
	return &move{
		kind:     kind,
		occursAt: occursAt,
		to:       to,
		from:     from,
		notes:    []string{note},
	}
}
