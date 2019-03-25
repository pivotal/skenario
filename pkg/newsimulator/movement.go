package newsimulator

import "time"

type Movement interface {
	OccursAt() time.Time
	From() SourceStock
	To() SinkStock
}

type move struct {
	from     SourceStock
	to       SinkStock
	occursAt time.Time
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

func NewMovement(occursAt time.Time, from SourceStock, to SinkStock) Movement {
	return &move{
		occursAt: occursAt,
		to: to,
		from: from,
	}
}

