package model

import "time"

// BEGIN INTERFACE

var (
	SkInterfaceVersion = 1
	SkCpu              = "cpu"
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

type PodCpuStat struct {
	time               time.Time
	averageUtilization int32
}

func (s *PodCpuStat) Time() int64 {
	return s.time.UnixNano()
}

func (s *PodCpuStat) Metric() string {
	return SkCpu
}

func (s *PodCpuStat) Value() (int32, bool) {
	return 0, false
}

func (s *PodCpuStat) AverageValue() (int32, bool) {
	return 0, false
}

func (s *PodCpuStat) AverageUtilization() (int32, bool) {
	return s.averageUtilization, true
}
