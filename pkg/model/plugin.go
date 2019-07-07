package model

import "time"

// BEGIN INTERFACE

const (
	SkInterfaceVersion  = 1
	SkMetricCpu         = "cpu"
	SkMetricConcurrency = "concurrency"
)

type SkAutoscaler interface {
	Scale(int64) (int32, error)
	Stat(SkStat) error
}

type SkStat interface {
	Time() int64
	Metric() string
	Value() (int32, bool)
	AverageValue() (int32, bool)
	AverageUtilization() (int32, bool)
}

// END INTERFACE

type podCpuStat struct {
	time               time.Time
	averageUtilization int32
}

var _ SkStat = (*podCpuStat)(nil)

func (s *podCpuStat) Time() int64 {
	return s.time.UnixNano()
}

func (s *podCpuStat) Metric() string {
	return SkMetricCpu
}

func (s *podCpuStat) Value() (int32, bool) {
	return 0, false
}

func (s *podCpuStat) AverageValue() (int32, bool) {
	return 0, false
}

func (s *podCpuStat) AverageUtilization() (int32, bool) {
	return s.averageUtilization, true
}

type podConcurrencyStat struct {
	time         time.Time
	averageValue int32
}

var _ SkStat = (*podConcurrencyStat)(nil)

func (s *podConcurrencyStat) Time() int64 {
	return s.time.UnixNano()
}

func (s *podConcurrencyStat) Metric() string {
	return SkMetricConcurrency
}

func (s *podConcurrencyStat) Value() (int32, bool) {
	return 0, false
}

func (s *podConcurrencyStat) AverageValue() (int32, bool) {
	return s.averageValue, true
}

func (s *podConcurrencyStat) AverageUtilization() (int32, bool) {
	return 0, false
}
