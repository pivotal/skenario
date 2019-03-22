package simulator

type ProcessIdentity string

type Identifiable interface {
	Identity() ProcessIdentity
}

type Process interface {
	Identifiable
	OnOccurrence(event Event) (result StateTransitionResult)
}

type SchedulingListener interface {
	Identifiable
	OnSchedule(event Event)
}

type Stock interface {
	Identifiable
	UpdateStock(movement StockMovementEvent)
}

type Stockable interface {
	Identifiable
	OnMovement(movement StockMovementEvent) (result MovementResult)
}
