package simulator

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/cache"
)

type Environment struct {
	futureEvents *cache.Heap
}

func NewEnvironment() *Environment {
	heap := cache.NewHeap(func(event interface{}) (key string, err error) {
		evt := event.(*Event)
		return strconv.FormatInt(evt.Time.UnixNano(), 10), nil
	}, func(leftEvent interface{}, rightEvent interface{}) bool {
		l := leftEvent.(*Event)
		r := rightEvent.(*Event)

		return l.Time.Before(r.Time)
	})

	return &Environment{
		futureEvents: heap,
	}
}

func (env *Environment) Run(ctx context.Context) {
	go func() {
		for {
			nextIface, err := env.futureEvents.Pop() // blocks until there is stuff to pop
			if err != nil && strings.Contains(err.Error(), "heap is closed"){
				return
			} else if err != nil {
				panic(err)
			}

			next := nextIface.(*Event)
			next.AdvanceFunc(next.Time, next.Description)
		}
	}()

	select {
	case <- ctx.Done():
		env.futureEvents.Close()
		fmt.Println("Done.")
		return
	}
}

func (env *Environment) AddProcess(name string) *Process {
	proc := &Process{
		Name:        name,
		Environment: env,
	}

	return proc
}

func (env *Environment) Schedule(event *Event) {
	err := env.futureEvents.Add(event)
	if err != nil {
		panic(err)
	}
}
