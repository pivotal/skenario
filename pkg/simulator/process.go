package simulator

type ProcessIdentity string

type Process interface {
	Identity() ProcessIdentity
	OnOccurrence(event *Event) (result TransitionResult)
}

type SchedulingListener interface {
	Identity() ProcessIdentity
	OnSchedule(event *Event)
}
