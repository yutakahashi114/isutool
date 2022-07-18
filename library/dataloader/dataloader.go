package dataloader

import (
	"errors"
	"time"

	"github.com/yutakahashi114/isutool/library/bulkexecute"
)

type DataLoader[K comparable, V any] struct {
	be *bulkexecute.BulkExecute[K, map[K]V]
}

func New[K comparable, V any](exec func([]K) (map[K]V, error), wait time.Duration) *DataLoader[K, V] {
	return &DataLoader[K, V]{
		be: bulkexecute.New(exec, wait, 64),
	}
}

func (dl *DataLoader[K, V]) Load(key K) (v V, err error) {
	m, err := dl.be.Execute(key)
	if err != nil {
		return v, err
	}
	v, ok := m[key]
	if !ok {
		return v, errors.New("not found")
	}
	return v, nil
}
