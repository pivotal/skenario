/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package simulator

import "fmt"

type stock struct {
	name       StockName
	stocksKind EntityKind
	stock      map[Entity]bool
}

func (s *stock) Name() StockName {
	return s.name
}

func (s *stock) KindStocked() EntityKind {
	return s.stocksKind
}

func (s *stock) Count() uint64 {
	return uint64(len(s.stock))
}

func (s *stock) EntitiesInStock() map[Entity]bool {
	return s.stock
}

func (s *stock) GetEntityByNumber(number int) Entity {
	counter := 0
	for entity := range s.stock {
		if counter == number {
			return entity
		}
		counter++
	}
	return nil
}
func (s *stock) Add(entity Entity) error {
	if entity == nil {
		return fmt.Errorf("could not add Entity, as it was nil")
	}

	if entity.Kind() != s.KindStocked() {
		return fmt.Errorf(
			"stock '%s' could not stock entity '%s'; stock accepts '%s' but kind is '%s'",
			s.Name(),
			entity.Name(),
			s.KindStocked(),
			entity.Kind(),
		)
	}
	s.stock[entity] = true
	return nil
}

func (s *stock) Remove(entity *Entity) Entity {
	//we don't need to remove a particular entity (entity == nil), then remove any
	if entity == nil {
		for en := range s.stock {
			delete(s.stock, en)
			return en
		}
		return nil
	}
	if s.stock[*entity] {
		delete(s.stock, *entity)
		return *entity
	}

	return nil
}

func newBaseStock(name StockName, kind EntityKind) *stock {
	return &stock{
		name:       name,
		stocksKind: kind,
		stock:      make(map[Entity]bool, 0),
	}
}

// Constructors

func NewThroughStock(name StockName, stocks EntityKind) ThroughStock {
	return newBaseStock(name, stocks)
}

func NewSourceStock(name StockName, sinks EntityKind) SourceStock {
	return newBaseStock(name, sinks)
}

func NewSinkStock(name StockName, sinks EntityKind) SinkStock {
	return newBaseStock(name, sinks)
}
