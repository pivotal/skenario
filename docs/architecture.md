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
* `Autoscaler Interface` - the interface over which the controller (autoscaler) passes input parameters 
  and over which the simulation passes back output metrics.
* `Plugins` - autoscalers wrapped in order to implement the autoscaler interface and to be driven 
  deterministically by an injected clock.

## Skenario architecture diagram

```   
            
                            plugin-k8s --- Kubernetes
                sk-plugin /                  /horizontal.go
         sk-plugin       / 
simulator <--> dispatcher 
                         \ 
                sk-plugin \
                          plugin-k8s-vpa --- Autoscaler
                                               /recommender.go 

```

* `simulator` - the core Simulation Environment
* `dispatcher` - responsible for plugin management
* `plugin-k8s, plugin-k8s-vpa` - autoscalers (hpa and vpa)wrapped in order to implement 
  the autoscaler interface and to be driven deterministically by an injected clock
* `sk-plugin` - Autoscaler Interface 

The idea of architecture with plugins is to make the autoscaling part be out of the scope 
Simulation environment. Skenario could support multiple Implementations without modifying 
the core Simulation Environment. In other words, adding new Implementations would not require 
updating the Simulation Environment.
Plugins are started in a separate process by Hashicorp go-plugin. Communication is done over gRPC. 

##Autoscaler Interface

sk-plugin defines a proto for communication between a simulator and an autoscaler.  
Plugins run out-of-process and therefore can be implemented in any language. 
There are two plugins implementing the sk-plugin protocol. plugin-k8s wraps the HPA controller, 
plugin-k8s-vpa wraps the VPA recommender. Also, dispatcher implements the sk-plugin protocol, but it 
is responsible for passing the right data to the right plugin. 

Implementations provide 4 callback functions, 2 input and 2 output.

* (input) 	Event - create, update and delete events for pods etc...
* (input) 	Stat - periodic system stats such as CPU usage or request concurrency.
* (output) 	HorizontalRecommendation - a request for a recommended scale in a horizontal way, given prior input callbacks.
* (output) 	VerticalRecommendation - a request for a recommended scale in a vertical way, given prior input callbacks.

### Event

The Event callback informs the Implementation about the state of the simulated cluster. Each event is scoped to an environment id. 
Every Scenario begins with the instantiation of a Simulation Environment with a unique environment id. The first event will always be CREATE Autoscaler.
And the last event will always be DELETE Autoscaler.

The CREATE Autoscaler event includes an opaque (to the Simulation) YAML blob and a string type for convenience.
This should be a meaningful Kubernetes object which the Implementation can use to configure the environment's autoscaler for testing.
E.g. Kubernetes would accept a HorizontalPodAutoscaler with a type of "hpa.v2beta2.autoscaling.k8s.io" or 
a VerticalPodAutoscaler with a type of "vpa.v1.autoscaling.k8s.io"(example).

The CREATE Pod event will provide basic resource request and state information on a pod in the simulated cluster.

### Stat
The Stat callback informs the Implementation about system statistics such as pod CPU usage and request concurrency. 
The Scenario includes a parameter that determines how often to send stats.

### HorizontalRecommendation

The HorizontalRecommendation callback requests a desired number of pods from the Implementation. 
The Scenario will include a parameter that determines how often to request a recommendation.

### VerticalRecommendation

The HorizontalRecommendation callback requests the desired size for pods from the Implementation. 
The Scenario will include a parameter that determines how often to request a recommendation.

##Dispatcher 

In the whole architecture "dispatcher" has the role of a manager.
The dispatcher is responsible for plugin lifecycle management and communication with it.
The idea is to delegate that work to the dispatcher is that we can connect as many plugins as we want
and it does not affect Skenario at all. Skenario just considers the dispatcher as a plugin and
communicate with it as with a plugin. All multi-pluggable logic is hidden in the dispatcher.

Dispatcher knows: 
* which plugins we need to connect and with which configuration
* which plugin we need to send which data

Basically, in Skenario we just say to "dispatcher" HorizontalRecommendation and 
it knows which plugins have this method.   
 
## Skenario and Kubernetes integration

All communication with HPA and VPA is done over plugins. The interesting thing is to 
look through Skenario and Kubernetes integration if we skip the plugin's layer.

