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
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/dispatcher"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
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

func (mss *MockStockType) Remove(entity *Entity) Entity {
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

func (es *EchoSourceStockType) Remove(entity *Entity) Entity {
	name := EntityName(fmt.Sprintf("entity-%d", es.series))
	es.series++
	return NewEntity(name, es.kind)
}

type fakeDispatcher struct {
}

func (fd *fakeDispatcher) Init(pluginsPaths []string) {
}
func (fd *fakeDispatcher) Shutdown() {
}
func (fd *fakeDispatcher) GetPlugin() skplug.Plugin {
	return fd
}

func NewFakeDispatcher() dispatcher.Dispatcher {
	return &fakeDispatcher{}
}

func (fd *fakeDispatcher) Event(partition string, time int64, typ proto.EventType, object skplug.Object) error {
	return nil
}

func (fd *fakeDispatcher) Stat(partition string, stat []*proto.Stat) error {
	return nil
}

func (fd *fakeDispatcher) HorizontalRecommendation(partition string, time int64) (rec int32, err error) {
	return 0, nil
}

func (fd *fakeDispatcher) VerticalRecommendation(partition string, time int64) (rec []*proto.RecommendedPodResources, err error) {
	return []*proto.RecommendedPodResources{}, nil
}

func (fd *fakeDispatcher) GetCapabilities() (rec []proto.Capability, err error) {
	return []proto.Capability{}, nil
}

//TODO to combine these into a Metadata API, because there will be more we want to return from the plugin, like default configuration etc
func (fd *fakeDispatcher) PluginType() (rec string, err error) {
	return "dispatcher-fake", nil
}
