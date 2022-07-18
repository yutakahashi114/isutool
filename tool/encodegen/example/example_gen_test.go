package example

import (
	"math"
	"reflect"
	"testing"
	"time"

	other "github.com/yutakahashi114/isutool/tool/encodegen/example/other"
	example "github.com/yutakahashi114/isutool/tool/encodegen/example/other/unpointer"
)

func TestFixedLength(t *testing.T) {
	type fields struct {
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
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success_max",
			fields: fields{
				Int8:    math.MaxInt8,
				Int16:   math.MaxInt16,
				Int32:   math.MaxInt32,
				Int64:   math.MaxInt64,
				Int:     math.MaxInt,
				Uint8:   math.MaxUint8,
				Uint16:  math.MaxUint16,
				Uint32:  math.MaxUint32,
				Uint64:  math.MaxUint64,
				Uint:    math.MaxUint,
				Float32: math.MaxFloat32,
				Float64: math.MaxFloat64,
				Byte:    math.MaxUint8,
				Rune:    math.MaxInt32,
				Bool:    true,
				Time:    now,
			},
		},
		{
			name: "success_min",
			fields: fields{
				Int8:    math.MinInt8,
				Int16:   math.MinInt16,
				Int32:   math.MinInt32,
				Int64:   math.MinInt64,
				Int:     math.MinInt,
				Uint8:   0,
				Uint16:  0,
				Uint32:  0,
				Uint64:  0,
				Uint:    0,
				Float32: math.SmallestNonzeroFloat32,
				Float64: math.SmallestNonzeroFloat64,
				Byte:    0,
				Rune:    math.MinInt32,
				Bool:    false,
				Time:    time.Time{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FixedLength{
				Int8:    tt.fields.Int8,
				Int16:   tt.fields.Int16,
				Int32:   tt.fields.Int32,
				Int64:   tt.fields.Int64,
				Int:     tt.fields.Int,
				Uint8:   tt.fields.Uint8,
				Uint16:  tt.fields.Uint16,
				Uint32:  tt.fields.Uint32,
				Uint64:  tt.fields.Uint64,
				Uint:    tt.fields.Uint,
				Float32: tt.fields.Float32,
				Float64: tt.fields.Float64,
				Byte:    tt.fields.Byte,
				Rune:    tt.fields.Rune,
				Bool:    tt.fields.Bool,
				Time:    tt.fields.Time,
			}
			encoded, err := f.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixedLength.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &FixedLength{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("FixedLength.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(f, decoded) {
				t.Errorf("FixedLength.Decode() = %v, want %v", decoded, f)
			}
		})
	}
}

func FuzzString(f *testing.F) {
	f.Add("test123")
	f.Fuzz(func(t *testing.T, str string) {
		s := &String{
			Str:   str,
			Bytes: []byte(str),
			Runes: []rune(str),
		}
		encoded, err := s.Encode()
		if (err != nil) != false {
			t.Errorf("String.Encode() error = %v, wantErr %v", err, false)
			return
		}
		decoded := &String{}
		decoded, err = decoded.Decode(encoded)
		if (err != nil) != false {
			t.Errorf("String.Decode() error = %v, wantErr %v", err, false)
			return
		}
		if !reflect.DeepEqual(s, decoded) {
			t.Errorf("String.Decode() = %v, want %v", decoded, s)
		}
	})
}

func TestPointer(t *testing.T) {
	type fields struct {
		Pint    *int
		Pstr    *string
		Pstruct *TestStruct
		Ppint   **int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Pint: toPointer(math.MaxInt),
				Pstr: toPointer("test123"),
				Pstruct: toPointer(TestStruct{
					Int:  -1,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				}),
				Ppint: toPointer(toPointer(math.MinInt)),
			},
		},
		{
			name: "success_nil",
			fields: fields{
				Pint:    nil,
				Pstr:    nil,
				Pstruct: nil,
				Ppint:   toPointer[*int](nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pointer{
				Pint:    tt.fields.Pint,
				Pstr:    tt.fields.Pstr,
				Pstruct: tt.fields.Pstruct,
				Ppint:   tt.fields.Ppint,
			}
			encoded, err := p.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Pointer.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &Pointer{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != false {
				t.Errorf("Pointer.Decode() error = %v, wantErr %v", err, false)
				return
			}
			if !reflect.DeepEqual(p, decoded) {
				t.Errorf("Pointer.Decode() = %v, want %v", decoded, p)
			}
		})
	}
}

func TestMap(t *testing.T) {
	type fields struct {
		StrInt        map[string]int
		StrIntStr     map[string]map[int]string
		PointerStrInt *map[string]int
		StructStruct  map[TestStructKey]TestStruct
		StructPstruct map[TestStructKey]*TestStruct
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				StrInt: map[string]int{
					"math.MaxInt": math.MaxInt,
					"math.MinInt": math.MinInt,
				},
				StrIntStr: map[string]map[int]string{
					"math.MaxInt": {
						math.MinInt: "test1",
						math.MaxInt: "test2",
					},
					"math.MinInt": {
						math.MaxInt: "test3",
						math.MinInt: "test4",
					},
				},
				PointerStrInt: &map[string]int{
					"math.MaxInt": math.MaxInt,
					"math.MinInt": math.MinInt,
				},
				StructStruct: map[TestStructKey]TestStruct{
					{}: {
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
					{Int: math.MaxInt, Str: "test123"}: {
						Int:    math.MinInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
					},
				},
				StructPstruct: map[TestStructKey]*TestStruct{
					{Int: math.MaxInt, Str: "test123"}: {
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
					{}: {
						Int:    math.MinInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
					},
				},
			},
		},
		{
			name: "success_empty_nil",
			fields: fields{
				StrInt:    map[string]int{},
				StrIntStr: nil,
				PointerStrInt: func() *map[string]int {
					m := map[string]int(nil)
					return &m
				}(),
				StructStruct:  nil,
				StructPstruct: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Map{
				StrInt:        tt.fields.StrInt,
				StrIntStr:     tt.fields.StrIntStr,
				PointerStrInt: tt.fields.PointerStrInt,
				StructStruct:  tt.fields.StructStruct,
				StructPstruct: tt.fields.StructPstruct,
			}
			encoded, err := m.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &Map{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(m, decoded) {
				t.Errorf("Map.Decode() = %v, want %v", decoded, m)
			}
		})
	}
}

func TestSlice(t *testing.T) {
	type fields struct {
		Strs        []string
		Intss       [][]int
		Pints       []*int
		PointerStrs *[]string
		Structs     []TestStruct
		Pstructs    []*TestStruct
		Auint8      [5]uint8
		Apstr       [5]*string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Strs: []string{
					"test123",
					"あいうえお",
				},
				Intss: [][]int{
					{math.MinInt, 0, math.MaxInt},
					nil,
					{math.MaxInt, 0, math.MinInt},
				},
				Pints: []*int{
					toPointer(math.MinInt),
					toPointer(0),
					toPointer(math.MaxInt),
				},
				PointerStrs: &[]string{
					"あいうえお",
					"test123",
				},
				Structs: []TestStruct{
					{
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
					{
						Int:    math.MaxInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
					},
					{},
				},
				Pstructs: []*TestStruct{
					{
						Int:    math.MaxInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
					},
					{},
					{
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
					nil,
				},
				Auint8: [5]uint8{0, 100, 150, 200, math.MaxUint8},
				Apstr:  [5]*string{toPointer("test123"), toPointer("あいうえお"), nil, toPointer(""), toPointer("!\"#$%&'()")},
			},
		},
		{
			name: "success_empty_nil",
			fields: fields{
				Strs:  []string{},
				Intss: [][]int{},
				Pints: nil,
				PointerStrs: func() *[]string {
					m := []string(nil)
					return &m
				}(),
				Structs:  nil,
				Pstructs: nil,
				Auint8:   [5]uint8{},
				Apstr:    [5]*string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slice{
				Strs:        tt.fields.Strs,
				Intss:       tt.fields.Intss,
				Pints:       tt.fields.Pints,
				PointerStrs: tt.fields.PointerStrs,
				Structs:     tt.fields.Structs,
				Pstructs:    tt.fields.Pstructs,
				Auint8:      tt.fields.Auint8,
				Apstr:       tt.fields.Apstr,
			}
			encoded, err := s.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Slice.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &Slice{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slice.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(s, decoded) {
				t.Errorf("Slice.Decode() = %v, want %v", decoded, s)
			}
		})
	}
}

func TestMapSlice(t *testing.T) {
	type fields struct {
		StrInts map[string][]int
		IntStrs []map[int]string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				StrInts: map[string][]int{
					"":        nil,
					"test":    {},
					"test123": {math.MinInt, math.MaxInt, 0},
				},
				IntStrs: []map[int]string{
					{
						0:     "あいうえお",
						12345: "12345",
					},
					{},
					nil,
					{
						math.MinInt: "",
						math.MaxInt: "test123",
					},
				},
			},
		},
		{
			name: "success_empty",
			fields: fields{
				StrInts: map[string][]int{},
				IntStrs: []map[int]string{},
			},
		},
		{
			name: "success_nil",
			fields: fields{
				StrInts: nil,
				IntStrs: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MapSlice{
				StrInts: tt.fields.StrInts,
				IntStrs: tt.fields.IntStrs,
			}
			encoded, err := m.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSlice.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &MapSlice{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSlice.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(m, decoded) {
				t.Errorf("MapSlice.Decode() = %v, want %v", decoded, m)
			}
		})
	}
}

func toPointer[V any](v V) *V {
	return &v
}

func TestPointerMap(t *testing.T) {
	type fields struct {
		PstrPint map[*string]*int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{

		{
			name: "success",
			fields: fields{
				PstrPint: map[*string]*int{
					toPointer("math.MaxInt"): toPointer(math.MaxInt),
					toPointer("math.MinInt"): toPointer(math.MinInt),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PointerMap{
				PstrPint: tt.fields.PstrPint,
			}
			encoded, err := p.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("PointerMap.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &PointerMap{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("PointerMap.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			pd := make(map[string]int)
			for k, v := range p.PstrPint {
				pd[*k] = *v
			}
			decodedd := make(map[string]int)
			for k, v := range decoded.PstrPint {
				decodedd[*k] = *v
			}
			if !reflect.DeepEqual(pd, decodedd) {
				t.Errorf("PointerMap.Decode() = %v, want %v", decodedd, pd)
			}
		})
	}
}

func TestFixedLengths(t *testing.T) {
	type fields struct {
		Map   map[int]uint
		Array [5]uint8
		Slice []int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success_max",
			fields: fields{
				Map: map[int]uint{
					math.MaxInt: math.MaxUint,
				},
				Array: [5]uint8{math.MaxUint8, math.MaxUint8, math.MaxUint8, math.MaxUint8, math.MaxUint8},
				Slice: []int{math.MaxInt, math.MinInt},
			},
		},
		{
			name: "success_min",
			fields: fields{
				Map: map[int]uint{
					math.MinInt: 0,
				},
				Array: [5]uint8{},
				Slice: []int{math.MinInt, math.MaxInt},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FixedLengths{
				Map:   tt.fields.Map,
				Array: tt.fields.Array,
				Slice: tt.fields.Slice,
			}
			encoded, err := f.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixedLengths.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &FixedLengths{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("FixedLengths.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(f, decoded) {
				t.Errorf("FixedLengths.Decode() = %v, want %v", decoded, f)
			}
		})
	}
}

func TestUnnamed(t *testing.T) {
	type fields struct {
		int           int
		uint          *uint
		TestStruct    TestStruct
		TestStructKey *TestStructKey
		Time          time.Time
		Struct        struct {
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
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				int:  12345,
				uint: toPointer[uint](123),
				TestStruct: TestStruct{
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
				TestStructKey: &TestStructKey{
					Int: math.MaxInt,
					Str: "test123",
				},
				Time: now,
				Struct: struct {
					Int int
					Str string
				}{
					Int: math.MinInt,
					Str: "あいうえお",
				},
				Structs: []struct {
					Int     int
					Str     string
					Structs []*struct {
						Int int
						Str string
					}
				}{
					{
						Int: -12345,
						Str: "!\"#$%&'()",
						Structs: []*struct {
							Int int
							Str string
						}{
							{
								Int: -12345,
								Str: "!\"#$%&'()",
							},
							{},
							nil,
						},
					},
					{
						Int:     0,
						Str:     "",
						Structs: nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Unnamed{
				int:           tt.fields.int,
				uint:          tt.fields.uint,
				TestStruct:    tt.fields.TestStruct,
				TestStructKey: tt.fields.TestStructKey,
				Time:          tt.fields.Time,
				Struct:        tt.fields.Struct,
				Structs:       tt.fields.Structs,
			}
			encoded, err := u.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Unnamed.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &Unnamed{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unnamed.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(u, decoded) {
				t.Errorf("Unnamed.Decode() = %v, want %v", decoded, u)
			}
		})
	}
}

func FuzzStrAlias(f *testing.F) {
	f.Add("test123")
	f.Fuzz(func(t *testing.T, str string) {
		s := toPointer(StrAlias(str))
		encoded, err := s.Encode()
		if (err != nil) != false {
			t.Errorf("StrAlias.Encode() error = %v, wantErr %v", err, false)
			return
		}
		decoded := toPointer(StrAlias(""))
		decoded, err = decoded.Decode(encoded)
		if (err != nil) != false {
			t.Errorf("StrAlias.Decode() error = %v, wantErr %v", err, false)
			return
		}
		if !reflect.DeepEqual(s, decoded) {
			t.Errorf("StrAlias.Decode() = %v, want %v", decoded, s)
		}
	})
}

func FuzzIntAlias(f *testing.F) {
	f.Add(123)
	f.Fuzz(func(t *testing.T, i int) {
		s := toPointer(IntAlias(i))
		encoded, err := s.Encode()
		if (err != nil) != false {
			t.Errorf("IntAlias.Encode() error = %v, wantErr %v", err, false)
			return
		}
		decoded := toPointer(IntAlias(0))
		decoded, err = decoded.Decode(encoded)
		if (err != nil) != false {
			t.Errorf("IntAlias.Decode() error = %v, wantErr %v", err, false)
			return
		}
		if !reflect.DeepEqual(s, decoded) {
			t.Errorf("IntAlias.Decode() = %v, want %v", decoded, s)
		}
	})
}

func TestStrsAlias(t *testing.T) {
	tests := []struct {
		name    string
		s       *StrsAlias
		wantErr bool
	}{
		{
			name: "success",
			s: &StrsAlias{
				"test123",
				"あいうえお",
				"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.s.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("StrsAlias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &StrsAlias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrsAlias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.s, decoded) {
				t.Errorf("StrsAlias.Decode() = %v, want %v", decoded, tt.s)
			}
		})
	}
}

func TestPintsAlias(t *testing.T) {
	tests := []struct {
		name    string
		p       *PintsAlias
		wantErr bool
	}{
		{
			name: "success",
			p: &PintsAlias{
				toPointer(math.MaxInt64),
				toPointer(0),
				nil,
				toPointer(math.MinInt64),
				toPointer(math.MaxInt16),
				toPointer(math.MinInt16),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.p.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("PintsAlias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &PintsAlias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("PintsAlias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.p, decoded) {
				t.Errorf("PintsAlias.Decode() = %v, want %v", decoded, tt.p)
			}
		})
	}
}

func TestIntsAlias(t *testing.T) {
	tests := []struct {
		name    string
		i       *IntsAlias
		wantErr bool
	}{
		{
			name: "success",
			i: &IntsAlias{
				math.MaxInt64,
				math.MinInt64,
				0,
				math.MaxInt16,
				math.MinInt16,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.i.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("IntsAlias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &IntsAlias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("IntsAlias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.i, decoded) {
				t.Errorf("IntsAlias.Decode() = %v, want %v", decoded, tt.i)
			}
		})
	}
}

func TestTestStructAlias(t *testing.T) {
	type fields struct {
		Int    int
		Str    string
		Pint   *int
		Strs   []string
		StrInt map[string]int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Int:  -1,
				Str:  "あいうえお",
				Pint: toPointer(0),
				Strs: []string{"test123", "あいうえお"},
				StrInt: map[string]int{
					"math.MaxInt": math.MaxInt,
					"math.MinInt": math.MinInt,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TestStructAlias{
				Int:    tt.fields.Int,
				Str:    tt.fields.Str,
				Pint:   tt.fields.Pint,
				Strs:   tt.fields.Strs,
				StrInt: tt.fields.StrInt,
			}
			encoded, err := tr.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("TestStructAlias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &TestStructAlias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestStructAlias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tr, decoded) {
				t.Errorf("TestStructAlias.Decode() = %v, want %v", decoded, tr)
			}
		})
	}
}

func TestTestStructs(t *testing.T) {
	tests := []struct {
		name    string
		tr      *TestStructs
		wantErr bool
	}{
		{
			name: "success",
			tr: &TestStructs{
				{
					Int:    math.MaxInt,
					Str:    "test123",
					Pint:   nil,
					Strs:   nil,
					StrInt: nil,
				},
				{},
				{
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.tr.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("TestStructs.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &TestStructs{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestStructs.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.tr, decoded) {
				t.Errorf("TestStructs.Decode() = %v, want %v", decoded, tt.tr)
			}
		})
	}
}

func TestPtestStructs(t *testing.T) {
	tests := []struct {
		name    string
		p       *PtestStructs
		wantErr bool
	}{
		{
			name: "success",
			p: &PtestStructs{
				{
					Int:    math.MaxInt,
					Str:    "test123",
					Pint:   nil,
					Strs:   nil,
					StrInt: nil,
				},
				{},
				{
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
				nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.p.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("PtestStructs.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &PtestStructs{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("PtestStructs.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.p, decoded) {
				t.Errorf("PtestStructs.Decode() = %v, want %v", decoded, tt.p)
			}
		})
	}
}

func TestStrTestStruct(t *testing.T) {
	tests := []struct {
		name    string
		s       *StrTestStruct
		wantErr bool
	}{
		{
			name: "success",
			s: &StrTestStruct{
				"": {
					Int:    math.MaxInt,
					Str:    "test123",
					Pint:   nil,
					Strs:   nil,
					StrInt: nil,
				},
				"test123": {},
				"あいうえお": {
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.s.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("StrTestStruct.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &StrTestStruct{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrTestStruct.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.s, decoded) {
				t.Errorf("StrTestStruct.Decode() = %v, want %v", decoded, tt.s)
			}
		})
	}
}

func TestStrTestStructAlias(t *testing.T) {
	tests := []struct {
		name    string
		s       *StrTestStructAlias
		wantErr bool
	}{
		{
			name: "success",
			s: &StrTestStructAlias{
				"": {
					Int:    math.MaxInt,
					Str:    "test123",
					Pint:   nil,
					Strs:   nil,
					StrInt: nil,
				},
				"test123": {},
				"あいうえお": {
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
				"!\"#$%&'()": nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.s.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("StrTestStructAlias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &StrTestStructAlias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrTestStructAlias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.s, decoded) {
				t.Errorf("StrTestStructAlias.Decode() = %v, want %v", decoded, tt.s)
			}
		})
	}
}

func TestAlias(t *testing.T) {
	type fields struct {
		StrAlias        StrAlias
		intAlias        IntAlias
		TestStructAlias TestStructAlias
		TestStructs     TestStructs
		StrTestStruct   StrTestStruct
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				StrAlias: "test123",
				intAlias: math.MaxInt,
				TestStructAlias: TestStructAlias{
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
				},
				TestStructs: TestStructs{
					{},
					{
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
				},
				StrTestStruct: StrTestStruct{
					"": {
						Int:    math.MaxInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
					},
					"test123": {},
					"あいうえお": {
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Alias{
				StrAlias:        tt.fields.StrAlias,
				intAlias:        tt.fields.intAlias,
				TestStructAlias: tt.fields.TestStructAlias,
				TestStructs:     tt.fields.TestStructs,
				StrTestStruct:   tt.fields.StrTestStruct,
			}
			encoded, err := a.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Alias.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &Alias{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Alias.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(a, decoded) {
				t.Errorf("Alias.Decode() = %v, want %v", decoded, a)
			}
		})
	}
}

func TestOtherPackage(t *testing.T) {
	type fields struct {
		Int        int
		Str        string
		TestStruct *other.TestStruct
		other      []other.TestStruct
		StrAlias   example.StrAlias
	}
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Int: math.MaxInt,
				Str: "test123",
				TestStruct: &other.TestStruct{
					Int:  math.MinInt,
					Str:  "あいうえお",
					Pint: toPointer(0),
					Strs: []string{"test123", "あいうえお"},
					StrInt: map[string]int{
						"math.MaxInt": math.MaxInt,
						"math.MinInt": math.MinInt,
					},
					Time: now,
				},
				other: []other.TestStruct{
					{
						Int:    math.MaxInt,
						Str:    "test123",
						Pint:   nil,
						Strs:   nil,
						StrInt: nil,
						Time:   time.Time{},
					},
					{},
					{
						Int:  math.MinInt,
						Str:  "あいうえお",
						Pint: toPointer(0),
						Strs: []string{"test123", "あいうえお"},
						StrInt: map[string]int{
							"math.MaxInt": math.MaxInt,
							"math.MinInt": math.MinInt,
						},
						Time: now,
					},
				},
				StrAlias: "あいうえお",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OtherPackage{
				Int:        tt.fields.Int,
				Str:        tt.fields.Str,
				TestStruct: tt.fields.TestStruct,
				other:      tt.fields.other,
				StrAlias:   tt.fields.StrAlias,
			}
			encoded, err := o.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("OtherPackage.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			decoded := &OtherPackage{}
			decoded, err = decoded.Decode(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("OtherPackage.Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(o, decoded) {
				t.Errorf("OtherPackage.Decode() = %v, want %v", decoded, o)
			}
		})
	}
}
