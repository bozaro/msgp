package gen

import (
	"fmt"
	"io"

	"github.com/tinylib/msgp/msgp"
)

func marshalJSON(w io.Writer) *marshalJSONGen {
	return &marshalJSONGen{
		p: printer{w: w},
	}
}

type marshalJSONGen struct {
	passes
	p    printer
	fuse []byte
}

func (m *marshalJSONGen) Method() Method { return MarshalJSON }

func (m *marshalJSONGen) Apply(dirs []string) error {
	return nil
}

func (m *marshalJSONGen) Execute(p Elem) error {
	if !m.p.ok() {
		return m.p.err
	}
	p = m.applyall(p)
	if p == nil {
		return nil
	}
	if !IsPrintable(p) {
		return nil
	}

	m.p.comment("MarshalJSON implements json.Marshaler")

	// save the vname before
	// calling methodReceiver so
	// that z.Msgsize() is printed correctly
	c := p.Varname()

	m.p.printf("\nfunc (%s %s) MarshalJSON() ([]byte, error) {", c, imutMethodReceiver(p))
	m.p.printf("\n	return %s.MarshalBufferJSON(make([]byte, 0, 1024))", c)
	m.p.closeblock()
	m.p.printf("\n")

	m.p.printf("\nfunc (%s %s) MarshalBufferJSON(b []byte) (o []byte, err error) {", c, imutMethodReceiver(p))
	m.p.printf("\no = b")
	next(m, p)
	m.fuseHook()
	m.p.nakedReturn()
	return m.p.err
}

func (m *marshalJSONGen) rawAppend(typ string, argfmt string, arg interface{}) {
	m.fuseHook()
	m.p.printf("\no, err = msgp.Append%sJSON(o, %s)", typ, fmt.Sprintf(argfmt, arg))
	m.p.print("\nif err != nil { return nil, err }")
}

func (m *marshalJSONGen) fuseHook() {
	if len(m.fuse) > 0 {
		m.p.printf("\n// write %q", string(m.fuse))
		m.rawbytes(m.fuse)
		m.fuse = m.fuse[:0]
	}
}

func (m *marshalJSONGen) Fuse(s string) {
	if len(m.fuse) == 0 {
		m.fuse = []byte(s)
	} else {
		m.fuse = append(m.fuse, []byte(s)...)
	}
}

func (m *marshalJSONGen) escape(s string) string {
	return string(msgp.EscapeJSON(nil, s))
}

func (m *marshalJSONGen) gStruct(s *Struct) {
	if !m.p.ok() {
		return
	}

	if s.AsTuple {
		m.tuple(s)
	} else {
		m.mapstruct(s)
	}
	return
}

func (m *marshalJSONGen) tuple(s *Struct) {
	m.p.printf("\n// array header", len(s.Fields))
	m.Fuse("[")
	for i := range s.Fields {
		if !m.p.ok() {
			return
		}
		if i > 0 {
			m.Fuse(",")
		}
		next(m, s.Fields[i].FieldElem)
	}
	m.Fuse("]")
}

func (m *marshalJSONGen) mapstruct(s *Struct) {
	omit := &omitemptyGen{p: m.p}
	m.p.printf("\n// map header")
	m.Fuse("{")
	hasData := false
	hasDataFlag := ""
	for i, field := range s.Fields {
		if !m.p.ok() {
			return
		}
		m.p.printf("\n// field %q", field.FieldTag)
		omit.expr = ""
		if field.OmitEmpty {
			next(omit, field.FieldElem)
			if omit.expr != "" {
				if !hasData && hasDataFlag == "" && (i < len(s.Fields)-1) {
					hasDataFlag = randIdent()
					m.p.printf("\n%s := false", hasDataFlag)
				}
				m.fuseHook()
				m.p.printf("\nif !%s {", omit.expr)
			}
		}
		if i > 0 {
			if hasData {
				m.Fuse(",")
			} else {
				m.fuseHook()
				m.p.printf("\nif %s {", hasDataFlag)
				m.Fuse(",")
				m.fuseHook()
				m.p.closeblock()
			}
		}
		m.Fuse(fmt.Sprintf("\"%s\":", m.escape(field.FieldTag)))

		next(m, s.Fields[i].FieldElem)
		if omit.expr != "" {
			m.fuseHook()
			if !hasData && (hasDataFlag != "") {
				m.p.printf("\n%s = true", hasDataFlag)
			}
			m.p.closeblock()
		} else {
			hasData = true
		}
	}
	m.Fuse("}")
}

