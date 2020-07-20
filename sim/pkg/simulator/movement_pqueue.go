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
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/tools/cache"
)

type MovementPriorityQueue interface {
	EnqueueMovement(movement Movement) (wasShifted bool, scheduledAt time.Time, err error)
	DequeueMovement() (movement Movement, err error, closed bool)
	Close()
	IsClosed() bool
}

type movementPQ struct {
	heap *cache.Heap
}

func (mpq *movementPQ) EnqueueMovement(movement Movement) (wasShifted bool, scheduledAt time.Time, err error) {
	wasShifted = false
	i := 0 * time.Nanosecond
	for {
		key := occursAtToStr(movement.OccursAt().Add(i))

		_, exists, err := mpq.heap.GetByKey(key)
		if err != nil {
			return false, time.Unix(0, -1), err
		}

		if exists {
			i++
			wasShifted = true
		} else {
			break
		}
	}

	if wasShifted {
		shiftedMovement := NewMovement(movement.Kind(), movement.OccursAt().Add(i), movement.From(), movement.To())
		return true, shiftedMovement.OccursAt(), mpq.heap.Add(shiftedMovement)
	}

	return false, movement.OccursAt(), mpq.heap.Add(movement)
}

// DequeueMovement picks the next earliest movement from the queue.
// It will block until there is a Movement to retrieve
// Returns:
// 	movement - the next Movement, if available
// 	err - any errors
// 	closed - whether the underlying queue has "closed", meaning no further
// 	movements can be dequeued.
func (mpq *movementPQ) DequeueMovement() (movement Movement, err error, closed bool) {
	n, err := mpq.heap.Pop()

	if err != nil && strings.Contains(err.Error(), "heap is closed") {
		return nil, nil, true
	} else if err != nil {
		return nil, err, false
	}

	next := n.(Movement)
	return next, nil, false
}

func (mpq *movementPQ) Close() {
	mpq.heap.Close()
}

func (mpq *movementPQ) IsClosed() bool {
	return mpq.heap.IsClosed()
}

func NewMovementPriorityQueue() MovementPriorityQueue {
	heap := cache.NewHeap(movementToKey, leftMovementIsEarlier)

	return &movementPQ{
		heap: heap,
	}
}

func movementToKey(movement interface{}) (key string, err error) {
	mv := movement.(Movement)
	return occursAtToStr(mv.OccursAt()), nil
}

func occursAtToStr(occursAt time.Time) string {
	return strconv.FormatInt(occursAt.UnixNano(), 10)
}

func leftMovementIsEarlier(left interface{}, right interface{}) bool {
	l := left.(Movement)
	r := right.(Movement)

	return l.OccursAt().Before(r.OccursAt())
}
