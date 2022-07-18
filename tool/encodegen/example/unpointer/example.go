//go:generate go run ../../encodegen.go -s=FixedLength,String,Pointer,Map,Slice,MapSlice,PointerMap,FixedLengths,Unnamed,StrAlias,IntAlias,StrsAlias,PintsAlias,IntsAlias,TestStructAlias,TestStructs,PtestStructs,StrTestStruct,StrTestStructAlias,Alias,OtherPackage -f=./example.go -p=false
package example

import (
	"time"

	"github.com/yutakahashi114/isutool/tool/encodegen/example/other"
	example "github.com/yutakahashi114/isutool/tool/encodegen/example/other/unpointer"
)

type TestStruct struct {
	Int    int
	Str    string
	Pint   *int
	Strs   []string
	StrInt map[string]int
}

type FixedLength struct {
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Int     int
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Uint    uint
	Float32 float32
	Float64 float64
	Byte    byte
	Rune    rune
	Bool    bool
	Time    time.Time
}

type String struct {
	Str   string
	Bytes []byte
	Runes []rune
}

type Pointer struct {
	Pint    *int
	Pstr    *string
	Pstruct *TestStruct
	Ppint   **int
}

type TestStructKey struct {
	Int int
	Str string
}

type Map struct {
	StrInt        map[string]int
	StrIntStr     map[string]map[int]string
	PointerStrInt *map[string]int
	StructStruct  map[TestStructKey]TestStruct
	StructPstruct map[TestStructKey]*TestStruct
}

type Slice struct {
	Strs        []string
	Intss       [][]int
	Pints       []*int
	PointerStrs *[]string
	Structs     []TestStruct
	Pstructs    []*TestStruct
	Auint8      [5]uint8
	Apstr       [5]*string
}

type MapSlice struct {
	StrInts map[string][]int
	IntStrs []map[int]string
}

type PointerMap struct {
	// original pointer and decoded pointer do not match.
	PstrPint map[*string]*int
}

type FixedLengths struct {
	Map   map[int]uint
	Array [5]uint8
	Slice []int
}

type Unnamed struct {
	int
	*uint
	TestStruct
	*TestStructKey
	time.Time
	Struct struct {
		Int int
		Str string
	}
	Structs []struct {
		Int     int
		Str     string
		Structs []*struct {
			Int int
			Str string
		}
	}
}

type StrAlias string

type IntAlias int

type StrsAlias []string

type PintsAlias []*int

type IntsAlias []IntAlias

type TestStructAlias TestStruct

type TestStructs []TestStruct

type PtestStructs []*TestStruct

type StrTestStruct map[string]TestStruct

type StrTestStructAlias map[StrAlias]*TestStructAlias

type Alias struct {
	StrAlias
	intAlias IntAlias
	TestStructAlias
	TestStructs
	StrTestStruct
}

type OtherPackage struct {
	Int int
	Str string
	*other.TestStruct
	other []other.TestStruct
	example.StrAlias
}
