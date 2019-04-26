# Skenario

Skenario is a simulator toolkit for Knative, originally created to assist
with Autoscaler development. 

See [the Concepts document](docs/concepts.md) for a discussion of how Skenario is designed.

See "[Implement workload simulator for autoscaler development](https://github.com/knative/serving/issues/1686)"
for background and notes. 

## Web GUI Usage

First, launch the server:

```
$ go run cmd/skenario/main.go
```

Then go to [https://localhost:3000](https://localhost:3000) to see the user interface.

Adjust parameters using the form and click "Execute simulation" to submit the parameters to the server process.
When the simulation is complete, a graph of the results will be displayed.

As with the CLI, the server stores simulation results in `skenario.db`. To suppress this behaviour, add
`?inmemory=true` to the URL.

When you are finished, `Ctrl-C` to kill the running server.

## CLI Usage

Basically:

```
$ go run cmd/skenario/main.go -h

  -duration duration
        Duration of time to simulate. (default 10m0s)

  -maxScaleUpRate float
        Maximum rate the autoscaler can raise its desired (default 10)

  -numberOfRequests uint
        Number of randomly-arriving requests to generate (default 10)

  -panicWindow duration
        Duration of panic window of the Autoscaler (default 6s)

  -rampDelta int
        RPS acceleration/deceleration rate (default 1)

  -rampMaxRPS int
        Max RPS of the ramp traffic pattern. Ignored by uniform pattern (default 50)

  -replicaLaunchDelay duration
        Time it takes a Replica to move from launching to active (default 1s)

  -replicaTerminateDelay duration
        Time it takes a Replica to move from launching or active to terminated (default 1s)

  -scaleToZeroGrace duration
        Duration of the scale-to-zero grace period of the Autoscaler (default 30s)

  -showTrace
        Show simulation trace (default true)

  -sineAmplitude int
        Maximum RPS of the sinusoidal traffic pattern (default 50)

  -sinePeriod duration
        Time between sinusoidal RPS peaks (default 1m0s)

  -stableWindow duration
        Duration of stable window of the Autoscaler (default 1m0s)

  -stepAfter duration
        When using the step traffic pattern, wait this long until the step occurs (default 10s)

  -stepRPS int
        RPS of the step traffic pattern (default 50)

  -storeRun
        Store simulation run results in skenario.db (default true)

  -targetConcurrencyDefault float
        Default target concurrency of Replicas (default 1)

  -targetConcurrencyPercentage float
        Percentage adjustment of target concurrency of Replicas (default 0.5)

  -tickInterval duration
        Tick interval duration of the Autoscaler (default 2s)

  -trafficPattern string
        Traffic pattern. Options are 'uniform', 'ramp', 'step' and 'sinusoidal'. (default "uniform")
```

### Data collection

The configuration and results for each run are collected in `skenario.db`. The schema lives in the `data` package.

The CLI will create a skenario.db file if it doesn't already exist.

### Reading the output

The output has three parts.

1. The header. This shows the totals for number of completed movements, number of ignored movements, clock time and
   simulation time.
1. Movements trace. This is every Movement that occurred during the life of the simulation.
    * "Time" shows the current nanosecond of the simulation. Only one Movement can occur per time.
    * "Movement name" is a simple descriptive title.
    * "From Stock" shows the Stock from which the Entity is being removed.
    * "To Stock" shows the Stock to which the Entity is being added.
    * "Notes" shows annotations added to the Movement by one or more Stocks or Entities. These are mostly empty.
1. Ignored Movements. The Environment did not carry these out during simulation. Most often this is because the
   Movement would have occurred after the simulation ended.
