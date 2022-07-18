package index

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

type TestStruct struct {
	ID        int
	Secondary int
}

func newTestStruct(id int) TestStruct {
	return TestStruct{
		ID:        id,
		Secondary: 9 - id%10,
	}
}

func newTestStructUpdate(id int) TestStruct {
	return TestStruct{
		ID:        id,
		Secondary: id % 10,
	}
}

func testStructRange(s int, reverse bool, secondary bool, updated bool) []TestStruct {
	out := make([]TestStruct, s)
	for i := 0; i < s; i++ {
		v := i
		if reverse {
			v = s - i - 1
		}
		if updated {
			out[i] = newTestStructUpdate(v)
		} else {
			out[i] = newTestStruct(v)
		}
	}
	if secondary {
		sort.Slice(out, func(i, j int) bool {
			if reverse {
				i, j = j, i
			}
			if out[i].Secondary == out[j].Secondary {
				return out[i].ID < out[j].ID
			}
			return out[i].Secondary < out[j].Secondary
		})
	}
	return out
}

func allTestStruct(i *Index[TestStruct], cond *Condition[TestStruct]) (out []TestStruct) {
	i.Ascend(cond, func(a TestStruct) bool {
		out = append(out, a)
		return true
	})
	return
}

func allRevTestStruct(i *Index[TestStruct], cond *Condition[TestStruct]) (out []TestStruct) {
	i.Descend(cond, func(a TestStruct) bool {
		out = append(out, a)
		return true
	})
	return
}

func TestIndex(t *testing.T) {
	primary := Cond(func(a, b TestStruct) bool { return a.ID < b.ID })
	secondary := Cond(func(a, b TestStruct) bool {
		if a.Secondary == b.Secondary {
			return a.ID < b.ID
		}
		return a.Secondary < b.Secondary
	})
	secondaryReverse := Cond(func(a, b TestStruct) bool {
		if a.Secondary == b.Secondary {
			return a.ID > b.ID
		}
		return a.Secondary > b.Secondary
	})
	const treeSize = 10000
	tr := New(
		64,
		primary,
		secondary,
		secondaryReverse,
	)
	for i := 0; i < 10; i++ {
		if tr.Len() != 0 {
			t.Fatalf("len: want %v, got %v", 0, tr.Len())
		}
		if min, ok := tr.Min(primary); ok || min != (TestStruct{}) {
			t.Fatalf("empty min, got %+v", min)
		}
		if min, ok := tr.Min(secondary); ok || min != (TestStruct{}) {
			t.Fatalf("empty min, got %+v", min)
		}
		if max, ok := tr.Max(primary); ok || max != (TestStruct{}) {
			t.Fatalf("empty max, got %+v", max)
		}
		if max, ok := tr.Max(secondary); ok || max != (TestStruct{}) {
			t.Fatalf("empty max, got %+v", max)
		}
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.ReplaceOrInsert(newTestStruct(item)); ok || x != (TestStruct{}) {
				t.Fatal("insert found item", item)
			}
		}
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.Get(newTestStruct(item), primary); !ok || x != newTestStruct(item) {
				t.Fatal("get didn't find item", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondary); !ok || x != newTestStruct(item) {
				t.Fatal("get didn't find item", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondaryReverse); !ok || x != newTestStruct(item) {
				t.Fatal("get didn't find item", item)
			}
		}
		if tr.Len() != treeSize {
			t.Fatalf("len: want %v, got %v", treeSize, tr.Len())
		}
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.ReplaceOrInsert(newTestStruct(item)); !ok || x != newTestStruct(item) {
				t.Fatal("insert didn't find item", item)
			}
		}
		if tr.Len() != treeSize {
			t.Fatalf("len: want %v, got %v", treeSize, tr.Len())
		}

		want := newTestStruct(0)
		if min, ok := tr.Min(primary); !ok || min != want {
			t.Fatalf("min: ok %v want %+v, got %+v", ok, want, min)
		}
		want = newTestStruct((treeSize + 10 - 1) % 10)
		if min, ok := tr.Min(secondary); !ok || min != want {
			t.Fatalf("min: ok %v want %+v, got %+v", ok, want, min)
		}
		if max, ok := tr.Max(secondaryReverse); !ok || max != want {
			t.Fatalf("max: ok %v want %+v, got %+v", ok, want, max)
		}

		want = newTestStruct(treeSize - 1)
		if max, ok := tr.Max(primary); !ok || max != want {
			t.Fatalf("max: ok %v want %+v, got %+v", ok, want, max)
		}
		want = newTestStruct(treeSize - 10)
		if max, ok := tr.Max(secondary); !ok || max != want {
			t.Fatalf("max: ok %v want %+v, got %+v", ok, want, max)
		}
		if min, ok := tr.Min(secondaryReverse); !ok || min != want {
			t.Fatalf("min: ok %v want %+v, got %+v", ok, want, min)
		}

		got := allTestStruct(tr, primary)
		wantRange := testStructRange(treeSize, false, false, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}
		got = allTestStruct(tr, secondary)
		wantRange = testStructRange(treeSize, false, true, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}
		got = allTestStruct(tr, secondaryReverse)
		wantRange = testStructRange(treeSize, true, true, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		gotrev := allRevTestStruct(tr, primary)
		wantrev := testStructRange(treeSize, true, false, false)
		if !reflect.DeepEqual(gotrev, wantrev) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", gotrev, wantrev)
		}
		gotrev = allRevTestStruct(tr, secondary)
		wantrev = testStructRange(treeSize, true, true, false)
		if !reflect.DeepEqual(gotrev, wantrev) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", gotrev, wantrev)
		}
		gotrev = allRevTestStruct(tr, secondaryReverse)
		wantrev = testStructRange(treeSize, false, true, false)
		if !reflect.DeepEqual(gotrev, wantrev) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", gotrev, wantrev)
		}

		for _, item := range rand.Perm(treeSize / 2) {
			item += treeSize / 2
			if x, ok := tr.Delete(newTestStruct(item)); !ok || x != newTestStruct(item) {
				t.Fatalf("didn't find %v", item)
			}
			if x, ok := tr.Delete(newTestStruct(item)); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), primary); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondary); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondaryReverse); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
		}

		got = allTestStruct(tr, primary)
		wantRange = testStructRange(treeSize/2, false, false, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		got = allTestStruct(tr, secondary)
		wantRange = testStructRange(treeSize/2, false, true, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		got = allTestStruct(tr, secondaryReverse)
		wantRange = testStructRange(treeSize/2, true, true, false)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		for _, item := range rand.Perm(treeSize / 2) {
			if x, ok := tr.Delete(newTestStruct(item)); !ok || x != newTestStruct(item) {
				t.Fatalf("didn't find %v", item)
			}
			if x, ok := tr.Delete(newTestStruct(item)); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), primary); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondary); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
			if x, ok := tr.Get(newTestStruct(item), secondaryReverse); ok || x != (TestStruct{}) {
				t.Fatalf("found %v", item)
			}
		}
		if got = allTestStruct(tr, primary); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
		if got = allTestStruct(tr, secondary); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
		if got = allTestStruct(tr, secondaryReverse); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
		if got = allRevTestStruct(tr, primary); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
		if got = allRevTestStruct(tr, secondary); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
		if got = allRevTestStruct(tr, secondaryReverse); len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
	}
}

