package model

import (
	"knative-simulator/pkg/simulator"
)

type KBuffer struct {
	env              *simulator.Environment
	requestsBuffered map[simulator.ProcessIdentity]*Request
	replicas         map[simulator.ProcessIdentity]*RevisionReplica
}

func (kb *KBuffer) AddRequest(reqName simulator.ProcessIdentity, req *Request) {

}

func (kb *KBuffer) DeleteRequest(reqName simulator.ProcessIdentity) *Request {
	delReq := kb.requestsBuffered[reqName]
	delete(kb.requestsBuffered, reqName)

	return delReq
}

func (kb *KBuffer) AddReplica(replica *RevisionReplica) {
	kb.replicas[replica.Identity()] = replica
}

func (kb *KBuffer) DeleteReplica(replica *RevisionReplica) *RevisionReplica {
	delRev := kb.replicas[replica.Identity()]
	delete(kb.replicas, replica.Identity())

	return delRev
}

func NewKBuffer(env *simulator.Environment) *KBuffer {
	return &KBuffer{
		env:              env,
		requestsBuffered: make(map[simulator.ProcessIdentity]*Request),
		replicas:         make(map[simulator.ProcessIdentity]*RevisionReplica),
	}
}
