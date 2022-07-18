//go:generate go run ../../encodegen.go -s=TestStruct -f=./other.go
package other

import "time"

type TestStruct struct {
	Int    int
	Str    string
	Pint   *int
	Strs   []string
	StrInt map[string]int
	Time   time.Time
}