func Test_Index_ReplaceOrInsert(t *testing.T) {
	primary := Cond(func(a, b TestStruct) bool { return a.ID < b.ID })
	secondary := Cond(func(a, b TestStruct) bool {
		if a.Secondary == b.Secondary {
			return a.ID < b.ID
		}
		return a.Secondary < b.Secondary
	})
	secondaryReverse := Cond(func(a, b TestStruct) bool {
		if a.Secondary == b.Secondary {
			return a.ID > b.ID
		}
		return a.Secondary > b.Secondary
	})
	const treeSize = 10000
	tr := New(
		64,
		primary,
		secondary,
		secondaryReverse,
	)
	for i := 0; i < 10; i++ {
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.ReplaceOrInsert(newTestStruct(item)); ok || x != (TestStruct{}) {
				t.Fatal("insert found item", item)
			}
		}
		for _, item := range rand.Perm(treeSize) {
			updated := newTestStructUpdate(item)
			if x, ok := tr.ReplaceOrInsert(updated); !ok || x != newTestStruct(item) {
				t.Fatal("insert didn't find item", item)
			}
		}
		if tr.Len() != treeSize {
			t.Fatalf("len: want %v, got %v", treeSize, tr.Len())
		}

		got := allTestStruct(tr, primary)
		wantRange := testStructRange(treeSize, false, false, true)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		got = allTestStruct(tr, secondary)
		wantRange = testStructRange(treeSize, false, true, true)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		got = allTestStruct(tr, secondaryReverse)
		wantRange = testStructRange(treeSize, true, true, true)
		if !reflect.DeepEqual(got, wantRange) {
			t.Fatalf("mismatch:\n got: %v\nwant: %v", got, wantRange)
		}

		for _, item := range rand.Perm(treeSize) {
			updated := newTestStructUpdate(item)
			if x, ok := tr.Delete(newTestStruct(item)); !ok || x != updated {
				t.Fatalf("didn't find %v", item)
			}
		}
	}
}

