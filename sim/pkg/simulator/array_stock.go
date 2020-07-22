package simulator

import "fmt"

type arrayStock struct {
	name       StockName
	stocksKind EntityKind
	stock      []*Entity
}

func (as *arrayStock) Name() StockName {
	return as.name
}

func (as *arrayStock) KindStocked() EntityKind {
	return as.stocksKind
}

func (as *arrayStock) Count() uint64 {
	return uint64(len(as.stock))
}

//time complexity O(len(as.stock))
func (as *arrayStock) EntitiesInStock() []*Entity {
	return as.stock
}

//time complexity O(1)
func (as *arrayStock) Add(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("could not add Entity, as it was nil")
	}

	if entity.Kind() != as.KindStocked() {
		return fmt.Errorf(
			"stock '%s' could not stock entity '%s'; stock accepts '%s' but kind is '%s'",
			as.Name(),
			entity.Name(),
			as.KindStocked(),
			entity.Kind(),
		)
	}

	as.stock = append(as.stock, &entity)
	return nil
}

//time complexity O(len(as.stock))
func (as *arrayStock) Remove(entity *Entity) Entity {
	//remove any entity
	if entity == nil {
		var e *Entity
		if as.Count() > 0 {
			e, as.stock = as.stock[0], as.stock[1:]
			return *e
		}
		return nil
	}
	//remove a particular entity
	removed := false
	for i := 0; i < len(as.stock); i++ {
		currentEntity := *as.stock[i]
		if currentEntity == *entity {
			as.stock[i] = as.stock[len(as.stock)-1]
			removed = true
		}
	}
	if removed {
		as.stock = as.stock[:len(as.stock)-1]
		return *entity
	}
	return nil
}

func newArrayBaseStock(name StockName, kind EntityKind) *arrayStock {
	return &arrayStock{
		name:       name,
		stocksKind: kind,
	}
}

// Constructors

func NewArrayThroughStock(name StockName, stocks EntityKind) ThroughStock {
	return newArrayBaseStock(name, stocks)
}

func NewSourceStock(name StockName, sinks EntityKind) SourceStock {
	return newArrayBaseStock(name, sinks)
}

func NewSinkStock(name StockName, sinks EntityKind) SinkStock {
	return newArrayBaseStock(name, sinks)
}
