package simulator

type Process interface {
	Identity() string
	OnOccurrence(event *Event) (result TransitionResult)
}

type SchedulingListener interface {
	Identity() string
	OnSchedule(event *Event)
}
