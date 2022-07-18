package asyncexecute

import (
	"sync"
	"time"
)

type AsyncExecute[V any] struct {
	m    sync.Mutex
	data []V
	exec func(data []V)
}

func New[V any](exec func([]V), wait time.Duration, cap int) *AsyncExecute[V] {
	aq := &AsyncExecute[V]{
		data: make([]V, 0, cap),
		exec: exec,
	}
	go aq.execute(wait)
	return aq
}

func (aq *AsyncExecute[V]) Set(data ...V) {
	aq.m.Lock()
	aq.data = append(aq.data, data...)
	aq.m.Unlock()
}

func (aq *AsyncExecute[V]) get() []V {
	aq.m.Lock()
	data := aq.data
	aq.data = make([]V, 0, cap(data))
	aq.m.Unlock()
	return data
}

func (aq *AsyncExecute[V]) execute(wait time.Duration) {
	c := time.NewTicker(wait)
	defer c.Stop()
	for {
		data := aq.get()
		if len(data) > 0 {
			aq.exec(data)
		}
		<-c.C
	}
}
