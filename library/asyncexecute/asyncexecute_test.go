package asyncexecute

import (
	"fmt"
	"testing"
	"time"
)

type User struct {
	ID   int
	Name string
}

func newUser(i int) User {
	return User{
		ID:   i,
		Name: fmt.Sprintf("name:%v", i),
	}
}

func Test_AsyncExecute(t *testing.T) {
	store := map[int]User{}
	aq := New(
		func(us []User) {
			for _, u := range us {
				store[u.ID] = u
			}
		},
		time.Millisecond*100,
		10,
	)

	for i := 0; i < 100; i++ {
		go func(i int) {
			users := []User{
				newUser(i * 5),
				newUser(i*5 + 1),
				newUser(i*5 + 2),
				newUser(i*5 + 3),
				newUser(i*5 + 4),
			}
			aq.Set(users...)
		}(i)
		time.Sleep(time.Millisecond * 5)
	}
	time.Sleep(time.Second)

	if len(store) != 500 {
		t.Errorf("len(store) want %v, got %v", 500, len(store))
	}
	for i := 0; i < 500; i++ {
		if store[i].ID != i {
			t.Errorf("store[i].ID want %v, got %v", i, store[i].ID)
		}
		if store[i].Name != fmt.Sprintf("name:%v", i) {
			t.Errorf("store[i].Name want %v, got %v", fmt.Sprintf("name:%v", i), store[i].Name)
		}
	}
}
