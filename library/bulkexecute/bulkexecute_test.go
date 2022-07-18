package bulkexecute

import (
	"fmt"
	"math/rand"
	"sync"
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

func Test_BulkExecute(t *testing.T) {
	m := sync.Mutex{}
	store := map[int]User{}
	executing := false
	be := New(
		func(us []User) ([]User, error) {
			m.Lock()
			if executing {
				panic("parallel execute")
			}
			executing = true
			for _, u := range us {
				if _, ok := store[u.ID]; ok {
					panic("duplicate input")
				}
				store[u.ID] = u
			}
			time.Sleep(time.Millisecond * time.Duration(len(us)))
			executing = false
			m.Unlock()
			return us, nil
		},
		time.Millisecond*10,
		10,
	)

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			users := []User{
				newUser(i * 5),
				newUser(i*5 + 1),
				newUser(i*5 + 2),
				newUser(i*5 + 3),
				newUser(i*5 + 4),
			}
			res, _ := be.Execute(users...)
			m.Lock()
			for j := i; j < i+5; j++ {
				if store[j].ID != j {
					t.Errorf("store[j].ID want %v, got %v", j, store[j].ID)
				}
				if store[j].Name != fmt.Sprintf("name:%v", j) {
					t.Errorf("store[j].Name want %v, got %v", fmt.Sprintf("name:%v", j), store[j].Name)
				}
			}
			m.Unlock()
			for _, u := range users {
				exist := false
				for _, resUser := range res {
					if resUser.ID == u.ID {
						exist = true
						break
					}
				}
				if !exist {
					t.Errorf("u.ID %v does not exist in response", u.ID)
				}
			}
			wg.Done()
		}(i)
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
	}
	wg.Wait()
	if len(store) != 5000 {
		t.Errorf("len(store) want %v, got %v", 5000, len(store))
	}
}