// append raw data
func (m *marshalJSONGen) rawbytes(bts []byte) {
	m.p.print("\no = append(o, ")
	for _, b := range bts {
		m.p.printf("0x%x,", b)
	}
	m.p.print(")")
}

func (m *marshalJSONGen) gMap(s *Map) {
	if !m.p.ok() {
		return
	}
	hasDataFlag := randIdent()
	m.Fuse("{")
	m.fuseHook()
	vname := s.Varname()
	m.p.printf("\n%s := false", hasDataFlag)
	m.p.printf("\nfor %s, %s := range %s {", s.Keyidx, s.Validx, vname)
	m.p.printf("\nif %s {", hasDataFlag)
	m.Fuse(",")
	m.fuseHook()
	m.p.closeblock()
	m.p.printf("\n%s = true", hasDataFlag)
	m.rawAppend(stringTyp, literalFmt, s.Keyidx)
	m.Fuse(":")
	next(m, s.Value)
	m.fuseHook()
	m.p.closeblock()
	m.Fuse("}")
}

func (m *marshalJSONGen) gSlice(s *Slice) {
	m.rangeSlice(s.Index, s.Varname(), s.Els)
}

func (m *marshalJSONGen) gArray(a *Array) {
	m.rangeSlice(a.Index, a.Varname(), a.Els)
}

func (m *marshalJSONGen) rangeSlice(idx string, iter string, inner Elem) {
	if !m.p.ok() {
		return
	}
	if be, ok := inner.(*BaseElem); ok && be.Value == Byte {
		m.rawAppend("Bytes", "(%s)[:]", iter)
		return
	}
	m.Fuse("[")
	m.fuseHook()
	m.p.printf("\n for %s := range %s {", idx, iter)
	m.p.printf("\n if %s != 0 {", idx)
	m.Fuse(",")
	m.fuseHook()
	m.p.closeblock()
	m.fuseHook()
	next(m, inner)
	m.fuseHook()
	m.p.closeblock()
	m.Fuse("]")
}

func (m *marshalJSONGen) gPtr(p *Ptr) {
	if !m.p.ok() {
		return
	}
	m.fuseHook()
	m.p.printf("\nif %s == nil {\no = msgp.AppendNilJSON(o)\n} else {", p.Varname())
	next(m, p.Value)
	m.fuseHook()
	m.p.closeblock()
}

func (m *marshalJSONGen) gBase(b *BaseElem) {
	if !m.p.ok() {
		return
	}
	m.fuseHook()
	vname := b.Varname()

	if b.Convert {
		if b.ShimMode == Cast {
			vname = tobaseConvert(b)
		} else {
			vname = randIdent()
			m.p.printf("\nvar %s %s", vname, b.BaseType())
			m.p.printf("\n%s, err = %s", vname, tobaseConvert(b))
			m.p.printf(errcheck)
		}
	}

	var echeck bool
	switch b.Value {
	case IDENT:
		echeck = true
		m.p.printf("\no, err = %s.MarshalBufferJSON(o)", vname)
	case Intf, Ext:
		echeck = true
		m.p.printf("\no, err = msgp.Append%sJSON(o, %s)", b.BaseName(), vname)
	default:
		m.rawAppend(b.BaseName(), literalFmt, vname)
	}

	if echeck {
		m.p.print(errcheck)
	}
}
