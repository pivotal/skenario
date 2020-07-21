package simulator

import "fmt"

type homogenousStock struct {
	name       StockName
	stocksKind EntityKind
	stock      []*Entity
}

func (hs *homogenousStock) Name() StockName {
	return hs.name
}

func (hs *homogenousStock) KindStocked() EntityKind {
	return hs.stocksKind
}

func (hs *homogenousStock) Count() uint64 {
	return uint64(len(hs.stock))
}

func (hs *homogenousStock) EntitiesInStock() []*Entity {
	return hs.stock
}

func (hs *homogenousStock) Add(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("could not add Entity, as it was nil")
	}

	if entity.Kind() != hs.KindStocked() {
		return fmt.Errorf(
			"stock '%s' could not stock entity '%s'; stock accepts '%s' but kind is '%s'",
			hs.Name(),
			entity.Name(),
			hs.KindStocked(),
			entity.Kind(),
		)
	}

	hs.stock = append(hs.stock, &entity)
	return nil
}

func (hs *homogenousStock) Remove(entity *Entity) Entity {
	var e *Entity
	if hs.Count() > 0 {
		e, hs.stock = hs.stock[0], hs.stock[1:]
		return *e
	}

	return nil
}

func newHomogenousBaseStock(name StockName, kind EntityKind) *homogenousStock {
	return &homogenousStock{
		name:       name,
		stocksKind: kind,
	}
}

// Constructors

func NewHomogenousThroughStock(name StockName, stocks EntityKind) ThroughStock {
	return newHomogenousBaseStock(name, stocks)
}

func NewSourceStock(name StockName, sinks EntityKind) SourceStock {
	return newHomogenousBaseStock(name, sinks)
}

func NewSinkStock(name StockName, sinks EntityKind) SinkStock {
	return newHomogenousBaseStock(name, sinks)
}
