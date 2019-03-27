package main

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

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

	printer := message.NewPrinter(language.AmericanEnglish)
	sb := new(strings.Builder)

	sb.WriteString("=== BEGIN TRACE ================================================================================================================================================\n")
	sb.WriteString("---------------------------------------------------------------------[ completed movements ]--------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %s\n", "TIME (ns)", "MOVEMENT NAME", "FROM STOCK", "TO STOCK", "NOTE"))
	sb.WriteString("----------------------------------------------------------------------------------------------------------------------------------------------------------------\n")

	for _, c := range completed {
		mv := c.Movement
		sb.WriteString(printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %s\n",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			mv.Note(),
		))
	}

	sb.WriteString("\n")
	sb.WriteString("----------------------------------------------------------------------[ ignored movements ]---------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %-28s %-35s\n", "TIME (ns)", "MOVEMENT NAME", "FROM STOCK", "TO STOCK", "NOTE", "REASON IGNORED"))
	sb.WriteString("----------------------------------------------------------------------------------------------------------------------------------------------------------------\n")
	for _, i := range ignored {
		mv := i.Movement
		sb.WriteString(printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %-28s %-35s\n",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			mv.Note(),
			i.Reason,
		))
	}

	sb.WriteString("=== END TRACE ==================================================================================================================================================\n")

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
