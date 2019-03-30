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
	"fmt"
	"math/rand"
	"time"

	"github.com/looplab/fsm"

	"knative-simulator/pkg/simulator"
)

const (
	StateRequestArrived            = "RequestArrived"
	StateRequestBuffered           = "RequestBuffered"
	StateRequestSendingToReplica   = "RequestSendingToReplica"
	StateRequestProcessing         = "RequestProcessing"
	StateRequestProcessingFinished = "RequestFinished"

	bufferRequest           = "buffer_request"
	sendToReplica           = "send_to_replica"
	beginRequestProcessing  = "begin_request_processing"
	finishRequestProcessing = "finish_request_processing"
)

var (
	evtRequestedBuffered       = fsm.EventDesc{Name: bufferRequest, Src: []string{StateRequestArrived}, Dst: StateRequestBuffered}
	evtRequestedSentToReplica  = fsm.EventDesc{Name: sendToReplica, Src: []string{StateRequestBuffered}, Dst: StateRequestSendingToReplica}
	evtBeginRequestProcessing  = fsm.EventDesc{Name: beginRequestProcessing, Src: []string{StateRequestSendingToReplica}, Dst: StateRequestProcessing}
	evtFinishRequestProcessing = fsm.EventDesc{Name: finishRequestProcessing, Src: []string{StateRequestProcessing}, Dst: StateRequestProcessingFinished}
)

type Request struct {
	name         simulator.ProcessIdentity
	fsm          *fsm.FSM
	env          *simulator.Environment
	currentStock simulator.Stock
	arrivalTime  time.Time
}

func (r *Request) Identity() simulator.ProcessIdentity {
	return r.name
}

func (r *Request) OnOccurrence(event simulator.Event) (result simulator.StateTransitionResult) {
	n := ""

	switch event.Name() {
	case beginRequestProcessing:
		r.env.Schedule(simulator.NewGeneralEvent(
			finishRequestProcessing,
			event.OccursAt().Add(500*time.Millisecond),
			r,
		))
	case finishRequestProcessing:
		duration := event.OccursAt().Sub(r.arrivalTime)
		n = fmt.Sprintf("Request took %dms", duration.Nanoseconds()/1000000)
	}

	currentState := r.fsm.Current()
	err := r.fsm.Event(string(event.Name()))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			//panic(err.Error())
			fmt.Println(err.Error())
		}
	}

	return simulator.StateTransitionResult{FromState: currentState, ToState: r.fsm.Current(), Note: n}
}

func (r *Request) OnMovement(movement simulator.StockMovementEvent) (result simulator.MovementResult) {
	r.currentStock = movement.To()

	switch movement.Name() {
	case bufferRequest:
		//currentState := r.fsm.Current()
	case sendToReplica:
		r.env.Schedule(simulator.NewGeneralEvent(
			beginRequestProcessing,
			movement.OccursAt().Add(10*time.Millisecond),
			r,
		))
	}
	err := r.fsm.Event(string(movement.Name()))
	if err != nil {
		switch err.(type) {
		case fsm.NoTransitionError:
		// ignore
		default:
			fmt.Println(err.Error())
		}
	}

	return simulator.MovementResult{FromStock: movement.From(), ToStock: movement.To()}
}

func (r *Request) CurrentlyAt() simulator.Stock {
	return r.currentStock
}

func NewRequest(env *simulator.Environment, buffer *KBuffer, arrivalTime time.Time) *Request {
	req := &Request{
		name:         simulator.ProcessIdentity(fmt.Sprintf("req-%012d", rand.Int63n(100000000000))),
		env:          env,
		currentStock: buffer,
		arrivalTime:  arrivalTime,
	}

	req.fsm = fsm.NewFSM(
		StateRequestArrived,
		fsm.Events{
			evtRequestedBuffered,
			evtRequestedSentToReplica,
			evtBeginRequestProcessing,
			evtFinishRequestProcessing,
		},
		fsm.Callbacks{},
	)

	return req
}
