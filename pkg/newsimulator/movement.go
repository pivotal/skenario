package newsimulator

type Movement interface {
	From() SourceStock
	To() SinkStock
}

type move struct {
	from SourceStock
	to   SinkStock
}

func (mv *move) From() SourceStock {
	return mv.from
}

func (mv *move) To() SinkStock {
	return mv.to
}

func NewMovement(from SourceStock, to SinkStock) Movement {
	return &move{
		to: to,
		from: from,
	}
}

