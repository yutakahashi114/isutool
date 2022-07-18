package mmap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"
	"os"
	"sort"
	"sync"
	"syscall"
)

func mmap(fd, length int) ([]byte, error) {
	return syscall.Mmap(
		fd,
		0,
		length,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
}

type Mmap[K ByteSeq, V Item[K, V]] struct {
	saveDirectoryPath string
	mod               int
	h1mod             *big.Int
	h2mod             *big.Int

	indexFile []byte
	dataFile  []byte

	maxItemCount int
	pool         *sync.Pool
}

type Item[K ByteSeq, V any] interface {
	Key() K
	Encode() ([]byte, error)
	Decode([]byte) (V, error)
}

type ByteSeq interface {
	~string | ~[]byte
}

const uint64ByteSize = 8

func New[K ByteSeq, V Item[K, V]](saveDirectoryPath string, maxItemCount int) (*Mmap[K, V], error) {
	mod, err := calcMod(maxItemCount)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(saveDirectoryPath, 0777)
	if err != nil {
		return nil, err
	}
	return &Mmap[K, V]{
		saveDirectoryPath: saveDirectoryPath,
		mod:               mod,
		h1mod:             big.NewInt(int64(mod)),
		h2mod:             big.NewInt(int64(mod - 1)),
		maxItemCount:      maxItemCount,
		pool: &sync.Pool{
			New: func() any {
				return new(big.Int)
			},
		},
	}, nil
}

func (m *Mmap[K, V]) Save(itemCount int, limit int, fn func(offset int) ([]V, error)) error {
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
	for i := 0; i < itemCount; i += limit {
		items, err := fn(i)
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

func (m *Mmap[K, V]) Load() error {
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

func (m *Mmap[K, V]) Get(key K) (v V, err error) {
	bigInt := m.pool.Get().(*big.Int)
	defer m.pool.Put(bigInt)
	bigInt.SetBytes([]byte(key))
	h1 := m.h1(bigInt)
	h2 := -1
	for i := 0; i < m.mod; i++ {
		if i > 0 && h2 == -1 {
			h2 = m.h2(bigInt)
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
		keyLen, n := binary.Uvarint(m.dataFile[from:])
		keyBytes := m.dataFile[from+uint64(n) : from+uint64(n)+keyLen]
		if !(string(keyBytes) == string(key)) {
			continue
		}
		v, err = v.Decode(m.dataFile[from+uint64(n)+keyLen : to])
		return v, err
	}
	return v, errors.New("not found")
}

func (m *Mmap[K, V]) store(item V, dataBuf *bytes.Buffer, indexBytes []byte, total int) (int, error) {
	bigInt := m.pool.Get().(*big.Int)
	defer m.pool.Put(bigInt)
	key := item.Key()
	bigInt.SetBytes([]byte(key))
	h1 := m.h1(bigInt)
	h2 := -1
	for i := 0; i < m.mod; i++ {
		if i > 0 && h2 == -1 {
			h2 = m.h2(bigInt)
		}
		h := (h1 + i*h2) % m.mod
		index := h * uint64ByteSize * 2

		fromStart, fromEnd := fromRange(index)
		toStart, toEnd := toRange(index)
		to := binary.BigEndian.Uint64(indexBytes[toStart:toEnd])
		if to == 0 {
			keyLen := len(key)
			keyBytes := make([]byte, binary.MaxVarintLen64+keyLen)
			n := binary.PutUvarint(keyBytes, uint64(keyLen))
			copy(keyBytes[n:], key)
			_, err := dataBuf.Write(keyBytes[:n+keyLen])
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
			usedLen := n + keyLen + len(bs)
			binary.BigEndian.PutUint64(indexBytes[fromStart:fromEnd], uint64(total))
			binary.BigEndian.PutUint64(indexBytes[toStart:toEnd], uint64(total+usedLen))
			return usedLen, nil
		}
		dataBytes := dataBuf.Bytes()
		from := binary.BigEndian.Uint64(indexBytes[fromStart:fromEnd])
		keyLen, n := binary.Uvarint(dataBytes[from:])
		keyBytes := dataBytes[from+uint64(n) : from+uint64(n)+keyLen]
		if string(keyBytes) == string(key) {
			return 0, errors.New("duplicate key")
		}
	}
	return 0, errors.New("cannot store")
}

func fromRange(index int) (int, int) {
	return index, index + uint64ByteSize
}

func toRange(index int) (int, int) {
	return index + uint64ByteSize, index + uint64ByteSize*2
}

func (m *Mmap[K, V]) h1(key *big.Int) int {
	bigInt := m.pool.Get().(*big.Int)
	defer m.pool.Put(bigInt)
	bigInt = bigInt.Mod(key, m.h1mod)
	return int(bigInt.Int64())
}

func (m *Mmap[K, V]) h2(key *big.Int) int {
	bigInt := m.pool.Get().(*big.Int)
	defer m.pool.Put(bigInt)
	bigInt = bigInt.Mod(key, m.h2mod)
	return int(bigInt.Int64() + 1)
}

func calcMod(itemLen int) (int, error) {
	itemLenWithBuffer := itemLen * 5 / 4
	i := sort.Search(len(primeNumbers), func(i int) bool {
		return itemLenWithBuffer <= primeNumbers[i]
	})
	if i < len(primeNumbers) {
		return primeNumbers[i], nil
	}
	i = sort.Search(len(primeNumbers), func(i int) bool {
		return itemLen <= primeNumbers[i]
	})
	if i < len(primeNumbers) {
		return primeNumbers[i], nil
	}
	return 0, errors.New("item count limit over")
}

var primeNumbers = []int{
	163,
	331,
	673,
	1361,
	2729,
	5471,
	10949,
	21911,
	43853,
	87719,
	175447,
	350899,
	701819,
	1403641,
	2807303,
	5614657,
	11229331,
	22458671,
	44917381,
	89834777,
	179669557,
	359339171,
	718678369,
	1437356741,
	// 2874713497,
	// 5749427029,
	// 11498854069,
	// 22997708177,
	// 45995416409,
	// 91990832831,
	// 183981665689,
	// 367963331389,
	// 735926662813,
	// 1471853325643,
	// 2943706651297,
	// 5887413302609,
	// 11774826605231,
	// 23549653210463,
	// 47099306420939,
	// 94198612841897,
	// 188397225683869,
	// 376794451367743,
	// 753588902735509,
	// 1507177805471059,
	// 3014355610942127,
	// 6028711221884317,
	// 12057422443768697,
	// 24114844887537407,
	// 48229689775074839,
}
