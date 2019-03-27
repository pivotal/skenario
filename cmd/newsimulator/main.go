package main

import (
	"fmt"
	"strings"
	"time"

	"knative-simulator/pkg/newmodel"

	"knative-simulator/pkg/newsimulator"
)

var startAt = time.Unix(0, 0)

func main() {
	r := NewRunner()

	newmodel.NewKnativeAutoscaler(r.Env(), startAt)

	report, err := r.RunAndReport()
	if err != nil {
		fmt.Printf("there was an error during simulation: %s", err.Error())
	}

	fmt.Printf("%s\n", report)
}

type Runner interface {
	Env() newsimulator.Environment
	RunAndReport() (string, error)
}

type runner struct {
	env newsimulator.Environment
}

func (r *runner) RunAndReport() (string, error) {
	completed, ignored, err := r.env.Run()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("completed:\n")
	for _, c := range completed {
		mv := c.Movement
		sb.WriteString(fmt.Sprintf("%d %s %s %s\n", mv.OccursAt().UnixNano(), mv.From().Name(), mv.To().Name(), mv.Note()))
	}

	sb.WriteString("ignored:\n")
	for _, i := range ignored {
		mv := i.Movement
		sb.WriteString(fmt.Sprintf("%s %d %s %s %s\n", i.Reason, mv.OccursAt().UnixNano(), mv.From().Name(), mv.To().Name(), mv.Note()))
	}

	return sb.String(), nil
}

func (r *runner) Env() newsimulator.Environment {
	return r.env
}

func NewRunner() Runner {
	return &runner{
		env: newsimulator.NewEnvironment(time.Unix(0,0), 10*time.Minute),
	}
}
