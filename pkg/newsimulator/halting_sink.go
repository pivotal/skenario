package newsimulator

// HaltingSink is intended for use by the Environment.
// It terminates .Run() by closing the future movements list.

type haltingSink struct {
	delegate        ThroughStock
	futureMovements MovementPriorityQueue
}

func NewHaltingSink(name StockName, stocks EntityKind, futureMovements MovementPriorityQueue) *haltingSink {
	return &haltingSink{
		delegate:        NewThroughStock(name, stocks),
		futureMovements: futureMovements,
	}
}

func (hs *haltingSink) Name() StockName {
	return hs.delegate.Name()
}

func (hs *haltingSink) KindStocked() EntityKind {
	return hs.delegate.KindStocked()
}

func (hs *haltingSink) Count() uint64 {
	return hs.delegate.Count()
}

func (hs *haltingSink) Add(entity Entity) error {
	hs.futureMovements.Close()
	return hs.delegate.Add(entity)
}

func (hs *haltingSink) Remove() Entity {
	return hs.delegate.Remove()
}

func (hs *haltingSink) EntitiesInStock() []Entity {
	return hs.delegate.EntitiesInStock()
}
