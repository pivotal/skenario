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

func (hs *haltingSink) Remove(entity *Entity) Entity {
	return hs.delegate.Remove(entity)
}

func (hs *haltingSink) EntitiesInStock() map[Entity]bool {
	return hs.delegate.EntitiesInStock()
}

func (hs *haltingSink) GetEntityByNumber(number int) Entity {
	return hs.delegate.GetEntityByNumber(number)
}
