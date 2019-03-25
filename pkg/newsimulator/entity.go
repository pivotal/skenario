package newsimulator


type EntityName string
type EntityKind string

type Entity interface {
	Name() EntityName
	Kind() EntityKind
}

type entity struct {
	name EntityName
	kind EntityKind
}

func (e *entity) Name() EntityName {
	return e.name
}

func (e *entity) Kind() EntityKind {
	return e.kind
}

func NewEntity(name EntityName, kind EntityKind) Entity {
	return &entity{
		name: name,
		kind: kind,
	}
}
