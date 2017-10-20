package msgp

import (
	"encoding/json"
	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jlexer"
	"time"
)

const (
	// RFC3339Millis represents a ISO8601 format to millis instead of to nanos
	RFC3339Millis = "2006-01-02T15:04:05.000Z07:00"
	// RFC3339Micro represents a ISO8601 format to micro instead of to nano
	RFC3339Micro = "2006-01-02T15:04:05.000000Z07:00"
)

var (
	dateTimeFormats = []string{RFC3339Micro, RFC3339Millis, time.RFC3339, time.RFC3339Nano}
)

func ReadFloat32JSON(l *jlexer.Lexer) float32 {
	return l.Float32()
}

func ReadFloat64JSON(l *jlexer.Lexer) float64 {
	return l.Float64()
}

// ReadInt appends an int to the slice
func ReadIntJSON(l *jlexer.Lexer) int {
	return l.Int()
}

// ReadInt8 appends an int8 to the slice
func ReadInt8JSON(l *jlexer.Lexer) int8 {
	return l.Int8()
}

// ReadInt16 appends an int16 to the slice
func ReadInt16JSON(l *jlexer.Lexer) int16 {
	return l.Int16()
}

// ReadInt32 appends an int32 to the slice
func ReadInt32JSON(l *jlexer.Lexer) int32 { return l.Int32() }

func ReadInt64JSON(l *jlexer.Lexer) int64 {
	return l.Int64()
}

// ReadUint appends a uint to the slice
func ReadUintJSON(l *jlexer.Lexer) uint {
	return l.Uint()
}

// ReadUint8 appends a uint8 to the slice
func ReadUint8JSON(l *jlexer.Lexer) uint8 {
	return l.Uint8()
}

// ReadByte is analogous to ReadUint8
func ReadByteJSON(l *jlexer.Lexer) byte {
	return l.Uint8()
}

// ReadUint16 appends a uint16 to the slice
func ReadUint16JSON(l *jlexer.Lexer) uint16 {
	return l.Uint16()
}

// ReadUint32 appends a uint32 to the slice
func ReadUint32JSON(l *jlexer.Lexer) uint32 {
	return l.Uint32()
}

func ReadUint64JSON(l *jlexer.Lexer) uint64 {
	return l.Uint64()
}

// ParseDateTime parses a string that represents an ISO8601 time or a unix epoch
func ParseDateTime(data string) (time.Time, error) {
	if data == "" {
		return time.Time{}, nil
	}
	var lastError error
	for _, layout := range dateTimeFormats {
		dd, err := time.Parse(layout, data)
		if err != nil {
			lastError = err
			continue
		}
		lastError = nil
		return dd, nil
	}
	return time.Time{}, lastError
}

func ReadTimeJSON(l *jlexer.Lexer) time.Time {
	if l.IsNull() {
		return time.Time{}
	}
	t, err := ParseDateTime(l.String())
	if err != nil {
		l.AddError(err)
		return time.Time{}
	}
	return t
}

func ReadExactBytesJSON(l *jlexer.Lexer, into []byte) {
	read := l.Bytes()
	if !l.Ok() {
		return
	}
	if len(read) != len(into) {
		l.AddError(ArrayError{Wanted: uint32(len(into)), Got: uint32(len(read))})
		return
	}
	copy(into, read)
}

func ReadBytesJSON(l *jlexer.Lexer) []byte {
	return l.Bytes()
}

func ReadBoolJSON(l *jlexer.Lexer) bool {
	return l.Bool()
}

func ReadStringJSON(l *jlexer.Lexer) string {
	return l.String()
}

func ReadComplex64JSON(l *jlexer.Lexer) complex64 {
	l.Delim('[')
	real := l.Float32()
	l.WantComma()
	imag := l.Float32()
	l.WantComma()
	l.Delim(']')
	return complex(real, imag)
}

func ReadComplex128JSON(l *jlexer.Lexer) complex128 {
	l.Delim('[')
	real := l.Float64()
	l.WantComma()
	imag := l.Float64()
	l.WantComma()
	l.Delim(']')
	return complex(real, imag)
}

func ReadIntfJSON(l *jlexer.Lexer, i interface{}) {
	if !l.Ok() {
		return
	}

	if m, ok := i.(easyjson.Unmarshaler); ok {
		m.UnmarshalEasyJSON(l)
		return
	}

	raw := l.Raw()
	if !l.Ok() {
		return
	}
	err := json.Unmarshal(raw, i)
	if err != nil {
		l.AddError(err)
	}
}

func ReadExtensionJSON(l *jlexer.Lexer, o Extension) {
	ReadIntfJSON(l, o)
}

func (r *Raw) UnmarshalEasyJSON(l *jlexer.Lexer) {
	*r = Raw(ReadBytesJSON(l))
}

func (r *RawExtension) UnmarshalEasyJSON(l *jlexer.Lexer) {
	l.Delim('{')
	raw := RawExtension{}
	for !l.IsDelim('}') {
		field := l.UnsafeString()
		l.WantColon()
		switch field {
		case "type":
			raw.Type = l.Int8()
		case "data":
			raw.Data = l.Bytes()
		default:
			l.Skip()
		}
		l.WantComma()
	}
	l.Delim('}')
	*r = raw
}

func (n *Number) UnmarshalEasyJSON(l *jlexer.Lexer) {
	if !l.Ok() {
		return
	}
	n.AsFloat64(l.Float64())
}
