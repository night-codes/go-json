package decoder

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/night-codes/go-json/internal/errors"
	"github.com/night-codes/go-json/internal/runtime"
)

type uintDecoder struct {
	stringDecoder *stringDecoder
	typ           *runtime.Type
	kind          reflect.Kind
	op            func(unsafe.Pointer, uint64)
	structName    string
	fieldName     string
}

func newUintDecoder(typ *runtime.Type, structName, fieldName string, op func(unsafe.Pointer, uint64)) *uintDecoder {
	return &uintDecoder{
		stringDecoder: newStringDecoder(structName, fieldName),
		typ:           typ,
		kind:          typ.Kind(),
		op:            op,
		structName:    structName,
		fieldName:     fieldName,
	}
}

func (d *uintDecoder) typeError(buf []byte, offset int64) *errors.UnmarshalTypeError {
	return &errors.UnmarshalTypeError{
		Value:  fmt.Sprintf("number %s", string(buf)),
		Type:   runtime.RType2Type(d.typ),
		Offset: offset,
	}
}

var (
	pow10u64 = [...]uint64{
		1e00, 1e01, 1e02, 1e03, 1e04, 1e05, 1e06, 1e07, 1e08, 1e09,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	}
	pow10u64Len = len(pow10u64)
)

func (d *uintDecoder) decodeStreamByte(s *Stream) ([]byte, error) {
	for {
		switch s.char() {
		case ' ', '\n', '\t', '\r':
			s.cursor++
			continue
		case '0':
			s.cursor++
			return numZeroBuf, nil
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			start := s.cursor
			for {
				s.cursor++
				if numTable[s.char()] {
					continue
				} else if s.char() == nul {
					if s.read() {
						s.cursor-- // for retry current character
						continue
					}
				}
				break
			}
			num := s.buf[start:s.cursor]
			return num, nil
		case '"':
			return d.stringDecoder.decodeStreamByte(s)
		case 'n':
			if err := nullBytes(s); err != nil {
				return nil, err
			}
			return nil, nil
		case nul:
			if s.read() {
				continue
			}
		default:
			return nil, d.typeError([]byte{s.char()}, s.totalOffset())
		}
		break
	}
	return nil, errors.ErrUnexpectedEndOfJSON("number(unsigned integer)", s.totalOffset())
}

func (d *uintDecoder) decodeByte(buf []byte, cursor int64) ([]byte, int64, error) {
	for {
		switch buf[cursor] {
		case ' ', '\n', '\t', '\r':
			cursor++
			continue
		case '0':
			cursor++
			return numZeroBuf, cursor, nil
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			start := cursor
			cursor++
			for numTable[buf[cursor]] {
				cursor++
			}
			num := buf[start:cursor]
			return num, cursor, nil
		case 'n':
			if err := validateNull(buf, cursor); err != nil {
				return nil, 0, err
			}
			cursor += 4
			return nil, cursor, nil
		case '"':
			return d.stringDecoder.decodeByte(buf, cursor)
		default:
			return nil, 0, d.typeError([]byte{buf[cursor]}, cursor)
		}
	}
}

func (d *uintDecoder) DecodeStream(s *Stream, depth int64, p unsafe.Pointer) error {
	bytes, err := d.decodeStreamByte(s)
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	str := *(*string)(unsafe.Pointer(&bytes))
	u64, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return d.typeError(bytes, s.totalOffset())
	}
	switch d.kind {
	case reflect.Uint8:
		if (1 << 8) <= u64 {
			return d.typeError(bytes, s.totalOffset())
		}
	case reflect.Uint16:
		if (1 << 16) <= u64 {
			return d.typeError(bytes, s.totalOffset())
		}
	case reflect.Uint32:
		if (1 << 32) <= u64 {
			return d.typeError(bytes, s.totalOffset())
		}
	}
	d.op(p, u64)
	return nil
}

func (d *uintDecoder) Decode(ctx *RuntimeContext, cursor, depth int64, p unsafe.Pointer) (int64, error) {
	bytes, c, err := d.decodeByte(ctx.Buf, cursor)
	if err != nil {
		return 0, err
	}
	if len(bytes) == 0 {
		return c, nil
	}
	cursor = c
	str := *(*string)(unsafe.Pointer(&bytes))
	u64, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, d.typeError(bytes, cursor)
	}
	switch d.kind {
	case reflect.Uint8:
		if (1 << 8) <= u64 {
			return 0, d.typeError(bytes, cursor)
		}
	case reflect.Uint16:
		if (1 << 16) <= u64 {
			return 0, d.typeError(bytes, cursor)
		}
	case reflect.Uint32:
		if (1 << 32) <= u64 {
			return 0, d.typeError(bytes, cursor)
		}
	}
	d.op(p, u64)
	return cursor, nil
}