func Test_Index_AscendRange_DescendRange(t *testing.T) {
	primary := Cond(func(a, b TestStruct) bool { return a.ID < b.ID })
	secondary := Cond(func(a, b TestStruct) bool {
		if a.Secondary == b.Secondary {
			return a.ID < b.ID
		}
		return a.Secondary < b.Secondary
	})
	const treeSize = 100
	tr := New(
		4,
		primary,
		secondary,
	)

	from := 10
	to := 90
	for i := 0; i < 10; i++ {
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.ReplaceOrInsert(newTestStruct(item)); ok || x != (TestStruct{}) {
				t.Fatal("insert found item", item)
			}
		}
		if tr.Len() != treeSize {
			t.Fatalf("len: want %v, got %v", treeSize, tr.Len())
		}
		// AscendRange
		{
			count := 0
			all := testStructRange(treeSize, false, false, false)
			tr.AscendRange(primary, all[from], all[to], func(ts TestStruct) bool {
				if ts != all[from+count] {
					t.Fatalf("all[from+count] want %v, got %v", all[from+count], ts)
				}
				count++
				return true
			})
			if count != (to - from) {
				t.Fatalf("AscendRange last want %v, got %v", to-from, count)
			}
			count = 0
			all = testStructRange(treeSize, false, true, false)
			tr.AscendRange(secondary, all[from], all[to], func(ts TestStruct) bool {
				if ts != all[from+count] {
					t.Fatalf("all[from+count] want %v, got %v", all[from+count], ts)
				}
				count++
				return true
			})
			if count != (to - from) {
				t.Fatalf("AscendRange last want %v, got %v", to-from, count)
			}
		}
		// DescendRange
		{
			count := 0
			all := testStructRange(treeSize, false, false, false)
			tr.DescendRange(primary, all[to], all[from], func(ts TestStruct) bool {
				if ts != all[to-count] {
					t.Fatalf("all[to-count] want %v, got %v", all[to-count], ts)
				}
				count++
				return true
			})
			if count != (to - from) {
				t.Fatalf("DescendRange last want %v, got %v", to-from, count)
			}
			count = 0
			all = testStructRange(treeSize, false, true, false)
			tr.DescendRange(secondary, all[to], all[from], func(ts TestStruct) bool {
				if ts != all[to-count] {
					t.Fatalf("all[to-count] want %v, got %v", all[to-count], ts)
				}
				count++
				return true
			})
			if count != (to - from) {
				t.Fatalf("DescendRange last want %v, got %v", to-from, count)
			}
		}
		for _, item := range rand.Perm(treeSize) {
			if x, ok := tr.Delete(newTestStruct(item)); !ok || x != newTestStruct(item) {
				t.Fatalf("didn't find %v", item)
			}
		}
	}
}

type User struct {
	ID   int
	Age  int
	Name string
}

func newUser(i int) User {
	return User{
		ID:   i,
		Age:  i % 100,
		Name: fmt.Sprintf("name:%v", i),
	}
}

var count = 100000
var degree = 256

func createUsers() []User {
	users := make([]User, 0, count)
	for _, v := range rand.Perm(count) {
		users = append(users, newUser(v))
	}
	return users
}

func (u User) OrderByID(target User) bool {
	return u.ID < target.ID
}

func (u User) OrderByAgeAndID(target User) bool {
	if u.Age == target.Age {
		return u.ID < target.ID
	}
	return u.Age < target.Age
}

var (
	userPrimary   = Cond(User.OrderByID)
	userSecondary = Cond(User.OrderByAgeAndID)
)

func createUsersIndex() (*Index[User], []User) {
	index := New(degree, userPrimary, userSecondary)
	users := createUsers()
	for _, u := range users {
		index.ReplaceOrInsert(u)
	}
	return index, users
}
func Benchmark_Index_ReplaceOrInsert_insert(b *testing.B) {
	users := createUsers()
	b.ResetTimer()
	i := 0
	for i < b.N {
		b.StopTimer()
		index := New(degree, userPrimary, userSecondary)
		b.StartTimer()
		for _, u := range users {
			index.ReplaceOrInsert(u)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func Benchmark_Index_ReplaceOrInsert_replace(b *testing.B) {
	index, users := createUsersIndex()
	b.ResetTimer()
	i := 0
	for i < b.N {
		for _, u := range users {
			u.Age = u.Age * 2
			index.ReplaceOrInsert(u)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func Benchmark_Index_Delete(b *testing.B) {
	users := createUsers()
	b.ResetTimer()
	i := 0
	for i < b.N {
		b.StopTimer()
		index := New(degree, userPrimary, userSecondary)
		for _, u := range users {
			index.ReplaceOrInsert(u)
		}
		b.StartTimer()
		for _, u := range users {
			index.Delete(u)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func Benchmark_Index_Get(b *testing.B) {
	index, users := createUsersIndex()
	b.ResetTimer()
	i := 0
	for i < b.N {
		for _, u := range users {
			_, _ = index.Get(u, userPrimary)
			i++
			if i >= b.N {
				return
			}
		}
	}
}
