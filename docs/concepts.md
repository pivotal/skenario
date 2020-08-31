# Skenario Concepts

This document introduces the core concepts and logic of Skenario, a simulator for
HPA.

Skenario is a simulator that borrows ideas from two distinct schools of modeling:
Discrete Event Simulation (DES) and Systems Dynamics (SD).

## The purpose of simulation

The motivation for Skenario is to present the Horizontal Pod Autoscaler with synthetic
inputs that lead to realistic reactions, so that its behaviour can be understood under
various scenarios.

Quoting from the [original issue](https://github.com/knative/serving/issues/1686):

> Autoscaling is a surprisingly difficult problem with a number of nested and
> overlapping problems to solve. In developing solutions, we can aim to validate
> or invalidate our design hypotheses in three main ways:
>
> * Empirically
>     * From production observations
>     * From load & performance testing
> * Theoretically
>     * Applying control theory
>     * Applying queueing theory
> * With simulation
>
> ## Why simulation?
>
> Because each validation approach has different strengths and weaknesses. Empirical
> validation is the final word, but is a slow-moving feedback loop (hours to months).
> Theoretical validation can dramatically shrink the solution search space, but is less
> accessible to each of our key personae (developers, operators, contributors) and does
> not yet address problems that remain unsolved in the research literature.
>
> Simulation splits the difference: it is faster than empirical validation with the risk
> of inaccuracy, simpler than theoretical validation with the cost of implementation.
> Simulation is intended to provide contributors with the ability to rapidly explore the
> design space and iterate on solutions. It is also intended to illuminate potential
> problems in advance of implementation. Simulation will probably provide input into
> autoscaling, routing, serving and upstream projects.

## The concept of time

A Discrete Event Simulation (DES), as the name suggests, updates the simulation based
on messages, commands, objects or some other representation of an "event" that mutates
the state of the simulated world. In Discrete Event Simulation, time is discrete: it
is divided into units that cannot be further subdivided.

By contrast, in Systems Dynamics, time is a continuous variable. There are no
discrete "events", rather there are "Flows" that represents rates of change.

The DES approach enables a key computational shortcut. When an event occurs, the
overall simulation clock can be advanced to that occurrence time. Times _between_
events need not be simulated. To an outside observer the clock "skips" from event
to event. This design is known in DES terminology as a "next-event" simulation.

By contrast, to simulate a continuous system, it will usually be necessary to
numerically solve integral calculus equations. This means iteratively computing
fixed slices of time. In DES terminology this is a "fixed-interval" simulation,
alternatively "continuous-time" simulation.

Being able to skip ahead has performance implications. Simulations of events in
very precise time do not need to be slower than simulations with imprecise time.
Runtime scales with the number of events to be simulated, rather than with the
number of events multiplied by the time-precision of the simulation.

Also convenient is that a DES model can allow events to be scheduled both before
and during the simulation execution. This is particularly useful in setting up
arrivals into the simulation from the "outside". For example, all the Requests
that will arrive during a simulation are added to the schedule before the simulation
scenario begins to execute.

In Skenario, simulation begins at time zero (the Epoch) and changes occur with
nanosecond resolution.

It is important to note that there is no parallelism in the design. In each pass
through the simulation loop, time appears "frozen" until the relevant Movement has
finished processing. Any changes made by participating Stocks and Models will
appear to occur instantaneously. In implementation terms this means I have not
made use of channels and I have not made any efforts towards goroutine safety.

## Core types

There are five major concepts in Skenario: Entities, Stocks, Movements, Models
and the Environment. 

### Entities

An entity is a discrete object that can be "moved". In DES, entities are typically
called "processes".

For example: if Skenario was a bank simulation, customers in a queue would be entities,
as they are individually-identifiable units that enter the bank, join the queue, go
to a teller, then leave the bank.

In Skenario, the major entities are Replicas and Requests. A Replica moves from
launching, to active, to terminating, to terminated. Requests can be at the source,
in the buffer, processing on a replica or completed.

As a design principle, you should place as little logic into an Entity as possible;
most of the logic of interest should be in the Stock which manages them. Think of
Entities as being anonymous.

#### Relationship to the Discrete Event Simulation concept of Process

Mainstream DES literature tends to focus on building a list of events of known types,
a loop that takes the next event from the list and then a switch statement that inspects
the event type and updates a global model. Typical example implementations involve
a collection of global arrays and flag variables (see for example 
[the sample code](http://highered.mheducation.com/sites/0073401323/information_center_view0/programs_from_the_book.html)
given for Averill M. Law's _Simulation Modeling and Analysis, 5th Ed_).

This approach does not treat Entities as individual structures, but instead as
numerical counts to be added and subtracted from variables in the model. In DES
terminology this is the "job-oriented" approach. Each "job" (eg, a simulated bank
teller) is simulated, but the customers are represented merely as a tally that
is added and subtracted from.

The alternative approach is "process-oriented". A "process" (eg, the bank customer)
can "occupy" a "resource" (eg, the bank teller). Instead of gathering statistics on
the jobs, the model collects statistics on the processes. Formally these are
equivalent, one can be converted to the other.

Literature and examples of the process-oriented approach are frustratingly thin
on the ground compared to the job-oriented approach. I suspect this is due to the
long history of DES, dating as it does to the early Fortran era. The process-oriented
approach, by contrast, essentially escaped into the broader community as
object-oriented programming via Simula.

An early design for Skenario attempted a process-oriented approach, where Entities
embedded in Events (called "Subjects" of the event) were responsible for all the
logic of reacting to an event. This broke down because Entities would need to listen
for events of interest, even if those events were not directly in their own scope of
responsibility. For example, a Request would need to listen for the activation of
Replicas, so that it could schedule itself to switch to the RequestProcessing state.

#### Relationship to the Agent-Based Modelling concept of Agent

[Agent-Based Modelling](https://en.wikipedia.org/wiki/Agent-based_model) (ABM) is a
third distinct paradigm of simulation, probably recognisable to many mainstream
programmers as a message-passing design. From ABM, Skenario takes the idea that
Entities may have their own logic. However it does not go as far as ABM in constructing
all system behaviour from the interaction of many agents pursuing their own agenda.

ABM systems are profoundly effective at modelling emergent behaviour of large
populations, but less helpful in understanding the dynamics of movement in more
constrained systems.

### Stocks

Stocks are "where" an entity exists at the beginning of each simulation loop. The
term "Stock" is taken from System Dynamics, where it forms part of the concept of
"Stocks and Flows". A stock is any variable that shows memory or history, that behaves
like an integral or summation.

The example often given is of a bathtub, which is a Stock of water. There is an
flow into the bathtub (the tap) and a flow out of the bathtub (the drain). At any
given point in time, the amount of water in the Stock is a function of the net rates
of flow integrated or summed over the timespan of interest.

Stocks are a central concept in Systems Dynamics because they introduce _delays_.
In Discrete Event Simulation the almost universal concept of a Stock is the Queue,
partly because of the strong Queueing Theory foundations of that field. I have
chosen the System Dynamics terminology for two reasons. One is that it ties more
neatly to "Movement", as described below. The other is that "queue", in a software
context, is typically taken to mean a FIFO queue. But Stocks may, in theory,
perform any mixing of Entities that they wish.

#### `Add()` and `Remove()`

These are the main methods for interacting with a Stock. In general, this is where
most special-case simulation logic should be placed, because these are the methods
called during a Movement.

Note that `Remove()` does allow the caller to select _which_ Entity to remove.
Also the caller is able to omit saying which Entity to remove and receive _an_ Entity, of the Stock's choosing.

These methods are the main extension points for Skenario. Several specialised
Stocks (eg RequestsRoutingStock, TrafficSource) implement simulation logic as part
of their `Add()` or `Remove()` methods. Several of these wrap a simpler delegate
Through stock.

#### Source Stocks, Sink Stocks and Through Stocks

Stocks need not always have both of `Add()` and `Remove()`.

A Stock which only has a `Remove()` method is a "Source" Stock. The typical pattern
for these is to be responsible for generation a new Entity. For example, the
TrafficSource Stock creates new Requests each time its `Remove()` method is called.

A Stock which only has an `Add()` method is a "Sink" Stock. These are typically
intended to prevent further movements of an Entity. An example is the
ReplicasTerminated Stock: once an Entity reaches it, it should never be able to
leave.

A Stock with both `Add()` and `Remove()` is a Through Stock.
There are two implementations of the generic through stock. One based on an array 
providing O(1) list operations (e.g. for round-robin load balancing on Replicas) 
and one based on a map provide O(n) list operations, but O(1) Add and Remove (e.g. for Request entities).

### Movements

Movements are the main substitute for "events" in the DES meaning of the term. The
core simulation loop iterates over Movements that have been Scheduled.

Each Movement has four key values: Kind, OccursAt, From and To, and one optional value: WhatToMove. 
The Kind is useful to group together Movements with different particulars. OccursAt is the point in
simulation time at which the Movement is intended to occur. The "From" and "To"
fields point to particular Stocks that the Environment will Remove() from and
Add() to. WhatToMove is a reference to a particular Entity. If we are intended 
to highlight which Entity we move (e.g. when we remove a specific Replica as a part of 
updating Replicas to stick to vertical scaling recommendations), we use "WhatToMove" field 
to point to a particular Entity. We can omit this field and the responsibility for 
selecting which Entity to move rests with the Source Stock.  

#### Relationship to the Discrete Event Simulation concept of Events

Why not events? In the early designs for Skenario, events were the main organising
concept and were used to drive finite-state machines for various simulated Entities.
But this turned out to be problematic for two reasons.

The first was that logic quickly became very tangled, with various simulated
objects listening for events being scheduled or executed by other objects. The core
of each simulated Entity became a large, hairy switch statement.

The second problem was that FSMs and events could not easily represent that Replicas
are created and destroyed during the life of the simulation. The assignment of Requests
to Replicas is critical to provoking a realistic response from the Autoscaler.
It is not enough to have an FSM representing "processing" as a state, it has to
represent "processing on replica-4" as a state. But the existence and reachability
of this state is contingent on the existence or non-existence of Replicas. This
was easier to represent using Stocks and Movements that deal in Entities.

#### Relationship to the Systems Dynamics concept of Flows

In System Dynamics there's no "movement" concept and generally Entities are
represented by a count or summation in a Stock. A Stock in System Dynamics is
connected to other Stocks via Flows. Like Stocks these are functions of time
elapsed, but they are derivatives, not integrals: rates of change, rather than
accumulated amounts of change.

This makes sense in a continuous setting, but does not make sense in a discrete
setting. In this respect Movements can be thought of as discretized Flows, or
alternatively as the inverse of a Flow. Where a Flow might expressed "X units
per second", a Movement will instead look at the seconds until a particular
Movement is to occur. When aggregated by time, Movements approximate a Flow.

### Models

Models are "the rest" of the code. Typically these own Stocks, wire dependencies and
potentially maintain other state.

In Skenario the Models provided are the Autoscaler and the Cluster. These
establish the various initial Stocks (eg. RequestsRouting, ReplicasLaunching) that are
used during the life of the simulation.

Generally speaking, logic should be kept out of Models. Everything that can be expressed
as a Movement ought to be; special-case logic should be placed in `Add()` or
`Remove()`. For example, the logic for scaling up or down is expressed mostly as
Movements between Stocks, rather than as variables manipulated by the Models.

### Environment

The Environment is the effective root of the program. Its `Run()` method has the
core simulation loop and the `AddToSchedule()` method provides the means for
simulation components to set up future Movements. The Environment also provides a
number of accessors to key global variables: the simulation's `CurrentMovementTime()`,
the simulation's `HaltTime()` and a `Context()` that holds a logger object as a value.

The Environment schedules two specialised Movements: the `start_to_running` Movement
and the `running_to_halted` Movement. These are placed at the boundary points of time.
That is: only the start Movement may occur at time zero, only the halt Movement may
occur at the halting time. Other Movements must occur between these two, meaning that
the permissible time range is expressible as `(0, halt)` or as `[1, halt-1]`.

## How a scenario is executed

### `Run()`

The Environment's `Run()` method implements the "next-event list" concept from
Discrete Event Simulation. This means that on each iteration, the Environment will
select the next Movement from a queue and execute it.

Ordering is by the `OccursAt` time of Movements. Internally, the Environment is
relying on a `MovementPriorityQueue` to maintain orderly records; ultimately this is
relies on [an ordered heap](https://godoc.org/k8s.io/client-go/tools/cache#Heap) to
maintain Movement ordering. Ordering is strict and total: only one Movement can occupy
any given `OccursAt` time.

Once a Movement has been dequeued, the simulation's current time is advanced to
the `OccursAt` value of the Movement. The Environment then calls the `Remove()` method
of the `From()` Stock to retrieve an Entity. If that call is successful, it then
`Add()`s that Entity to the `To()` stock.

On each iteration, Movements that occurred successfully are captured in the
`CompletedMovements` array for later display. Movements that were not successful, most
often because of an empty `From()` stock, are captured in the `IgnoredMovements`
instead.

### `AddToSchedule()`

This method is how new Movements are scheduled for simulation. Any object with a
reference to the Environment may call this method. It is the sole mechanism for
inserting elements into the queue of Movements.

The Environment will only accept Movements which will occur during the remaining life
of the simulation. This means it will reject Movements scheduled before the current
time and it will reject events that would occur after the halt time. Such Movements
are added to the `IgnoredMovements` array.

The Environment also rejects new Movements when an existing Movement is already
scheduled to `OccursAt` that time. This is partly due to the underlying implementation,
which uses the `OccursAt` value as the key for ordering of Movements. This underlying
data structure allows existing values for a given key to be modified, meaning that
without guarding against scheduling collisions, Movements could be overwritten without
warning.

The design also reflects that events -- Movements -- are _discrete_. One and only one
change to the world can occur at a time. Put another way: the simulation is intended to
be strictly deterministic.

In practical terms, the fact that the Environment will ignore Movements that have
schedule collisions with another Movement means that some logic is needed in callers to
reduce the odds of collision. In future I think it will be better for the Environment
to handle the schedule-shifting itself.

For debugging purposes, the CLI shows a table of ignored Movements and the reason why
they were ignored.

### Example: Autoscaler Ticktock

In its natural environment, the Horizontal Pod Autoscaler (HPA) and The Vertical Pod Autoscaler (VPA) 
are triggered on a `TickInterval`, defaulting to 15 seconds. Upon each `TickInterval` HPA recalculates 
its desired number of replicas while VPA updates recommendations regarding size of replicas
 in terms of cpu capacity.

The `AutoscalerTicktockStock` is used to manage this regular behaviour. At creation
time, Movements from the `AutoscalerTicktockStock` back into itself are scheduled,
so that `AutoscalerTicktockStock` is both of the `From()` and `To()` stocks in the
Movements. On each `Add()` the stock will drive the actual HPA and VPA, prompting it to
calculate new desired values.

### Example: Metrics Ticktock
//TODO 
### Example: Replicas

Replicas are the unit that the HPA and VPA are scaling up and down (HPA in terms of quantity, 
VPA - size). The responsiveness of the overall system depends in no small part on 
how quickly Replicas can become active and able to process incoming traffic. 
The Movements graph for Replicas is:

```
                           +--------------------------------------+
                           |                                      |
                           |                                      V         
ReplicaSource --> ReplicasLaunching --> ReplicasActive --> ReplicasTerminated
```

The diagram shows four possible transitions:

* `begin_launch`, from ReplicaSource to ReplicasLaunching
* `finish_launching`, from ReplicasLaunching to ReplicasActive
* `terminate_launch`, from ReplicasLaunching to ReplicasTerminated
* `terminate_active`, from ReplicasActive to ReplicasTerminated

Time spent in the ReplicasLaunching stock is influential on system behaviour because
Requests can continue to arrive while launching takes place. In the original designs
this was substantially more detailed; for this generation of the design I chose to
ignore the details while rebuilding the core simulator framework. A lot of improvements
to simulation accuracy will probably come from breaking that Stock into finer detail. 

Replicas are represented with the `ReplicaEntity`, a specialisation of Entity. The
specialisation holds logic necessary to activate and deactivate a replica in the Kubernetes.

### Example: Requests

Indirectly, Requests are the signal that the Horizontal Pod Autoscaler and the Vertical 
Pod Autoscaler are trying to respond to. In a traditional simulation these are called 
"arrivals". Unlike traditional simulation, the configuration of the simulated system varies 
throughout the simulation as the HPA changes its desired Replica count.

The Movements graph for Requests is:
 
```

TrafficSource -+-> RequestsRouting  -+-> RequestsProcessing -+-> RequestsComplete
                                                             |       
                                                             +-> RequestsFailed
```

The diagram shows five possible Movements:

* `arrive_at_routing`, from TrafficSource to RequestsRouting. These are scheduled
  during the creation of the simulation
* `send_to_replica`, from RequestsRouting to RequestsProcessing
* `complete_request`, from RequestsProcessing to RequestsComplete
* `request_failed`, from RequestsProcessing to RequestsFailed

The happy path (`arrive_at_routing`, `send_to_replica`, `complete_request`) is linear.
However, this is true only in aggregate: the exact path for each Request can vary. While
all Requests will come from the TrafficSource and all of them will spend one
Movement in RequestsRouting, the precise RequestsProcessing / RequestsComplete stocks
are not known when the original Request arrival is scheduled. There are multiple
RequestsProcessing stocks, each belonging to a Replica.

There are two alternative paths.

Initially, if a Request arrives, it's sent to RequestRouting. From RequestRouting we schedule a Movement to 
ReplicasActive stock. On each such Movement it picks a replica with a round-robin scheme and
schedule a Movement of the Request to its RequestsProcessing. The RequestsProcessing
stock will itself schedule a Movement into RequestsComplete or RequestsFailed, representing timeouts.

This is probably the second major influence on Autoscaler behaviour. By
[Little's Law](http://web.mit.edu/~sgraves/www/papers/Little%27s%20Law-Published.pdf),
a longer time to process a Request will mean that more Requests are being processed at
any given time. The problem is that this signal itself takes time to build up,
especially if processing time is very long but arrivals are infrequent.

This is a good example of the System Dynamics principle that Stocks create delays and
that these delays can lead to counter-intuitive non-linear dynamics.
