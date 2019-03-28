package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/logrusorgru/aurora"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"knative-simulator/pkg/newmodel"

	"knative-simulator/pkg/newsimulator"
)

var startAt = time.Unix(0, 0)
//var runFor = 10000*time.Hour
var runFor = 10 * time.Second
var startRunning = time.Now()
var au = aurora.NewAurora(true)

func main() {
	r := NewRunner()

	newmodel.NewKnativeAutoscaler(r.Env(), startAt)

	err := r.RunAndReport(os.Stdout)
	if err != nil {
		fmt.Printf("there was an error during simulation: %s", err.Error())
	}
}

type Runner interface {
	Env() newsimulator.Environment
	RunAndReport(writer io.Writer) error
}

type runner struct {
	env newsimulator.Environment
}

func (r *runner) RunAndReport(writer io.Writer) error {
	fmt.Fprint(writer, "Running simulation ... ")

	completed, ignored, err := r.env.Run()
	if err != nil {
		return err
	}

	fmt.Fprintf(writer,
		"%5s      %19s %-8d  %17s %-8d  %20s %-10s    %20s %-12s\n\n",
		au.Bold("Done."),
		au.BgGreen("Completed movements"),
		au.Bold(len(completed)),
		au.BgBrown("Ignored movements"),
		au.Bold(len(ignored)),
		au.Cyan("Running time:"),
		time.Now().Sub(startRunning).String(),
		au.Cyan("Simulated time:"),
		runFor.String(),
	)

	printer := message.NewPrinter(language.AmericanEnglish)
	fmt.Fprintln(writer, au.BgGreen(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %-58s","Time (ns)", "Movement Name", "From Stock", "To Stock", "Note")))

	for _, c := range completed {
		mv := c.Movement
		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			mv.Note(),
		))
	}

	fmt.Fprint(writer, "\n")
	fmt.Fprintf(writer, "%s\n", au.Bold("Ignored Movements"))
	fmt.Fprintln(writer, au.BgBrown(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %-28s %-29s", "Time (ns)", "Movement Name", "From Stock", "To Stock", "Note", "Reason Ignored")))
	for _, i := range ignored {
		mv := i.Movement

		coloredReason := ""
		switch i.Reason {
		case newsimulator.OccursInPast:
			coloredReason = au.Red(i.Reason).String()
		case newsimulator.OccursAfterHalt:
			coloredReason = au.Magenta(i.Reason).String()
		case newsimulator.OccursSimultaneouslyWithAnotherMovement:
			coloredReason = au.Cyan(i.Reason).String()
		}

		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %-28s %-29s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			mv.Note(),
			coloredReason,
		))
	}
	fmt.Fprint(writer, "\n")

	return nil
}

func (r *runner) Env() newsimulator.Environment {
	return r.env
}

func NewRunner() Runner {
	return &runner{
		env: newsimulator.NewEnvironment(startAt, runFor),
	}
}
