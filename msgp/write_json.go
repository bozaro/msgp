package msgp

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	UintSizeJSON       = Int64SizeJSON
	Uint8SizeJSON      = 3
	Uint16SizeJSON     = 5
	Uint32SizeJSON     = 10
	Uint64SizeJSON     = 20
	IntSizeJSON        = UintSizeJSON + 1
	Int8SizeJSON       = Uint8SizeJSON + 1
	Int16SizeJSON      = Uint16SizeJSON + 1
	Int32SizeJSON      = Uint32SizeJSON + 1
	Int64SizeJSON      = Uint64SizeJSON + 1
	ByteSizeJSON       = Uint8SizeJSON
	Float32SizeJSON    = 8
	Float64SizeJSON    = 8
	Complex64SizeJSON  = 3 + Float32SizeJSON*2
	Complex128SizeJSON = 3 + Float64SizeJSON*2

	TimeSizeJSON = 30 // 2005-08-09T18:31:42.000-03:00
	BoolSizeJSON = 5  // false
	NilSizeJSON  = 4  // null
)

type MarshalerJSON interface {
	MarshalBufferJSON(b []byte) ([]byte, error)
}

type SizerJSON interface {
	SizeJSON() int
}

func AppendNilJSON(b []byte) []byte {
	return append(b, 'n', 'u', 'l', 'l')
}

func AppendFloat32JSON(b []byte, f float32) ([]byte, error) {
	return strconv.AppendFloat(b, float64(f), 'f', -1, 32), nil
}

func AppendFloat64JSON(b []byte, f float64) ([]byte, error) {
	return strconv.AppendFloat(b, f, 'f', -1, 64), nil
}

// AppendInt appends an int to the slice
func AppendIntJSON(b []byte, i int) ([]byte, error) { return AppendInt64JSON(b, int64(i)) }

// AppendInt8 appends an int8 to the slice
func AppendInt8JSON(b []byte, i int8) ([]byte, error) { return AppendInt64JSON(b, int64(i)) }

// AppendInt16 appends an int16 to the slice
func AppendInt16JSON(b []byte, i int16) ([]byte, error) { return AppendInt64JSON(b, int64(i)) }

// AppendInt32 appends an int32 to the slice
func AppendInt32JSON(b []byte, i int32) ([]byte, error) { return AppendInt64JSON(b, int64(i)) }

func AppendInt64JSON(b []byte, u int64) ([]byte, error) {
	return strconv.AppendInt(b, u, 10), nil
}

// AppendUint appends a uint to the slice
func AppendUintJSON(b []byte, u uint) ([]byte, error) { return AppendUint64JSON(b, uint64(u)) }

// AppendUint8 appends a uint8 to the slice
func AppendUint8JSON(b []byte, u uint8) ([]byte, error) { return AppendUint64JSON(b, uint64(u)) }

// AppendByte is analogous to AppendUint8
func AppendByteJSON(b []byte, u byte) ([]byte, error) { return AppendUint8JSON(b, uint8(u)) }

// AppendUint16 appends a uint16 to the slice
func AppendUint16JSON(b []byte, u uint16) ([]byte, error) { return AppendUint64JSON(b, uint64(u)) }

// AppendUint32 appends a uint32 to the slice
func AppendUint32JSON(b []byte, u uint32) ([]byte, error) { return AppendUint64JSON(b, uint64(u)) }

func AppendUint64JSON(b []byte, u uint64) ([]byte, error) {
	return strconv.AppendUint(b, u, 10), nil
}

func AppendTimeJSON(b []byte, t time.Time) ([]byte, error) {
	bts, err := t.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return append(b, bts...), nil
}

func AppendBytesJSON(b []byte, s []byte) ([]byte, error) {
	size := base64.StdEncoding.EncodedLen(len(s))
	o := Require(b, size+2)
	o = append(o, '"')
	beg := len(o)
	end := beg + size
	for i := beg; i < end; i++ {
		o = append(o, 0)
	}
	base64.StdEncoding.Encode(o[beg:end], s)
	o = append(o, '"')
	return o, nil
}

func AppendBoolJSON(b []byte, v bool) ([]byte, error) {
	if v {
		return append(b, 't', 'r', 'u', 'e'), nil
	} else {
		return append(b, 'f', 'a', 'l', 's', 'e'), nil
	}
}

func AppendStringJSON(b []byte, s string) ([]byte, error) {
	o := b
	o = append(o, '"')
	o = EscapeJSON(o, s)
	o = append(o, '"')
	return o, nil
}

func AppendComplex64JSON(b []byte, c complex64) ([]byte, error) {
	var err error
	o := b
	o = append(o, '[')
	o, err = AppendFloat32JSON(o, float32(real(c)))
	if err != nil {
		return nil, err
	}
	o = append(o, ',')
	o, err = AppendFloat32JSON(o, float32(imag(c)))
	if err != nil {
		return nil, err
	}
	o = append(o, ']')
	return o, nil
}

