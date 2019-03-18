package simulator

type Process interface {
	Identity() string
	OnAdvance(event *Event) (result TransitionResult)
}
