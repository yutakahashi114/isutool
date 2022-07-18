package btreemutex

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/btree"
)

func intRange(s int, key int) []int {
	out := make([]int, s)
	for i := 0; i < s; i++ {
		out[i] = i*3 + key
	}
	return out
}

func allInt(t *BTreeMutex[int]) (out []int) {
	t.Ascend(func(a int) bool {
		out = append(out, a)
		return true
	})
	return
}

func Test_BTreeMutexMap(t *testing.T) {
	treeMap := NewMap[int](4, 10, func(a, b int) bool {
		return a < b
	})
	wg := sync.WaitGroup{}
	initCallCount := int64(0)
	offset := 100
	count := 500
	wg.Add(count * 3)
	for i := 0; i < count*3; i++ {
		go func(i int) {
			key := i % 3
			tree, _ := treeMap.GetOrInit(key, func(b *btree.BTreeG[int]) error {
				atomic.AddInt64(&initCallCount, 1)

				for j := 0; j < offset; j++ {
					b.ReplaceOrInsert(j*3 + key)
				}
				return nil
			})

			tree.ReplaceOrInsert((i/3+offset)*3 + key)
			wg.Done()
		}(i)
	}
	wg.Wait()
	if initCallCount != 3 {
		t.Fatalf("initFunc call count want 3, got %v", initCallCount)
	}
	for _, k := range []int{0, 1, 2} {
		tree, err := treeMap.GetOrInit(k, func(b *btree.BTreeG[int]) error { return nil })
		if err != nil {
			t.Fatalf("GetOrInit return err want nil, got %v", err)
		}
		got := allInt(tree)
		want := intRange(count+offset, k)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, want)
		}
	}
}
