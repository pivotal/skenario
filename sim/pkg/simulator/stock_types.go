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

type StockName string

type baseStock interface {
	Name() StockName
	KindStocked() EntityKind
	Count() uint64
	EntitiesInStock() map[Entity]bool
	GetEntityByNumber(number int) Entity
}

type removable interface {
	Remove(entity *Entity) Entity
}

type addable interface {
	Add(entity Entity) error
}

type SourceStock interface {
	baseStock
	removable
}

type SinkStock interface {
	baseStock
	addable
}

type ThroughStock interface {
	baseStock
	removable
	addable
}
