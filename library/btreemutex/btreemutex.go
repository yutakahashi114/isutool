package btreemutex

import (
	"sync"

	"github.com/google/btree"
)

type BTreeMutex[T any] struct {
	m     sync.RWMutex
	bTree *btree.BTreeG[T]
}

func New[T any](degree int, less btree.LessFunc[T]) *BTreeMutex[T] {
	return &BTreeMutex[T]{
		bTree: btree.NewG(degree, less),
	}
}

func (t *BTreeMutex[T]) Get(key T) (res T, found bool) {
	t.m.RLock()
	res, found = t.bTree.Get(key)
	t.m.RUnlock()
	return res, found
}

func (t *BTreeMutex[T]) ReplaceOrInsert(item T) (res T, found bool) {
	t.m.Lock()
	res, found = t.bTree.ReplaceOrInsert(item)
	t.m.Unlock()
	return res, found
}

func (t *BTreeMutex[T]) BulkReplaceOrInsert(items []T) {
	t.m.Lock()
	for _, item := range items {
		t.bTree.ReplaceOrInsert(item)
	}
	t.m.Unlock()
}

func (t *BTreeMutex[T]) Delete(item T) (T, bool) {
	t.m.Lock()
	res, found := t.bTree.Delete(item)
	t.m.Unlock()
	return res, found
}

func (t *BTreeMutex[T]) Ascend(iterator btree.ItemIteratorG[T]) {
	t.m.RLock()
	t.bTree.Ascend(iterator)
	t.m.RUnlock()
}

func (t *BTreeMutex[T]) AscendRange(greaterOrEqual, lessThan T, iterator btree.ItemIteratorG[T]) {
	t.m.RLock()
	t.bTree.AscendRange(greaterOrEqual, lessThan, iterator)
	t.m.RUnlock()
}

func (t *BTreeMutex[T]) Descend(iterator btree.ItemIteratorG[T]) {
	t.m.RLock()
	t.bTree.Descend(iterator)
	t.m.RUnlock()
}

func (t *BTreeMutex[T]) DescendRange(lessOrEqual, greaterThan T, iterator btree.ItemIteratorG[T]) {
	t.m.RLock()
	t.bTree.DescendRange(lessOrEqual, greaterThan, iterator)
	t.m.RUnlock()
}

type BTreeMutexMap[K comparable, T any] struct {
	m       sync.RWMutex
	treeMap map[K]*BTreeMutex[T]
	less    btree.LessFunc[T]
	degree  int
}

func NewMap[K comparable, T any](degree int, cap int, less btree.LessFunc[T]) *BTreeMutexMap[K, T] {
	return &BTreeMutexMap[K, T]{
		treeMap: make(map[K]*BTreeMutex[T], cap),
		less:    less,
		degree:  degree,
	}
}

func (b *BTreeMutexMap[K, T]) GetOrInit(key K, initFunc func(*btree.BTreeG[T]) error) (*BTreeMutex[T], error) {
	tree := b.getOrSet(key)
	tree.m.Lock()
	if tree.bTree == nil {
		tree.bTree = btree.NewG(b.degree, b.less)
		err := initFunc(tree.bTree)
		if err != nil {
			tree.m.Unlock()
			return nil, err
		}
	}
	tree.m.Unlock()
	return tree, nil
}

func (b *BTreeMutexMap[K, T]) getOrSet(key K) *BTreeMutex[T] {
	b.m.RLock()
	tree, ok := b.treeMap[key]
	b.m.RUnlock()
	if ok {
		return tree
	}
	b.m.Lock()
	tree, ok = b.treeMap[key]
	if ok {
		b.m.Unlock()
		return tree
	}

	tree = &BTreeMutex[T]{}

	b.treeMap[key] = tree
	b.m.Unlock()

	return tree
}
