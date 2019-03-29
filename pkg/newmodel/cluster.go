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
	RecordToAutoscaler(scaler autoscaler.UniScaler, atTime *time.Time, ctx context.Context)
}

type clusterModel struct {
	env               newsimulator.Environment
	currentDesired    int32
	replicasLaunching newsimulator.ThroughStock
	replicasActive    newsimulator.ThroughStock
}

func (cm *clusterModel) Env() newsimulator.Environment {
	return cm.env
}

func (cm *clusterModel) CurrentDesired() int32 {
	return cm.currentDesired
}

func (cm *clusterModel) SetDesired(desired int32) {
	desireDelta := desired - int32(cm.replicasLaunching.Count())

	delay := 10 * time.Nanosecond
	if desireDelta > 0 {
		for ; desireDelta > 0; desireDelta-- {
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
		for ; desireDelta < 0; desireDelta++ {
			cm.replicasLaunching.Remove()
		}
	} else {
		// No change.
	}

	cm.currentDesired = desired
}

func (cm *clusterModel) CurrentLaunching() uint64 {
	return cm.replicasLaunching.Count()
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
		env:               env,
		replicasLaunching: newsimulator.NewThroughStock("ReplicasLaunching", newsimulator.EntityKind("Replica")),
		replicasActive:    newsimulator.NewThroughStock("ReplicasActive", newsimulator.EntityKind("Replica")),
	}
}
