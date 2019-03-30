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

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/cache"
)

type MovementPriorityQueue interface {
	EnqueueMovement(movement Movement) error
	DequeueMovement() (movement Movement, err error, closed bool)
	Close()
	IsClosed() bool
}

type movementPQ struct {
	heap *cache.Heap
}

func (mpq *movementPQ) EnqueueMovement(movement Movement) error {
	key, err := occursAtToKey(movement)
	if err != nil {
		return fmt.Errorf("could not create a heap key for Movement %s: %s", movement.Kind(), err.Error())
	}

	_, exists, err := mpq.heap.GetByKey(key)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf(
			"could not add Movement '%s' to run at '%d', there is already another movement scheduled at that time",
			movement.Kind(),
			movement.OccursAt().UnixNano(),
		)
	}

	return mpq.heap.Add(movement)
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
	heap := cache.NewHeap(occursAtToKey, leftMovementIsEarlier)

	return &movementPQ{
		heap: heap,
	}
}

func occursAtToKey(movement interface{}) (key string, err error) {
	mv := movement.(Movement)
	return strconv.FormatInt(mv.OccursAt().UnixNano(), 10), nil
}

func leftMovementIsEarlier(left interface{}, right interface{}) bool {
	l := left.(Movement)
	r := right.(Movement)

	return l.OccursAt().Before(r.OccursAt())
}
