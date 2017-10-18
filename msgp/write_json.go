package msgp

import (
	"strconv"
	"time"
	"encoding/json"
	"bytes"
)

type MarshalerJSON interface {
	MarshalBufferJSON(b []byte) ([]byte, error)
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
	o := b
	o = append(o, '[')
	for i, v := range s {
		if i != 0 {
			o = append(o, ',')
		}
		var err error
		o, err = AppendByteJSON(o, v)
		if err != nil {
			return nil, err
		}
	}
	o = append(o, ']')
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
	o, err = AppendFloat32JSON(o, real(c))
	if err != nil {
		return nil, err
	}
	o = append(o, ',')
	o, err = AppendFloat32JSON(o, imag(c))
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

	if m, ok := i.(Extension); ok {
		o := b
		o = append(o, '{', '"', 't', 'y', 'p', 'e', '"', ':')
		o, err := AppendInt8JSON(o, m.ExtensionType())
		if err != nil {
			return nil, err
		}
		o = append(o, ',', '"', 'd', 'a', 't', 'a', '"', ':')
		d := make([]byte, 0, m.Len())
		err = m.MarshalBinaryTo(d)
		if err != nil {
			return nil, err
		}
		o, err = AppendBytesJSON(o, d)
		if err != nil {
			return nil, err
		}
		o = append(o, '}')
		return o, nil
	}

	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return append(b, data...), nil
}

func AppendExtensionJSON(b []byte, o Extension) ([]byte, error) {
	return AppendIntfJSON(b, o)
}

func (r Raw) MarshalBufferJSON(b []byte) ([]byte, error) {
	if len(r) == 0 {
		return AppendNilJSON(b), nil
	}
	buf := bytes.NewBuffer(nil)
	_, err := UnmarshalAsJSON(buf, r)
	if err != nil {
		return nil, err
	}
	return append(b, buf.Bytes()...), nil
}

func EscapeJSON(b []byte, s string) []byte {
	return append(b, []byte(s) ...)
}
