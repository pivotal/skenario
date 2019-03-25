package newsimulator

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestEntity(t *testing.T) {
	suite := spec.New("Entity suite", spec.Report(report.Terminal{}))
	suite("Entity", testEntity)

	suite.Run(t)
}

func testEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject Entity

	it.Before(func() {
		subject = NewEntity("test entity name", "test entity kind")
	})

	it("creates an entity", func() {
		assert.Equal(t, subject.Name(), "test entity name")
		assert.Equal(t, subject.Kind(), "test entity kind")
	})
}