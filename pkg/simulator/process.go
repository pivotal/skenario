package simulator

type ProcessIdentity string

type Identifiable interface {
	Identity() ProcessIdentity
}

type Process interface {
	Identifiable
	OnOccurrence(event Event) (result TransitionResult)
}

type SchedulingListener interface {
	Identifiable
	OnSchedule(event Event)
}
