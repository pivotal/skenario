package main

import (
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"

	"knative-simulator/pkg/newsimulator"
)

func TestCmdMain(t *testing.T) {
	spec.Run(t, "cmd main", testMain, spec.Report(report.Terminal{}))
}

func testMain(t *testing.T, describe spec.G, it spec.S) {
	var subject Runner
	var ignoredMovement newsimulator.Movement
	var from, to newsimulator.ThroughStock

	it.Before(func() {
		subject = NewRunner()
		from = newsimulator.NewThroughStock("test from stock", "test kind")
		to = newsimulator.NewThroughStock("test to stock", "test kind")
		ignoredMovement = newsimulator.NewMovement(time.Now(), from, to, "ignored movement")

		subject.Env().AddToSchedule(ignoredMovement)
	})

	describe("RunAndReport()", func() {
		var rpt string
		var err error

		it.Before(func() {
			rpt, err = subject.RunAndReport()
			assert.NoError(t, err)
		})
		it("prints completed", func() {
			assert.Contains(t, rpt, "BeforeScenario")
			assert.Contains(t, rpt, "RunningScenario")
			assert.Contains(t, rpt, "HaltedScenario")
		})

		it("prints ignored", func() {
			assert.Contains(t, rpt, "ignored movement")
		})
	})

	describe("NewRunner()", func() {
		it("has an Environment", func() {
			assert.NotNil(t, subject.Env())
		})
	})
}
