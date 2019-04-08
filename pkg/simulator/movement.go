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

import "time"

type MovementKind string

type Annotateable interface {
	Notes() []string
	AddNote(note string)
}

type coreMovement interface {
	Kind() MovementKind
	OccursAt() time.Time
	From() SourceStock
	To() SinkStock
}

type Movement interface {
	coreMovement
	Annotateable
}

type move struct {
	kind     MovementKind
	from     SourceStock
	to       SinkStock
	occursAt time.Time
	notes    []string
}

func (mv *move) Kind() MovementKind {
	return mv.kind
}

func (mv *move) OccursAt() time.Time {
	return mv.occursAt
}

func (mv *move) From() SourceStock {
	return mv.from
}

func (mv *move) To() SinkStock {
	return mv.to
}

func (mv *move) Notes() []string {
	return mv.notes
}

func (mv *move) AddNote(note string) {
	mv.notes = append(mv.notes, note)
}

func NewMovement(kind MovementKind, occursAt time.Time, from SourceStock, to SinkStock) Movement {
	return &move{
		kind:     kind,
		occursAt: occursAt,
		to:       to,
		from:     from,
		notes:    make([]string, 0),
	}
}
