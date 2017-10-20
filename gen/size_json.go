package gen

import (
	"fmt"
	"io"
	"strconv"

	"github.com/tinylib/msgp/msgp"
)

func sizesJSON(w io.Writer) *sizeJSONGen {
	return &sizeJSONGen{
		p:     printer{w: w},
		state: assign,
	}
}

type sizeJSONGen struct {
	passes
	p     printer
	state sizeState
}

func (s *sizeJSONGen) Method() Method { return Size }

func (s *sizeJSONGen) Apply(dirs []string) error {
	return nil
}

func builtinSizeJSON(typ string) string {
	return "msgp." + typ + "SizeJSON"
}

// this lets us chain together addition
// operations where possible
func (s *sizeJSONGen) addConstant(sz string) {
	if !s.p.ok() {
		return
	}

	switch s.state {
	case assign:
		s.p.print("\ns = " + sz)
		s.state = expr
		return
	case add:
		s.p.print("\ns += " + sz)
		s.state = expr
		return
	case expr:
		s.p.print(" + " + sz)
		return
	}

	panic("unknown size state")
}

func (s *sizeJSONGen) Execute(p Elem) error {
	if !s.p.ok() {
		return s.p.err
	}
	p = s.applyall(p)
	if p == nil {
		return nil
	}
	if !IsPrintable(p) {
		return nil
	}

	s.p.comment("SizeJSON returns an upper bound estimate of the number of bytes occupied by the serialized message")

	s.p.printf("\nfunc (%s %s) SizeJSON() (s int) {", p.Varname(), imutMethodReceiver(p))
	s.state = assign
	next(s, p)
	s.p.nakedReturn()
	return s.p.err
}

func (s *sizeJSONGen) gStruct(st *Struct) {
	if !s.p.ok() {
		return
	}

	fieldSize := 2 + len(st.Fields) // brackets and separators
	if len(st.Fields) > 0 {
		fieldSize -= 1
	}
	if !st.AsTuple {
		for _, field := range st.Fields {
			fieldSize += len(s.escape(field.FieldTag)) + 3 // name, quotes and colon
		}
	}
	s.addConstant(strconv.Itoa(fieldSize))
	for _, field := range st.Fields {
		if !s.p.ok() {
			return
		}
		next(s, field.FieldElem)
	}
}

func (s *sizeJSONGen) gPtr(p *Ptr) {
	s.state = add // inner must use add
	s.p.printf("\nif %s == nil {\ns += msgp.NilSizeJSON\n} else {", p.Varname())
	next(s, p.Value)
	s.state = add // closing block; reset to add
	s.p.closeblock()
}

func (s *sizeJSONGen) gSlice(sl *Slice) {
	if !s.p.ok() {
		return
	}

	s.addConstant(fmt.Sprintf("2 + %s", lenExpr(sl))) // brackets and separators

	// if the slice's element is a fixed size
	// (e.g. float64, [32]int, etc.), then
	// print the length times the element size directly
	if str, ok := s.fixedsizeExpr(sl.Els); ok {
		s.addConstant(fmt.Sprintf("(%s * (%s))", lenExpr(sl), str))
		return
	}

	// add inside the range block, and immediately after
	s.state = add
	s.p.rangeBlock(sl.Index, sl.Varname(), s, sl.Els)
	s.state = add
}

func (s *sizeJSONGen) gArray(a *Array) {
	if !s.p.ok() {
		return
	}

	s.addConstant(fmt.Sprintf("2 + int(%s)", a.Size)) // brackets and separators

	// if the array's children are a fixed
	// size, we can compile an expression
	// that always represents the array's wire size
	if str, ok := s.fixedsizeExpr(a); ok {
		s.addConstant(str)
		return
	}

	s.state = add
	s.p.rangeBlock(a.Index, a.Varname(), s, a.Els)
	s.state = add
}

func (s *sizeJSONGen) gMap(m *Map) {
	vn := m.Varname()
	s.addConstant("2") // brackets
	s.p.printf("\nif %s != nil {", vn)
	s.p.printf("\n for %s, %s := range %s {", m.Keyidx, m.Validx, vn)
	s.p.printf("\n  _ = %s", m.Validx) // we may not use the value
	s.p.printf("\n  s += 4 + len(%s)", m.Keyidx)
	s.state = expr
	next(s, m.Value)
	s.p.print("\n }")
	s.p.print("\n} else {")
	s.p.print("\n s += msgp.NilSizeJSON")
	s.p.print("\n}")
	s.state = add
}

func (s *sizeJSONGen) gBase(b *BaseElem) {
	if !s.p.ok() {
		return
	}
	if b.Convert && b.ShimMode == Convert {
		s.state = add
		vname := randIdent()
		s.p.printf("\nvar %s %s", vname, b.BaseType())

		// ensure we don't get "unused variable" warnings from outer slice iterations
		s.p.printf("\n_ = %s", b.Varname())

		s.p.printf("\ns += %s", basesizeExprJSON(b.Value, vname, b.BaseName()))
		s.state = expr

	} else {
		vname := b.Varname()
		if b.Convert {
			vname = tobaseConvert(b)
		}
		s.addConstant(basesizeExprJSON(b.Value, vname, b.BaseName()))
	}
}

func (s *sizeJSONGen) escape(v string) string {
	return string(msgp.EscapeJSON(nil, v))
}

// return a fixed-size expression, if possible.
// only possible for *BaseElem and *Array.
// returns (expr, ok)
func (s *sizeJSONGen) fixedsizeExpr(e Elem) (string, bool) {
	switch e := e.(type) {
	case *Array:
		if str, ok := s.fixedsizeExpr(e.Els); ok {
			return fmt.Sprintf("2 + ((%s + 1) * (%s))", e.Size, str), true
		}
	case *BaseElem:
		if fixedSize(e.Value) {
			return builtinSizeJSON(e.BaseName()), true
		}
	case *Struct:
		var str string
		for _, f := range e.Fields {
			if fs, ok := s.fixedsizeExpr(f.FieldElem); ok {
				if str == "" {
					str = fs
				} else {
					str += "+" + fs
				}
			} else {
				return "", false
			}
		}
		var hdrlen int
		hdrlen += len(e.Fields)*4 + 1 // brackets, name, quotes, separators, colons
		for _, f := range e.Fields {
			hdrlen += len(s.escape(f.FieldTag))
		}
		return fmt.Sprintf("%d + %s", hdrlen, str), true
	}
	return "", false
}

// print size expression of a variable name
func basesizeExprJSON(value Primitive, vname, basename string) string {
	switch value {
	case Ext, Intf:
		return "msgp.GuessSizeJSON(" + vname + ")"
	case IDENT:
		return vname + ".SizeJSON()"
	case Bytes:
		return "msgp.BytesSizeJSON(" + vname + ")"
	case String:
		return "msgp.StringSizeJSON(" + vname + ")"
	default:
		return builtinSizeJSON(basename)
	}
}
