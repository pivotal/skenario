package simulator

import "fmt"

type mapStock struct {
	name       StockName
	stocksKind EntityKind
	stock      map[Entity]bool
}

func (ms *mapStock) Name() StockName {
	return ms.name
}

func (ms *mapStock) KindStocked() EntityKind {
	return ms.stocksKind
}

func (ms *mapStock) Count() uint64 {
	return uint64(len(ms.stock))
}

//time complexity O(len(ms.stock))
func (ms *mapStock) EntitiesInStock() []*Entity {
	entities := make([]*Entity, 0)

	for entity := range ms.stock {
		entities = append(entities, &entity)
	}
	return entities
}

//time complexity O(1)
func (ms *mapStock) Add(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("could not add Entity, as it was nil")
	}

	if entity.Kind() != ms.KindStocked() {
		return fmt.Errorf(
			"stock '%s' could not stock entity '%s'; stock accepts '%s' but kind is '%s'",
			ms.Name(),
			entity.Name(),
			ms.KindStocked(),
			entity.Kind(),
		)
	}
	ms.stock[entity] = true
	return nil
}

//time complexity O(1)
func (ms *mapStock) Remove(entity *Entity) Entity {
	//remove any entity
	if entity == nil {
		var toRemove Entity
		for en := range ms.stock {
			toRemove = en
			break
		}
		if toRemove != nil {
			ms.stock[toRemove] = false
			return toRemove
		}
		return nil
	}
	//remove a particular entity
	if ms.stock[*entity] {
		delete(ms.stock, *entity)
		return *entity
	}

	return nil
}

func newMapBaseStock(name StockName, kind EntityKind) *mapStock {
	return &mapStock{
		name:       name,
		stocksKind: kind,
		stock:      make(map[Entity]bool, 0),
	}
}

func NewMapThroughStock(name StockName, stocks EntityKind) ThroughStock {
	return newMapBaseStock(name, stocks)
}
