package gen

import (
	"io"
	"text/template"
)

var (
	marshalTestTempl = template.New("MarshalTest")
	encodeTestTempl  = template.New("EncodeTest")
	jsonTestTempl    = template.New("MarshalJsonTest")
)

// TODO(philhofer):
// for simplicity's sake, right now
// we can only generate tests for types
// that can be initialized with the
// "Type{}" syntax.
// we should support all the types.

func mtest(w io.Writer) *mtestGen {
	return &mtestGen{w: w}
}

type mtestGen struct {
	passes
	w io.Writer
}

func (m *mtestGen) Tags() []string {
	return msgTags
}

func (m *mtestGen) IsTests() bool {
	return true
}

func (m *mtestGen) Imports() []string {
	return []string{}
}

func (m *mtestGen) Execute(p Elem) error {
	p = m.applyall(p)
	if p != nil && IsPrintable(p) {
		switch p.(type) {
		case *Struct, *Array, *Slice, *Map:
			return marshalTestTempl.Execute(m.w, p)
		}
	}
	return nil
}

func (m *mtestGen) Method() Method { return marshaltest }

type etestGen struct {
	passes
	w io.Writer
}

func etest(w io.Writer) *etestGen {
	return &etestGen{w: w}
}

func (e *etestGen) Execute(p Elem) error {
	p = e.applyall(p)
	if p != nil && IsPrintable(p) {
		switch p.(type) {
		case *Struct, *Array, *Slice, *Map:
			return encodeTestTempl.Execute(e.w, p)
		}
	}
	return nil
}

func (s *etestGen) Tags() []string {
	return msgTags
}

func (e *etestGen) IsTests() bool {
	return true
}

func (e *etestGen) Imports() []string {
	return []string{
		"bytes",
	}
}

func (e *etestGen) Method() Method { return encodetest }

type jtestGen struct {
	passes
	w io.Writer
}

func jtest(w io.Writer) *jtestGen {
	return &jtestGen{w: w}
}

func (e *jtestGen) Execute(p Elem) error {
	p = e.applyall(p)
	if p != nil && IsPrintable(p) {
		switch p.(type) {
		case *Struct, *Array, *Slice, *Map:
			return jsonTestTempl.Execute(e.w, p)
		}
	}
	return nil
}

func (e *jtestGen) Tags() []string {
	return jsonTags
}

func (e *jtestGen) IsTests() bool {
	return true
}

func (e *jtestGen) Imports() []string {
	return []string{}
}

func (e *jtestGen) Method() Method { return jsontest }

func init() {
	template.Must(marshalTestTempl.Parse(`func TestMarshalUnmarshal{{.TypeName}}(t *testing.T) {
	v := {{.TypeName}}{}
	bts, err := v.MarshalMsg(nil)
	if err != nil {
		t.Fatal(err)
	}
	left, err := v.UnmarshalMsg(bts)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) > 0 {
		t.Errorf("%d bytes left over after UnmarshalMsg(): %q", len(left), left)
	}

	left, err = msgp.Skip(bts)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) > 0 {
		t.Errorf("%d bytes left over after Skip(): %q", len(left), left)
	}
}

func BenchmarkMarshalMsg{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		v.MarshalMsg(nil)
	}
}

func BenchmarkAppendMsg{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	bts := make([]byte, 0, v.Msgsize())
	bts, _ = v.MarshalMsg(bts[0:0])
	b.SetBytes(int64(len(bts)))
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		bts, _ = v.MarshalMsg(bts[0:0])
	}
}

func BenchmarkUnmarshal{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	bts, _ := v.MarshalMsg(nil)
	b.ReportAllocs()
	b.SetBytes(int64(len(bts)))
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		_, err := v.UnmarshalMsg(bts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

`))

	template.Must(encodeTestTempl.Parse(`func TestEncodeDecode{{.TypeName}}(t *testing.T) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer
	msgp.Encode(&buf, &v)

	m := v.Msgsize()
	if buf.Len() > m {
		t.Logf("WARNING: Msgsize() for %v is inaccurate", v)
	}

	vn := {{.TypeName}}{}
	err := msgp.Decode(&buf, &vn)
	if err != nil {
		t.Error(err)
	}

	buf.Reset()
	msgp.Encode(&buf, &v)
	err = msgp.NewReader(&buf).Skip()
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkEncode{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer
	msgp.Encode(&buf, &v)
	b.SetBytes(int64(buf.Len()))
	en := msgp.NewWriter(msgp.Nowhere)
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		v.EncodeMsg(en)
	}
	en.Flush()
}

func BenchmarkDecode{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	var buf bytes.Buffer
	msgp.Encode(&buf, &v)
	b.SetBytes(int64(buf.Len()))
	rd := msgp.NewEndlessReader(buf.Bytes(), b)
	dc := msgp.NewReader(rd)
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		err := v.DecodeMsg(dc)
		if  err != nil {
			b.Fatal(err)
		}
	}
}

`))

	template.Must(jsonTestTempl.Parse(`func TestMarshalUnmarshalJSON{{.TypeName}}(t *testing.T) {
	v := {{.TypeName}}{}
	buf, err := v.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	var raw interface{}
	err = json.Unmarshal(buf, &raw)
	if err != nil {
		t.Log(string(buf))
		t.Error(err)
	}

	m := v.SizeJSON()
	if len(buf) > m {
		t.Logf("WARNING: SizeJSON() for %v is inaccurate\nJSON [%d -> %d]: %s", v, m, len(buf), string(buf))
	}

	vn := {{.TypeName}}{}
	err = vn.UnmarshalJSON(buf)
	if err != nil {
		t.Logf("JSON: %s", string(buf))
		t.Error(err)
	}
}

func BenchmarkMarshalJSON{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	buf, err := v.MarshalJSON()
	if err != nil {
		b.Error(err)
	}
	data := make([]byte, 0, v.SizeJSON())

	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		data = data[:0]
		_, err = v.MarshalBufferJSON(data)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkUnmarshalJSON{{.TypeName}}(b *testing.B) {
	v := {{.TypeName}}{}
	buf, err := v.MarshalJSON()
	if err != nil {
		b.Error(err)
	}
	b.SetBytes(int64(len(buf)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o {{.TypeName}}
		err = o.UnmarshalJSON(buf)
		if err != nil {
			b.Error(err)
		}
	}
}

`))
}
