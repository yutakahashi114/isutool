package mmap

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

type User struct {
	ID    int
	Name  string
	Email string
	Age   int
}

func newUser(i int) User {
	return User{
		ID:    i,
		Name:  fmt.Sprintf("name:%v", i),
		Email: fmt.Sprintf("email:%v", i),
		Age:   i % 100,
	}
}

func (u User) Key() string {
	return u.Name
}

func (u User) Encode() ([]byte, error) {
	size := 0
	// ID
	size += binary.MaxVarintLen64
	// Name
	size += binary.MaxVarintLen64
	size += len(u.Name)
	// Email
	size += binary.MaxVarintLen64
	size += len(u.Email)
	// Age
	size += binary.MaxVarintLen64

	out := make([]byte, size)

	n := 0
	// ID
	n += binary.PutVarint(out[n:], int64(u.ID))
	// Name
	n += binary.PutUvarint(out[n:], uint64(len(u.Name)))
	n += copy(out[n:], u.Name)
	// Email
	n += binary.PutUvarint(out[n:], uint64(len(u.Email)))
	n += copy(out[n:], u.Email)
	// Age
	n += binary.PutVarint(out[n:], int64(u.Age))

	return out[:n], nil
}

func (u User) Decode(in []byte) (User, error) {
	n := 0
	// ID
	idRaw, idLen := binary.Varint(in[n:])
	u.ID = int(idRaw)
	n += idLen
	// Name
	nameLen, nameLenLen := binary.Uvarint(in[n:])
	n += nameLenLen
	u.Name = string(in[n : n+int(nameLen)])
	n += int(nameLen)
	// Email
	emailLen, emailLenLen := binary.Uvarint(in[n:])
	n += emailLenLen
	u.Email = string(in[n : n+int(emailLen)])
	n += int(emailLen)
	// Age
	ageRaw, ageLen := binary.Varint(in[n:])
	u.Age = int(ageRaw)
	n += ageLen

	return u, nil
}

func byteSliceToString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

func Test_Mmap(t *testing.T) {
	count := 100
	users := make([]User, count)
	m, err := New[string, User]("test", count)
	if err != nil {
		t.Fatalf("New err want %v, got %v", nil, err)
	}
	for _, i := range rand.Perm(count) {
		u := newUser(i)
		users[i] = u
	}
	err = m.Save(count, count/10, func(offset int) ([]User, error) {
		return users[offset : offset+count/10], nil
	})
	if err != nil {
		t.Fatalf("Store err want %v, got %v", nil, err)
	}
	err = m.Load()
	if err != nil {
		t.Fatalf("Load err want %v, got %v", nil, err)
	}
	for _, i := range rand.Perm(count) {
		u, err := m.Get(fmt.Sprintf("name:%v", i))
		if err != nil {
			t.Fatalf("Get err want %v, got %v", nil, err)
		}
		want := newUser(i)
		if !reflect.DeepEqual(u, want) {
			t.Fatalf("Get() = %v, want %v", u, want)
		}
	}
}

const count = 1_000_000

func Benchmark_Mmap(b *testing.B) {
	m, err := New[string, User]("test", count)
	if err != nil {
		b.Fatal(err)
	}
	users := make([]User, count)
	for i := 1; i <= count; i++ {
		u := newUser(i)
		users[i-1] = u
	}
	err = m.Save(count, count, func(offset int) ([]User, error) {
		return users, nil
	})
	if err != nil {
		b.Fatal(err)
	}
	err = m.Load()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	i := 0
	for i < b.N {
		for j := 1; j <= count; j++ {
			u, err := m.Get("name:" + strconv.Itoa(j))
			_, _ = u, err
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func Benchmark_map(b *testing.B) {
	users := make([]User, count)
	for i := 1; i <= count; i++ {
		users[i-1] = newUser(i)
	}
	m := make(map[int]User, count)
	for _, u := range users {
		m[u.ID] = u
	}
	b.ResetTimer()
	i := 0
	for i < b.N {
		for j := 1; j <= count; j++ {
			u, ok := m[j]
			_, _ = u, ok
			i++
			if i >= b.N {
				return
			}
		}
	}
}
