package mmap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

type MmapInt[V ItemInt[V]] struct {
	saveDirectoryPath string
	mod               int
	h1mod             int
	h2mod             int

	indexFile []byte
	dataFile  []byte

	maxItemCount int
}

type ItemInt[V any] interface {
	Key() int
	Encode() ([]byte, error)
	Decode([]byte) (V, error)
}

func NewInt[V ItemInt[V]](saveDirectoryPath string, maxItemCount int) (*MmapInt[V], error) {
	mod, err := calcMod(maxItemCount)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(saveDirectoryPath, 0777)
	if err != nil {
		return nil, err
	}
	return &MmapInt[V]{
		saveDirectoryPath: saveDirectoryPath,
		mod:               mod,
		h1mod:             mod,
		h2mod:             mod - 1,
		maxItemCount:      maxItemCount,
	}, nil
}

func (m *MmapInt[V]) Save(itemCount int, limit int, fn func(offset int) ([]V, error)) error {
	if itemCount > m.maxItemCount {
		return errors.New("item count limit over")
	}
	indexFile, err := os.Create(m.saveDirectoryPath + "/index")
	if err != nil {
		return err
	}
	defer indexFile.Close()
	dataFile, err := os.Create(m.saveDirectoryPath + "/data")
	if err != nil {
		return err
	}
	defer dataFile.Close()
	buf := &bytes.Buffer{}
	indexBytes := make([]byte, m.mod*uint64ByteSize*2)
	buf.Grow(500 * itemCount)
	var total int
	for offset := 0; offset < itemCount; offset += limit {
		items, err := fn(offset)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			usedLen, err := m.store(item, buf, indexBytes, total)
			if err != nil {
				return err
			}
			total += usedLen
		}
	}

	_, err = buf.WriteTo(dataFile)
	if err != nil {
		return err
	}
	_, err = indexFile.Write(indexBytes)
	if err != nil {
		return err
	}
	return nil
}

func (m *MmapInt[V]) Load() error {
	indexFile, err := os.Open(m.saveDirectoryPath + "/index")
	if err != nil {
		return err
	}
	defer indexFile.Close()
	indexInfo, err := indexFile.Stat()
	if err != nil {
		return err
	}
	m.indexFile, err = mmap(int(indexFile.Fd()), int(indexInfo.Size()))
	if err != nil {
		return err
	}

	dataFile, err := os.Open(m.saveDirectoryPath + "/data")
	if err != nil {
		return err
	}
	defer dataFile.Close()
	dataInfo, err := dataFile.Stat()
	if err != nil {
		return err
	}
	m.dataFile, err = mmap(int(dataFile.Fd()), int(dataInfo.Size()))
	if err != nil {
		return err
	}
	return nil
}

func (m *MmapInt[V]) Get(key int) (v V, err error) {
	h1 := m.h1(key)
	h2 := -1
	for i := 0; i < m.mod; i++ {
		if i > 0 && h2 == -1 {
			h2 = m.h2(key)
		}
		h := (h1 + i*h2) % m.mod
		index := h * uint64ByteSize * 2

		toStart, toEnd := toRange(index)
		to := binary.BigEndian.Uint64(m.indexFile[toStart:toEnd])
		if to == 0 {
			return v, errors.New("not found")
		}
		fromStart, fromEnd := fromRange(index)
		from := binary.BigEndian.Uint64(m.indexFile[fromStart:fromEnd])
		existKey, n := binary.Varint(m.dataFile[from:])
		if existKey != int64(key) {
			continue
		}
		v, err = v.Decode(m.dataFile[from+uint64(n) : to])
		return v, err
	}
	return v, errors.New("not found")
}

func (m *MmapInt[V]) store(item V, dataBuf *bytes.Buffer, indexBytes []byte, total int) (int, error) {
	key := item.Key()
	h1 := m.h1(key)
	h2 := -1
	for i := 0; i < m.mod; i++ {
		if i > 0 && h2 == -1 {
			h2 = m.h2(key)
		}
		h := (h1 + i*h2) % m.mod
		index := h * uint64ByteSize * 2

		fromStart, fromEnd := fromRange(index)
		toStart, toEnd := toRange(index)
		to := binary.BigEndian.Uint64(indexBytes[toStart:toEnd])
		if to == 0 {
			keyBytes := make([]byte, binary.MaxVarintLen64)
			n := binary.PutVarint(keyBytes, int64(key))
			_, err := dataBuf.Write(keyBytes[:n])
			if err != nil {
				return 0, err
			}
			bs, err := item.Encode()
			if err != nil {
				return 0, err
			}
			_, err = dataBuf.Write(bs)
			if err != nil {
				return 0, err
			}
			usedLen := n + len(bs)
			binary.BigEndian.PutUint64(indexBytes[fromStart:fromEnd], uint64(total))
			binary.BigEndian.PutUint64(indexBytes[toStart:toEnd], uint64(total+usedLen))
			return usedLen, nil
		}
		dataBytes := dataBuf.Bytes()
		from := binary.BigEndian.Uint64(indexBytes[fromStart:fromEnd])
		existKey, _ := binary.Varint(dataBytes[from:])
		if existKey == int64(key) {
			return 0, errors.New("duplicate key")
		}
	}
	return 0, errors.New("cannot store")
}

func (m *MmapInt[V]) h1(key int) int {
	return key % m.h1mod
}

func (m *MmapInt[V]) h2(key int) int {
	return (key % m.h2mod) + 1
}
