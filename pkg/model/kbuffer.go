package model

import "knative-simulator/pkg/simulator"

type KBuffer struct {
	env      *simulator.Environment
	requests map[string]*Request
}

func (kb *KBuffer) AddRequest(reqName string, req *Request) {
	kb.requests[reqName] = req
}

func (kb *KBuffer) DeleteRequest(reqName string) *Request {
	delReq := kb.requests[reqName]
	delete(kb.requests, reqName)

	return delReq
}

func NewKBuffer(env *simulator.Environment) *KBuffer {
	return &KBuffer{
		env:      env,
		requests: make(map[string]*Request),
	}
}
