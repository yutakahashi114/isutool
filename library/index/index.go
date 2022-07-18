package index

import (
	"fmt"
	"sync"

	"github.com/google/btree"
)

type Index[T any] struct {
	m       sync.RWMutex
	trees   []*treeAndCond[T]
	treeMap map[*Condition[T]]*btree.BTreeG[T]
}

type treeAndCond[T any] struct {
	tree *btree.BTreeG[T]
	cond *Condition[T]
}

func New[T any](degree int, primary *Condition[T], secondaries ...*Condition[T]) *Index[T] {
	trees := make([]*treeAndCond[T], 0, len(secondaries)+1)
	treeMap := make(map[*Condition[T]]*btree.BTreeG[T], len(secondaries)+1)

	tree := &treeAndCond[T]{
		tree: btree.NewG(degree, btree.LessFunc[T](primary.lessFunc)),
		cond: primary,
	}
	trees = append(trees, tree)
	treeMap[primary] = tree.tree
	for _, secondary := range secondaries {
		tree := &treeAndCond[T]{
			tree: btree.NewG(degree, btree.LessFunc[T](secondary.lessFunc)),
			cond: secondary,
		}
		trees = append(trees, tree)
		treeMap[secondary] = tree.tree
	}
	return &Index[T]{
		trees:   trees,
		treeMap: treeMap,
	}
}

type Condition[T any] struct {
	lessFunc btree.LessFunc[T]
}

func Cond[T any](lessFunc btree.LessFunc[T]) *Condition[T] {
	return &Condition[T]{lessFunc: lessFunc}
}

func (i *Index[T]) Get(key T, cond *Condition[T]) (res T, found bool) {
	i.m.RLock()
	resp, found := i.treeMap[cond].Get(key)
	if found {
		res = resp
	}
	i.m.RUnlock()
	return
}

func (i *Index[T]) ReplaceOrInsert(value T) (res T, found bool) {
	i.m.Lock()
	resp, found := i.trees[0].tree.ReplaceOrInsert(value)
	if found {
		res = resp
		for _, t := range i.trees[1:] {
			t.tree.Delete(resp)
			t.tree.ReplaceOrInsert(value)
		}
	} else {
		for _, t := range i.trees[1:] {
			t.tree.ReplaceOrInsert(value)
		}
	}
	i.m.Unlock()
	return
}

func (i *Index[T]) MustInsert(value T) {
	i.m.Lock()
	for i, t := range i.trees {
		res, found := t.tree.ReplaceOrInsert(value)
		if found {
			panic(fmt.Sprintf("index %d already exist, insert: %v, exist: %v", i, value, res))
		}
	}
	i.m.Unlock()
}

func (i *Index[T]) Delete(value T) (res T, found bool) {
	i.m.Lock()
	remove, found := i.trees[0].tree.Delete(value)
	if found {
		res = remove
		for _, t := range i.trees[1:] {
			t.tree.Delete(remove)
		}
	}
	i.m.Unlock()
	return
}

func (i *Index[T]) Ascend(cond *Condition[T], iterator btree.ItemIteratorG[T]) {
	i.m.RLock()
	i.treeMap[cond].Ascend(iterator)
	i.m.RUnlock()
}

func (i *Index[T]) AscendRange(cond *Condition[T], greaterOrEqual, lessThan T, iterator btree.ItemIteratorG[T]) {
	i.m.RLock()
	i.treeMap[cond].AscendRange(greaterOrEqual, lessThan, iterator)
	i.m.RUnlock()
}

func (i *Index[T]) Descend(cond *Condition[T], iterator btree.ItemIteratorG[T]) {
	i.m.RLock()
	i.treeMap[cond].Descend(iterator)
	i.m.RUnlock()
}

func (i *Index[T]) DescendRange(cond *Condition[T], lessOrEqual, greaterThan T, iterator btree.ItemIteratorG[T]) {
	i.m.RLock()
	i.treeMap[cond].DescendRange(lessOrEqual, greaterThan, iterator)
	i.m.RUnlock()
}

func (i *Index[T]) Min(cond *Condition[T]) (res T, found bool) {
	i.m.RLock()
	resp, found := i.treeMap[cond].Min()
	if found {
		res = resp
	}
	i.m.RUnlock()
	return
}

func (i *Index[T]) Max(cond *Condition[T]) (res T, found bool) {
	i.m.RLock()
	resp, found := i.treeMap[cond].Max()
	if found {
		res = resp
	}
	i.m.RUnlock()
	return
}

func (i *Index[T]) Len() int {
	i.m.RLock()
	res := i.trees[0].tree.Len()
	i.m.RUnlock()
	return res
}

func StringExactMatch(target string) string {
	return string(append([]byte(target), 0))
}

func StringPrefixMatch(target string) string {
	bytes := []byte(target)
	for i := len(bytes) - 1; i >= 0; i-- {
		if bytes[i] != 255 {
			bytes[i] = bytes[i] + 1
			return string(bytes)
		}
		bytes[i] = 0
	}
	// 全byteが255のとき、以降全てが前方一致するのでlessThanで定義できない
	// ほぼ起きないので一旦無視
	panic(target)
}
