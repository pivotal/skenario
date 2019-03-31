/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"knative-simulator/pkg/model"

	"knative-simulator/pkg/simulator"
)

var (
	startAt                     = time.Unix(0, 0)
	startRunning                = time.Now()
	au                          = aurora.NewAurora(true)
	simDuration                 = flag.Duration("duration", 10*time.Minute, "Duration of time to simulate.")
	tickInterval                = flag.Duration("tickInterval", 2*time.Second, "Tick interval duration of the Autoscaler")
	stableWindow                = flag.Duration("stableWindow", 60*time.Second, "Duration of stable window of the Autoscaler")
	panicWindow                 = flag.Duration("panicWindow", 6*time.Second, "Duration of panic window of the Autoscaler")
	scaleToZeroGrace            = flag.Duration("scaleToZeroGrace", 30*time.Second, "Duration of the scale-to-zero grace period of the Autoscaler")
	targetConcurrencyDefault    = flag.Float64("targetConcurrencyDefault", 1.0, "Default target concurrency of Replicas")
	targetConcurrencyPercentage = flag.Float64("targetConcurrencyPercentage", 0.5, "Percentage adjustment of target concurrency of Replicas")
	maxScaleUpRate              = flag.Float64("maxScaleUpRate", 10.0, "Maximum rate the autoscaler can raise its desired")
)

func main() {
	flag.Parse()
	r := NewRunner()

	cluster := model.NewCluster(r.Env())
	cluster.SetDesired(1)
	model.NewKnativeAutoscaler(r.Env(), startAt, cluster, r.AutoscalerConfig())

	err := r.RunAndReport(os.Stdout)
	if err != nil {
		fmt.Printf("there was an error during simulation: %s", err.Error())
	}
}

type Runner interface {
	Env() simulator.Environment
	AutoscalerConfig() model.KnativeAutoscalerConfig
	RunAndReport(writer io.Writer) error
}

type runner struct {
	env simulator.Environment
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
		simDuration.String(),
	)

	printer := message.NewPrinter(language.AmericanEnglish)
	fmt.Fprintln(writer, au.BgGreen(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %-58s", "Time (ns)", "Movement Name", "From Stock", "To Stock", "Notes")).Bold())

	for _, c := range completed {
		mv := c.Movement
		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			strings.Join(mv.Notes(), "\n                                                                                                      "),
		))
	}

	fmt.Fprint(writer, "\n")
	fmt.Fprintln(writer, au.BgBrown(fmt.Sprintf("%20s  %-24s %-24s ⟶   %-24s  %-28s %-29s", "Time (ns)", "Movement Name", "From Stock", "To Stock", "Notes", "Reason Ignored")).Bold())
	for _, i := range ignored {
		mv := i.Movement

		coloredReason := ""
		switch i.Reason {
		case simulator.OccursInPast:
			coloredReason = au.Red(i.Reason).String()
		case simulator.OccursAfterHalt:
			coloredReason = au.Magenta(i.Reason).String()
		case simulator.OccursSimultaneouslyWithAnotherMovement:
			coloredReason = au.Cyan(i.Reason).String()
		}

		fmt.Fprintln(writer, printer.Sprintf(
			"%20d  %-24s %-24s ⟶   %-24s  %-28s %-29s",
			mv.OccursAt().UnixNano(),
			mv.Kind(),
			mv.From().Name(),
			mv.To().Name(),
			strings.Join(mv.Notes(), "\n                                                                                                      "),
			coloredReason,
		))
	}
	fmt.Fprint(writer, "\n")

	return nil
}

func (r *runner) Env() simulator.Environment {
	return r.env
}

func (r *runner) AutoscalerConfig() model.KnativeAutoscalerConfig {
	return model.KnativeAutoscalerConfig{
		TickInterval:                *tickInterval,
		StableWindow:                *stableWindow,
		PanicWindow:                 *panicWindow,
		ScaleToZeroGracePeriod:      *scaleToZeroGrace,
		TargetConcurrencyDefault:    *targetConcurrencyDefault,
		TargetConcurrencyPercentage: *targetConcurrencyPercentage,
		MaxScaleUpRate:              *maxScaleUpRate,
	}
}

func NewRunner() Runner {
	return &runner{
		env: simulator.NewEnvironment(startAt, *simDuration),
	}
}
