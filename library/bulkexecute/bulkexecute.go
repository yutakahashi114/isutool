package bulkexecute

import (
	"sync"
	"time"
)

const execNum = 3

type BulkExecute[V any, R any] struct {
	m sync.Mutex

	chans [execNum]chan struct{}
	wgs   [execNum]sync.WaitGroup

	exec func(data []V) (R, error)
	wait time.Duration

	data  []V
	count uint64

	res R
	err error
}

func New[V any, R any](exec func([]V) (R, error), wait time.Duration, cap int) *BulkExecute[V, R] {
	initCh := make(chan struct{}, 1)
	initCh <- struct{}{}
	return &BulkExecute[V, R]{
		chans: [execNum]chan struct{}{
			initCh,
			make(chan struct{}, 1),
			make(chan struct{}, 1),
		},
		exec: exec,
		wait: wait,
		data: make([]V, 0, cap),
	}
}

func (be *BulkExecute[V, R]) Execute(data ...V) (R, error) {
	be.m.Lock()
	be.data = append(be.data, data...)
	current := be.count
	be.m.Unlock()

	be.wgs[current].Add(1)
	defer be.wgs[current].Done()

	_, ok := <-be.chans[current]
	if !ok {
		return be.res, be.err
	}

	time.Sleep(be.wait)

	next := (current + 1) % execNum

	be.m.Lock()
	d := be.data
	be.data = make([]V, 0, cap(d))
	be.count = next
	be.m.Unlock()

	res, err := be.exec(d)
	be.wgs[(current+execNum-1)%execNum].Wait()
	be.res, be.err = res, err

	close(be.chans[current])
	be.chans[current] = make(chan struct{}, 1)
	be.chans[next] <- struct{}{}

	return res, err
}
