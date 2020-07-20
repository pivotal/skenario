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

import (
	"fmt"

	"github.com/stretchr/testify/mock"
)

type MockStockType struct {
	mock.Mock
}

func (mss *MockStockType) Name() StockName {
	mss.Called()
	return StockName("mock source")
}

func (mss *MockStockType) KindStocked() EntityKind {
	mss.Called()
	return EntityKind("mock kind")
}

func (mss *MockStockType) Count() uint64 {
	mss.Called()
	return uint64(0)
}

func (mss *MockStockType) EntitiesInStock() []*Entity {
	mss.Called()
	return []*Entity{}
}

func (mss *MockStockType) Remove() Entity {
	mss.Called()
	return NewEntity("test entity", "mock kind")
}

func (mss *MockStockType) Add(entity Entity) error {
	mss.Called(entity)
	return nil
}

// We hand-roll the echo source stock, otherwise the compiler will use ThroughStock,
// leading to nil errors when we try to .Remove() a non-existent entry.
type EchoSourceStockType struct {
	name   StockName
	kind   EntityKind
	series int
}

func (es *EchoSourceStockType) Name() StockName {
	return es.name
}

func (es *EchoSourceStockType) KindStocked() EntityKind {
	return es.kind
}

func (es *EchoSourceStockType) Count() uint64 {
	return 0
}

func (es *EchoSourceStockType) EntitiesInStock() []*Entity {
	return []*Entity{}
}

func (es *EchoSourceStockType) Remove() Entity {
	name := EntityName(fmt.Sprintf("entity-%d", es.series))
	es.series++
	return NewEntity(name, es.kind)
}
