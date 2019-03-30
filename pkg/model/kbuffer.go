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

package model

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/knative/serving/pkg/autoscaler"

	"knative-simulator/pkg/simulator"
)

type KBuffer struct {
	env              *simulator.Environment
	requestsBuffered map[simulator.ProcessIdentity]*Request
	replicas         map[simulator.ProcessIdentity]*RevisionReplica
	autoscaler       *KnativeAutoscaler
}

const (
	addReplicaToKBuffer      = "add_replica_to_kbuffer"
	removeReplicaFromKBuffer = "remove_replica_from_kbuffer"
)

func (kb *KBuffer) Identity() simulator.ProcessIdentity {
	return "KBuffer"
}

func (kb *KBuffer) UpdateStock(movement simulator.StockMovementEvent) {

	switch movement.Name() {
	case addReplicaToKBuffer:
		rr := movement.Subject().(*RevisionReplica)
		kb.replicas[movement.SubjectIdentity()] = rr
	case removeReplicaFromKBuffer:
		delete(kb.replicas, movement.SubjectIdentity())
	default:
		if kb == movement.To() {
			numRequests := len(kb.requestsBuffered)
			numReplicas := len(kb.replicas)

			req := movement.Subject().(*Request)
			kb.requestsBuffered[req.Identity()] = req
			numRequests++

			if numRequests > 0 && numReplicas > 0 {
				var replicaKeys []simulator.ProcessIdentity
				for k := range kb.replicas {
					replicaKeys = append(replicaKeys, k)
				}
				key := replicaKeys[rand.Intn(numReplicas)]

				i := 1
				for _, v := range kb.requestsBuffered {
					mv := simulator.NewMovementEvent(
						sendToReplica,
						movement.OccursAt().Add(time.Duration(i)*time.Nanosecond),
						v,
						kb,
						kb.replicas[key],
					)

					kb.env.Schedule(mv)
					i++
				}
			} else if numRequests > 0 {
				occurs := movement.OccursAt().Add(1 * time.Nanosecond)
				kb.autoscaler.autoscaler.Record(context.Background(), autoscaler.Stat{
					Time:                      &occurs,
					PodName:                   "KBuffer",
					AverageConcurrentRequests: float64(numRequests),
					RequestCount:              int32(numRequests),
				})
			}

		} else if kb == movement.From() {
			delete(kb.requestsBuffered, movement.SubjectIdentity())
		} else {
			panic(fmt.Errorf("impossible movement: %+v", movement))
		}
	}
}

func (kb *KBuffer) OnSchedule(event simulator.Event) {
	switch event.Name() {
	case finishLaunchingReplica:
		gevt := event.(simulator.GeneralEvent)
		rr := gevt.Subject().(*RevisionReplica)

		kb.env.Schedule(simulator.NewMovementEvent(
			addReplicaToKBuffer,
			event.OccursAt().Add(10*time.Millisecond),
			rr,
			&Cluster{},
			kb,
		))
	case terminateReplica:
		gevt := event.(simulator.GeneralEvent)
		rr := gevt.Subject().(*RevisionReplica)

		kb.env.Schedule(simulator.NewMovementEvent(
			removeReplicaFromKBuffer,
			event.OccursAt().Add(-10*time.Millisecond),
			rr,
			kb,
			&Cluster{},
		))
	}
}

func NewKBuffer(env *simulator.Environment, scaler *KnativeAutoscaler) *KBuffer {
	kb := &KBuffer{
		env:              env,
		requestsBuffered: make(map[simulator.ProcessIdentity]*Request),
		replicas:         make(map[simulator.ProcessIdentity]*RevisionReplica),
		autoscaler:       scaler,
	}

	kb.autoscaler.buffer = kb

	t := env.Time()

	kb.autoscaler.autoscaler.Record(context.Background(), autoscaler.Stat{
		Time:                      &t,
		PodName:                   "KBuffer",
		AverageConcurrentRequests: 0,
		RequestCount:              0,
	})

	return kb
}
