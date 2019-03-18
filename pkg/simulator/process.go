package simulator

type Process interface {
	Identity() string
	OnAdvance(event *Event) (result TransitionResult)
}

type SchedulingListener interface {
	Identity() string
	OnSchedule(event *Event)
}
