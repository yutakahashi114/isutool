// Code generated by github.com/yutakahashi114/isutool; DO NOT EDIT.
package example

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

var (
	_ = math.MaxUint8
)

func (s StrAlias) Encode() ([]byte, error) {
	out := make([]byte, s.MaxSize())

	_n, err := s.EncodeTo(out)
	if err != nil {
		return nil, err
	}

	return out[:_n], nil
}

func (s StrAlias) Decode(in []byte) (StrAlias, error) {
	_, err := DecodeStrAlias(in, &s)
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s StrAlias) MaxSize() int {
	_size := 0

	_size += binary.MaxVarintLen64 + len(s)
	return _size
}

func (s StrAlias) EncodeTo(out []byte) (int, error) {
	_timeMarshalBinary := func(t time.Time, out []byte) (int, error) {
		var timeZero = time.Time{}.Unix()

		// cf. https://github.com/golang/go/blob/dc00aed6de101700fd02b30f93789b9e9e1fe9a1/src/time/time.go#L1206
		var offsetMin int16 // minutes east of UTC. -1 is UTC.
		var offsetSec int8
		version := 1

		if t.Location() == time.UTC {
			offsetMin = -1
		} else {
			_, offset := t.Zone()
			if offset%60 != 0 {
				version = 2
				offsetSec = int8(offset % 60)
			}

			offset /= 60
			if offset < -32768 || offset == -1 || offset > 32767 {
				return 0, errors.New("TimeMarshalBinary: unexpected zone offset")
			}
			offsetMin = int16(offset)
		}

		unix := t.Unix()
		sec := unix - timeZero
		nsec := t.UnixNano() - unix*1000000000
		out[0] = byte(version)   // byte 0 : version
		out[1] = byte(sec >> 56) // bytes 1-8: seconds
		out[2] = byte(sec >> 48)
		out[3] = byte(sec >> 40)
		out[4] = byte(sec >> 32)
		out[5] = byte(sec >> 24)
		out[6] = byte(sec >> 16)
		out[7] = byte(sec >> 8)
		out[8] = byte(sec)
		out[9] = byte(nsec >> 24) // bytes 9-12: nanoseconds
		out[10] = byte(nsec >> 16)
		out[11] = byte(nsec >> 8)
		out[12] = byte(nsec)
		out[13] = byte(offsetMin >> 8) // bytes 13-14: zone offset in minutes
		out[14] = byte(offsetMin)

		if version == 2 {
			out[15] = byte(offsetSec)
		}

		return 16, nil
	}
	_ = _timeMarshalBinary

	_n := 0

	_n += binary.PutVarint(out[_n:], int64(len(s)))
	_n += copy(out[_n:], s)

	return _n, nil
}

func DecodeStrAlias(in []byte, s *StrAlias) (_n int, err error) {

	_ƒçsåLen, _ƒçsåLenSize := binary.Varint(in[_n:])
	_n += _ƒçsåLenSize
	(*s) = StrAlias(in[_n : _n+int(_ƒçsåLen)])
	_n += int(_ƒçsåLen)
	return _n, nil
}
