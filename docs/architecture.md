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

The idea of architecture with plugins is to make autoscaling part be out of the scope of 
the Simulation environment. Skenario could support multiple Implementations without modifying 
the core Simulation Environment. In other words, adding new Implementations would not require 
updating the Simulation Environment.
Plugins are started in a separate process by Hashicorp go-plugin. Communication is done over gRPC. 

##Autoscaler Interface

sk-plugin defines a proto for communication between a simulator and an autoscaler.  
Plugins run out-of-process and therefore can be implemented in any language. 
There are two plugins implementing the sk-plugin protocol. plugin-k8s wraps the HPA controller, 
plugin-k8s-vpa wraps the VPA recommender. Also dispatcher implements the sk-plugin protocol, but it 
is resposible for passing the right data to the right plugin. 

Implementations provide 4 callback functions, 2 input and 2 output.

* (input) 	Event - create, update and delete events for pods etc...
* (input) 	Stat - periodic system stats such as CPU usage or request concurrency.
* (output) 	HorizontalRecommendation - a request for a recommended scale in a horizontal way, given prior input callbacks.
* (output) 	VerticalRecommendation - a request for a recommended scale in a vertical way, given prior input callbacks.

##Dispatcher 

In the whole architecture "dispatcher" has a role of a manager.
Dispatcher is responsible for plugin lifecycle management and communication with it.
The idea is to delegate that work to dispatcher is that we can connect as many plugins as we want
and it does not effect Skenario at all. Skenario just considers dispatcher as a plugin and
communicate with it as with a plugin. All multi-plugable logic is hidden in dispatcher.

Dispatcher knows: 
* which plugins we need to connect and with which configuration
* which plugin we need to send which data

Basically, in Skenario we just say to "dispatcher" HorizontalRecommendation and 
it knows which plugins have this method.   
 
## Skenario and Kubernetes integration

All communication with HPA and VPA is done over plugins. The interesting thing is to 
look through Skenario and Kubernetes integration if we skip plugin's layer.

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

## Horizontal scaling in Skenario

For more information how Skenario scales horizontally see [concepts.md].
Horizontal scaling looks like communication between components. 

Skenario asks dispatcher:
    - Dispatcher, tell me how many pods do I need?
Dispatcher gets a request and delegates it to the HPA plugin:
    - The HPA plugin, tell me how many pods does Skenario need?
The HPA plugin gets a request and delegates it to Kubernetes:     
    - Kubernetes, tell me how many pods does Skenario need?
Kubernetes get a request and gives a response, the number of pods, to the HPA plugin:
    - The HPA plugin, here is the number of pods which Skenario needs.
The HPA plugin gets a response and passes it to the dispatcher:
    - Dispatcher, here is the number of pods which Skenario needs.
Dispatcher gets a response and passes it to Skenario:
    - Skenario, here is the number of pods which you need.

From architecture perspective, the diagram below reflects scaling lifecycle.
                                
```

Skenario ---> Dispatcher ---> HPA plugin ---> Kubernetes

Skenario <--- Dispatcher <--- HPA plugin <--- Kubernetes

```

### HPA plugin

plugin-k8s is the HPA plugin. It wraps the HPA controller. 

## Vertical scaling in Skenario

For more information how Skenario scales vertically see [concepts.md].
Vertical scaling looks like communication between components. 

Skenario asks dispatcher:
    - Dispatcher, tell me which size of pods do I need?
Dispatcher gets a request and delegates it to the VPA plugin:
    - The VPA plugin, tell me which size of pods does Skenario need?
The VPA plugin gets a request and delegates it to Kubernetes:     
    - Kubernetes, tell me which size of pods does Skenario need?
Kubernetes get a request and gives a response, the size (cpu capacity) which is
appropriate for every pod, to the VPA plugin:
    - The VPA plugin, here is the size which is appropriate for every pod in Skenario.
The HPA plugin gets a response and passes it to the dispatcher:
    - Dispatcher, here is the size which is appropriate for every pod in Skenario.
Dispatcher gets a response and passes it to Skenario:
    - Skenario, here is the size which is appropriate for every pod in Skenario.

From architecture perspective, the diagram below reflects scaling lifecycle.
                                
```

Skenario ---> Dispatcher ---> VPA plugin ---> Kubernetes

Skenario <--- Dispatcher <--- VPA plugin <--- Kubernetes

```



### VPA plugin

plugin-k8s-vpa is the VPA plugin. It wraps the VPA recommender.  

## Metrics lifecycle