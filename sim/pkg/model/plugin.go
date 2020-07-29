package model

import (
	"skenario/pkg/simulator"
	"time"
)

// BEGIN INTERFACE

const (
	SkInterfaceVersion = 1

	SkMetricCpu         = "cpu"
	SkMetricConcurrency = "concurrency"

	SkStatePending     = "pending"
	SkStateRunning     = "running"
	SkStateReady       = "ready"
	SkStateTerminating = "terminating"
)

type SkPlugin interface {
	NewAutoscaler(SkEnvironment, string) SkAutoscaler
}

type SkEnvironment interface {
	Pods() []SkPod
}

type SkPod interface {
	Name() string
	State() string
	LastTransistion() int64
	CpuRequest() int32
}

type SkAutoscaler interface {
	Scale(int64) (int32, error)
	Stat(SkStat) error
}

type SkStat interface {
	Time() int64
	PodName() string
	Metric() string
	Value() int32
}

// END INTERFACE

type podCpuStat struct {
	time              time.Time
	podName           string
	averageMillicores int32
}

var _ SkStat = (*podCpuStat)(nil)

func (s *podCpuStat) Time() int64 {
	return s.time.UnixNano()
}

func (s *podCpuStat) PodName() string {
	return s.podName
}

func (s *podCpuStat) Metric() string {
	return SkMetricCpu
}

func (s *podCpuStat) Value() int32 {
	return s.averageMillicores
}

type podConcurrencyStat struct {
	time               time.Time
	podName            string
	averageConcurrency int32
}

var _ SkStat = (*podConcurrencyStat)(nil)

func (s *podConcurrencyStat) Time() int64 {
	return s.time.UnixNano()
}

func (s *podConcurrencyStat) PodName() string {
	return s.podName
}

func (s *podConcurrencyStat) Metric() string {
	return SkMetricConcurrency
}

func (s *podConcurrencyStat) Value() int32 {
	return s.averageConcurrency
}

// HORIZONTAL POD AUTOSCALER

type horizontalPodAutoscaler struct {
	env      simulator.Environment
	tickTock AutoscalerTicktockStock
}

func NewHorizontalPodAutoscaler(env simulator.Environment, startAt time.Time, cluster ClusterModel) {
	hpa := &horizontalPodAutoscaler{
		env: env,
	}
	for theTime := startAt.Add(15 * time.Second).Add(time.Nanosecond); theTime.Before(env.HaltTime()); theTime = theTime.Add(15 * time.Second) {
		for _, autoscaler := range hpa.tickTock.EntitiesInStock() {
			hpa.env.AddToSchedule(simulator.NewMovement(
				"autoscaler_tick",
				theTime,
				hpa.tickTock,
				hpa.tickTock,
				autoscaler,
			))
		}
	}
}
