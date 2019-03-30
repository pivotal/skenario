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

package newsimulator

import "fmt"

type stock struct {
	name       StockName
	stocksKind EntityKind

	stock []Entity
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

func (s *stock) EntitiesInStock() []Entity {
	return s.stock
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

	s.stock = append(s.stock, entity)
	return nil
}

func (s *stock) Remove() Entity {
	if s.Count() > 0 {
		e := s.stock[0]
		s.stock = s.stock[:s.Count()-1]
		return e
	}

	return nil
}

func newBaseStock(name StockName, kind EntityKind) *stock {
	return &stock{
		name:       name,
		stocksKind: kind,
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
