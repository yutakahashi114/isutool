package cache

import (
	"fmt"
	"sync"
	"testing"
)

type User struct {
	ID   int
	Name string
}

func newUser(key int, i int) User {
	return User{
		ID:   key,
		Name: fmt.Sprintf("name:%v", i),
	}
}

func Test_Cache_go(t *testing.T) {
	c := New[int, User](10)
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			key := i % 4
			switch i % 3 {
			case 0:
				c.Set(key, newUser(key, i))
			case 1:
				fmt.Println(c.Get(key))
			case 2:
				c.Update(key, func(current User) User {
					current.Name += "update"
					return current
				})
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	t.Logf("no error")
}

func Test_Cache(t *testing.T) {
	c := New[int, User](10)
	check := func(t *testing.T, key int, i int, u User) {
		if u.ID != key {
			t.Fatalf("key %v u.ID want %v, got %v", i, key, u.ID)
		}
		if u.Name != fmt.Sprintf("name:%v", i) {
			t.Fatalf("key %v u.Name want %v, got %v", i, fmt.Sprintf("name:%v", i), u.Name)
		}
	}
	for i := 0; i < 10; i++ {
		key := i
		c.Set(key, newUser(key, i))
	}
	for i := 0; i < 10; i++ {
		key := i
		v, ok := c.Get(key)
		if !ok {
			t.Fatalf("key %v ok want %v, got %v", key, true, ok)
		}
		check(t, key, i, v)
	}
	v, ok := c.Get(10)
	if ok {
		t.Errorf("key %v ok want %v, got %v", "10", false, ok)
	}

	for i := 0; i < 5; i++ {
		key := i
		c.Set(key, newUser(key, i+10))
	}

	for i := 0; i < 10; i++ {
		key := i
		if i < 5 {
			v, ok := c.Get(key)
			if !ok {
				t.Fatalf("key %v ok want %v, got %v", key, true, ok)
			}
			check(t, key, i+10, v)
		} else {
			v, ok := c.Get(key)
			if !ok {
				t.Fatalf("key %v ok want %v, got %v", key, true, ok)
			}
			check(t, key, i, v)
		}
	}

	key := 10

	// set
	v, err := c.GetOrSet(key, func() (User, error) {
		return newUser(key, 10), nil
	})
	if err != nil {
		t.Errorf("key %v err want %v, got %v", "10", nil, err)
	}
	check(t, key, 10, v)

	v, ok = c.Get(key)
	if !ok {
		t.Fatalf("key %v ok want %v, got %v", key, true, ok)
	}
	check(t, key, 10, v)

	// get
	v, err = c.GetOrSet(key, func() (User, error) {
		// never set
		return newUser(key, 11), nil
	})
	if err != nil {
		t.Errorf("key %v err want %v, got %v", "10", nil, err)
	}
	check(t, key, 10, v)

	v, ok = c.Get(key)
	if !ok {
		t.Fatalf("key %v ok want %v, got %v", key, true, ok)
	}
	check(t, key, 10, v)

	// error
	key = 11
	v, err = c.GetOrSet(key, func() (User, error) {
		return User{}, fmt.Errorf("something")
	})
	if err == nil {
		t.Errorf("key %v err want %v, got %v", "11", fmt.Errorf("something"), err)
	}
}
