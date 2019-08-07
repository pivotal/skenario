# Skenario

Skenario is a simulator toolkit for Knative, originally created to assist
with Autoscaler development. 

See [the Concepts document](docs/concepts.md) for a discussion of how Skenario is designed.

See "[Implement workload simulator for autoscaler development](https://github.com/knative/serving/issues/1686)"
for background and notes. 

| Job | Status |
| ---: | --- |
| Main Tests | [![Tests](http://wings.pivotal.io/api/v1/teams/jchester-knative/pipelines/skenario/jobs/test/badge)](https://wings.pivotal.io/teams/jchester-knative/pipelines/skenario/jobs/test/) |
| PR Tests | [![PRs](http://wings.pivotal.io/api/v1/teams/jchester-knative/pipelines/skenario/jobs/test-pr/badge)](https://wings.pivotal.io/teams/jchester-knative/pipelines/skenario/jobs/test-pr/) |


## Web GUI Usage

First, launch the server:

```
$ go run cmd/skenario/main.go
```

Then go to [https://localhost:3000](https://localhost:3000) to see the user interface.

Adjust parameters using the form and click "Execute simulation" to submit the parameters to the server process.
When the simulation is complete, a graph of the results will be displayed.

The server stores simulation results in `skenario.db`. To suppress this behaviour, add
`?inmemory=true` to the URL.

When you are finished, `Ctrl-C` to kill the running server.