```
Skenario                            Kubernetes
                                                            scale
           create/delete            > HPA Object   <-----------| scale
Event ---------------------         > VPA Object   <----------------| 
                                    > Pods                     |    |
                                                               |    |
                                                               |    |
          update                                               |    | 
Stat ----------------------         > Metrics Client           |    |
                                                               |    |
                                                               |    | 
HorizontalRecommendation --         > reconcileAutoscaler() ---     |
                                                                    |
                                                                    |
                                                                    |
VerticalRecommendation ----         > runOnce() --------------------

```
The diagram above shows how Skenario communicates with Kubernetes at a high level.

## Metrics lifecycle

The Scenario includes a parameter that determines how often to send statistics.
System statistics such as pod CPU usage and request concurrency go the following way
    - Skenario asks pods to give metrics
    - Skenario passes metrics per pod to the dispatcher
    - Dispatcher passes metrics to plugins
    - Plugins pass metrics to the autoscalers

## Horizontal scaling in Skenario

For more information how Skenario scales horizontally see [concepts.md].
Horizontal scaling looks like communication between components. 

Skenario asks dispatcher:
    - Dispatcher, tell me how many pods do I need?
The dispatcher gets a request and delegates it to the HPA plugin:
    - The HPA plugin, tell me how many pods does Skenario need?
The HPA plugin gets a request and delegates it to Kubernetes:     
    - Kubernetes, tell me how many pods does Skenario need?
Kubernetes gets a request and gives a response, the number of pods, to the HPA plugin:
    - The HPA plugin, here is the number of pods that Skenario needs.
The HPA plugin gets a response and passes it to the dispatcher:
    - Dispatcher, here is the number of pods that Skenario needs.
The dispatcher gets a response and passes it to Skenario:
    - Skenario, here is the number of pods that you need.

From an architecture perspective, the diagram below reflects the scaling lifecycle.
                                
```

Skenario ---> Dispatcher ---> HPA plugin ---> Kubernetes

Skenario <--- Dispatcher <--- HPA plugin <--- Kubernetes

```

### HPA plugin

plugin-k8s is the HPA plugin. The HPA plugin creates a HorizontalController 
for each Simulation Environment. It keeps track of pods and stats per environment 
and provides them to the controller via mocks and fakes, injected at construction.

Instead of starting the controller (which runs goroutines) the plugin drives 
the controller by calling the reconcileAutoscaler method with the current HPA object. 
The controller updates the HPA object through a reactor on the fake client.
 
## Vertical scaling in Skenario

For more information how Skenario scales vertically see [concepts.md].
Vertical scaling looks like communication between components. 

Skenario asks dispatcher:
    - Dispatcher, tell me which size of pods do I need?
The dispatcher gets a request and delegates it to the VPA plugin:
    - The VPA plugin, tell me which size of pods does Skenario need?
The VPA plugin gets a request and delegates it to Kubernetes:     
    - Kubernetes, tell me which size of pods does Skenario need?
Kubernetes gets a request and gives a response, the size (cpu capacity) that is
appropriate for every pod, to the VPA plugin:
    - The VPA plugin, here is the size which is appropriate for every pod in Skenario.
The HPA plugin gets a response and passes it to the dispatcher:
    - Dispatcher, here is the size which is appropriate for every pod in Skenario.
The dispatcher gets a response and passes it to Skenario:
    - Skenario, here is the size which is appropriate for every pod in Skenario.

From an architecture perspective, the diagram below reflects the scaling lifecycle.
                                
```

Skenario ---> Dispatcher ---> VPA plugin ---> Kubernetes

Skenario <--- Dispatcher <--- VPA plugin <--- Kubernetes

```

### VPA plugin

plugin-k8s-vpa is the VPA plugin. As vertical scaling is more complicated 
than horizontal we are interested only in the recommender component and 
ignore other parts and simulate their work in Skenario.
The VPA plugin creates a Recommender for each Simulation Environment. 
It keeps track of pods and stats per environment and provides them to the recommender
via mocks and fakes, injected at construction.

The plugin drives the recommender by calling the runOnce method with the current VPA object. 
The recommender updates the VPA object through a reactor on the fake client.  