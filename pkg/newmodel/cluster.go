package newmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/knative/serving/pkg/autoscaler"

	"knative-simulator/pkg/newsimulator"
)

type ClusterModel interface {
	Model
	CurrentDesired() int32
	SetDesired(int32)
	CurrentLaunching() uint64
	CurrentActive() uint64
	RecordToAutoscaler(scaler autoscaler.UniScaler, atTime *time.Time, ctx context.Context)
}

type clusterModel struct {
	env                newsimulator.Environment
	currentDesired     int32
	replicasLaunching  newsimulator.ThroughStock
	replicasActive     newsimulator.ThroughStock
	replicasTerminated newsimulator.SinkStock
}

func (cm *clusterModel) Env() newsimulator.Environment {
	return cm.env
}

//TODO: can we get rid of this and the variable?
func (cm *clusterModel) CurrentDesired() int32 {
	return cm.currentDesired
}

func (cm *clusterModel) SetDesired(desired int32) {
	launching := int32(cm.replicasLaunching.Count())
	active := int32(cm.replicasActive.Count())

	desireDelta := desired - (launching + active)

	delay := 10 * time.Nanosecond
	if desireDelta > 0 {
		for ; desireDelta > 0; desireDelta-- {
			// TODO: better replica names, please
			err := cm.replicasLaunching.Add(newsimulator.NewEntity("a replica", newsimulator.EntityKind("Replica")))
			if err != nil {
				panic(fmt.Sprintf("could not scale up in ClusterModel: %s", err.Error()))
			}

			cm.env.AddToSchedule(newsimulator.NewMovement(
				"launching -> active",
				cm.env.CurrentMovementTime().Add(delay),
				cm.replicasLaunching,
				cm.replicasActive,
			))

			delay += 10
		}
	} else if desireDelta < 0 {
		// for now I assume launching replicas are terminated before active replicas
		desireDelta = desireDelta + launching
		for ; launching > 0; launching-- {
			cm.env.AddToSchedule(newsimulator.NewMovement(
				"launching -> terminated",
				cm.env.CurrentMovementTime().Add(delay),
				cm.replicasLaunching,
				cm.replicasTerminated,
			))
			delay += 10
		}

		for ; desireDelta < 0; desireDelta++ {
			cm.env.AddToSchedule(newsimulator.NewMovement(
				"active -> terminated",
				cm.env.CurrentMovementTime().Add(delay),
				cm.replicasActive,
				cm.replicasTerminated,
			))
			delay += 10
		}
	} else {
		// No change.
	}

	cm.currentDesired = desired
}

func (cm *clusterModel) CurrentLaunching() uint64 {
	return cm.replicasLaunching.Count()
}

func (cm *clusterModel) CurrentActive() uint64 {
	return cm.replicasActive.Count()
}

func (cm *clusterModel) RecordToAutoscaler(scaler autoscaler.UniScaler, atTime *time.Time, ctx context.Context) {
	for _, e := range cm.replicasActive.EntitiesInStock() {
		scaler.Record(ctx, autoscaler.Stat{
			Time:                      atTime,
			PodName:                   string(e.Name()),
			AverageConcurrentRequests: 1,
			RequestCount:              1,
		})
	}
}

func NewCluster(env newsimulator.Environment) ClusterModel {
	return &clusterModel{
		env:                env,
		replicasLaunching:  newsimulator.NewThroughStock("ReplicasLaunching", newsimulator.EntityKind("Replica")),
		replicasActive:     newsimulator.NewThroughStock("ReplicasActive", newsimulator.EntityKind("Replica")),
		replicasTerminated: newsimulator.NewSinkStock("ReplicasTerminated", newsimulator.EntityKind("Replica")),
	}
}
