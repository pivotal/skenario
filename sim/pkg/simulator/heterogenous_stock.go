package simulator

import "fmt"

type heterogenousStock struct {
	name       StockName
	stocksKind EntityKind
	stock      map[Entity]bool
}

func (hs *heterogenousStock) Name() StockName {
	return hs.name
}

func (hs *heterogenousStock) KindStocked() EntityKind {
	return hs.stocksKind
}

func (hs *heterogenousStock) Count() uint64 {
	return uint64(len(hs.stock))
}

func (hs *heterogenousStock) EntitiesInStock() []*Entity {
	entities := make([]*Entity, 0)

	for entity := range hs.stock {
		entities = append(entities, &entity)
	}
	return entities
}

func (hs *heterogenousStock) Add(entity Entity) error {
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
	hs.stock[entity] = true
	return nil
}

func (hs *heterogenousStock) Remove(entity *Entity) Entity {
	if hs.stock[*entity] {
		delete(hs.stock, *entity)
		return *entity
	}

	return nil
}

func newHeterogenousBaseStock(name StockName, kind EntityKind) *heterogenousStock {
	return &heterogenousStock{
		name:       name,
		stocksKind: kind,
		stock:      make(map[Entity]bool, 0),
	}
}

func NewHeterogenousThroughStock(name StockName, stocks EntityKind) ThroughStock {
	return newHeterogenousBaseStock(name, stocks)
}