func AppendComplex128JSON(b []byte, c complex128) ([]byte, error) {
	var err error
	o := b
	o = append(o, '[')
	o, err = AppendFloat64JSON(o, real(c))
	if err != nil {
		return nil, err
	}
	o = append(o, ',')
	o, err = AppendFloat64JSON(o, imag(c))
	if err != nil {
		return nil, err
	}
	o = append(o, ']')
	return o, nil
}

func AppendIntfJSON(b []byte, i interface{}) ([]byte, error) {
	if i == nil {
		return AppendNilJSON(b), nil
	}

	if m, ok := i.(MarshalerJSON); ok {
		return m.MarshalBufferJSON(b)
	}

	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return append(b, data...), nil
}

func GuessSizeJSON(i interface{}) int {
	if i == nil {
		return NilSizeJSON
	}

	if m, ok := i.(SizerJSON); ok {
		return m.SizeJSON()
	}

	return 10
}

func AppendExtensionJSON(b []byte, o Extension) ([]byte, error) {
	return AppendIntfJSON(b, o)
}

func (r Raw) SizeJSON() int {
	return BytesSizeJSON(r)
}

func (r Raw) MarshalBufferJSON(b []byte) ([]byte, error) {
	return AppendBytesJSON(b, r)
}

func (r RawExtension) SizeJSON() int {
	return 20 + Int8SizeJSON + BytesSizeJSON(r.Data)
}

func (r RawExtension) MarshalBufferJSON(b []byte) ([]byte, error) {
	o := b
	o = append(o, '{', '"', 't', 'y', 'p', 'e', '"', ':')
	o, err := AppendInt8JSON(o, r.Type)
	if err != nil {
		return nil, err
	}
	o = append(o, ',', '"', 'd', 'a', 't', 'a', '"', ':')
	o, err = AppendBytesJSON(o, r.Data)
	if err != nil {
		return nil, err
	}
	o = append(o, '}')
	return o, nil
}

func (n Number) SizeJSON() int {
	switch n.typ {
	case IntType:
		return Int64SizeJSON
	case UintType:
		return Uint64SizeJSON
	case Float64Type:
		return Float64SizeJSON
	case Float32Type:
		return Float32SizeJSON
	default:
		return Int64SizeJSON
	}
}

func (n *Number) MarshalBufferJSON(b []byte) ([]byte, error) {
	switch n.typ {
	case IntType:
		return AppendInt64JSON(b, int64(n.bits))
	case UintType:
		return AppendUint64JSON(b, uint64(n.bits))
	case Float64Type:
		return AppendFloat64JSON(b, math.Float64frombits(n.bits))
	case Float32Type:
		return AppendFloat32JSON(b, math.Float32frombits(uint32(n.bits)))
	default:
		return AppendInt64JSON(b, 0)
	}
}

func isNotEscapedSingleChar(c byte) bool {
	return c != '\\' && c != '"' && c >= 0x20 && c < utf8.RuneSelf
}

func EscapeJSON(b []byte, s string) []byte {
	p := 0 // last non-escape symbol
	o := b

	for i := 0; i < len(s); {
		c := s[i]

		if isNotEscapedSingleChar(c) {
			// single-width character, no escaping is required
			i++
			continue
		} else if c < utf8.RuneSelf {
			// single-with character, need to escape
			o = append(o, s[p:i]...)
			switch c {
			case '\t':
				o = append(o, '\\', 't')
			case '\r':
				o = append(o, '\\', 'r')
			case '\n':
				o = append(o, '\\', 'n')
			case '\\':
				o = append(o, '\\', '\\')
			case '"':
				o = append(o, '\\', '"')
			default:
				o = append(o, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
			}

			i++
			p = i
			continue
		}

		// broken utf
		runeValue, runeWidth := utf8.DecodeRuneInString(s[i:])
		if runeValue == utf8.RuneError && runeWidth == 1 {
			o = append(o, s[p:i]...)
			o = append(o, '\\', 'u', 'f', 'f', 'f', 'd')
			i++
			p = i
			continue
		}

		// jsonp stuff - tab separator and line separator
		if runeValue == '\u2028' || runeValue == '\u2029' {
			o = append(o, s[p:i]...)
			o = append(o, '\\', 'u', '2', '0', '2', hex[runeValue&0xF])
			i += runeWidth
			p = i
			continue
		}
		i += runeWidth
	}
	o = append(o, s[p:]...)
	return o
}

func BytesSizeJSON(b []byte) int {
	return 2 + (len(b)*4+2)/3
}

func StringSizeJSON(s string) int {
	return 2 + len(s)
}
