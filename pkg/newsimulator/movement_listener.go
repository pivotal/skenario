package newsimulator

type MovementListener interface {
	OnMovement(movement Movement) error
}

