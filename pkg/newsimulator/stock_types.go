package newsimulator

type StockName string

type baseStock interface {
	Name() StockName
	KindStocked() EntityKind
	Count() uint64
}

type removable interface {
	Remove() Entity
}

type addable interface {
	Add(entity Entity) error
}

type SourceStock interface {
	baseStock
	removable
}

type SinkStock interface {
	baseStock
	addable
}

type ThroughStock interface {
	baseStock
	removable
	addable
}
