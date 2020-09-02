# Skenario Architecture

This document introduces the architecture of Skenario, a simulator for
HPA and VPA.

Skenario is an open-source simulator and interface for the HPA and VPA autoscalers.
Skenario provides a simulated cluster with a web UI.  The user configures an HPA or VPA object, 
selects a predefined traffic profile and receives a collection of metrics which are plotted on 
the screen showing what happened during the simulation.

##Terms

* `Implementation` - the autoscaling system under test.
* `Scenario` - a set of parameters which include input traffic, cluster configuration and limits, 
  and an Implementation under test.
* `Simulation Environment (or just Simulation)` - the core machinery, instantiated once per Scenario.

## Architecture diagram

```
                           Plugin hpa --- Kubernetes
                         /                  /horizontal.go
Skenario <--> Dispatcher 
                         \ 
                          Plugin vpa --- Autoscaler
                                          /recommender.go 

```

* `Skenario` - the core Simulation Environment
* `Dispatcher` - responsible for plugin management
* `Plugin vpa, Plugin hpa` - autoscalers wrapped in order to implement the autoscaler interface 
  and to be driven deterministically by an injected clock


The idea of architecture with plugins is to make autoscaling part be out of the scope of 
the Simulation environment. Skenario could support multiple Implementations without modifying 
the core Simulation Environment.

### Skenario layering

