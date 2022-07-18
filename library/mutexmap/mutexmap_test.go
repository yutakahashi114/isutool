package mutexmap

import (
	"strconv"
	"sync"
	"testing"
)

func Test_RWMutexMap(t *testing.T) {
	mm := NewRW[string](10)
	wg := sync.WaitGroup{}
	incrMap0 := make(map[int]int, 1)
	incrMap1 := make(map[int]int, 1)
	incrMap2 := make(map[int]int, 1)
	for i := 0; i < 10002; i++ {
		wg.Add(1)
		go func(i int) {
			key := strconv.Itoa(i % 3)
			var incrMap map[int]int
			switch key {
			case "0":
				incrMap = incrMap0
			case "1":
				incrMap = incrMap1
			case "2":
				incrMap = incrMap2
			}
			if i%2 != 0 {
				mm.RLock(key)
				_ = incrMap[0]
				mm.RUnlock(key)
			} else {
				mm.Lock(key)
				incrMap[0] = incrMap[0] + 1
				mm.Unlock(key)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	if incrMap0[0] != 1667 {
		t.Errorf("incrMap0[0] want %v, got %v", 1667, incrMap0[0])
	}
	if incrMap1[0] != 1667 {
		t.Errorf("incrMap1[0] want %v, got %v", 1667, incrMap1[0])
	}
	if incrMap2[0] != 1667 {
		t.Errorf("incrMap2[0] want %v, got %v", 1667, incrMap2[0])
	}
}

func Benchmark_RWMutexMap(b *testing.B) {
	mm := NewRW[string](10)
	wg := sync.WaitGroup{}
	incrMap0 := make(map[int]int, 1)
	incrMap1 := make(map[int]int, 1)
	incrMap2 := make(map[int]int, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			key := strconv.Itoa(i % 3)
			var incrMap map[int]int
			switch key {
			case "0":
				incrMap = incrMap0
			case "1":
				incrMap = incrMap1
			case "2":
				incrMap = incrMap2
			}
			if i%2 != 0 {
				mm.RLock(key)
				_ = incrMap[0]
				mm.RUnlock(key)
			} else {
				mm.Lock(key)
				incrMap[0] = incrMap[0] + 1
				mm.Unlock(key)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}
