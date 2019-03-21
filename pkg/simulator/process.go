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
	AddStock(item Stockable)
	RemoveStock(item Stockable)
}

type Stockable interface {
	Identifiable
	OnMovement(movement StockMovementEvent) (result MovementResult)
	CurrentlyAt() Stock
}
