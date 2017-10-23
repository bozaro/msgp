package gen

import (
	"io"
)

func unmarshalJSON(w io.Writer) *unmarshalJSONGen {
	return &unmarshalJSONGen{
		p: printer{w: w},
	}
}

type unmarshalJSONGen struct {
	passes
	p        printer
	hasfield bool
}

func (e *unmarshalJSONGen) Tags() []string {
	return jsonTags
}

func (e *unmarshalJSONGen) IsTests() bool {
	return false
}

func (e *unmarshalJSONGen) Imports() []string {
	return []string{
		"github.com/mailru/easyjson/jlexer",
	}
}

func (u *unmarshalJSONGen) Method() Method { return UnmarshalJSON }

func (u *unmarshalJSONGen) needsField() {
	if u.hasfield {
		return
	}
	u.p.print("\nvar field string; _ = field")
	u.hasfield = true
}

func (u *unmarshalJSONGen) Execute(p Elem) error {
	u.hasfield = false
	if !u.p.ok() {
		return u.p.err
	}
	p = u.applyall(p)
	if p == nil {
		return nil
	}
	if !IsPrintable(p) {
		return nil
	}

	u.p.comment("UnmarshalJSON implements json.Unmarshaler")
	selfvar := p.Varname()
	receiver := methodReceiver(p)
	u.p.printf("\nfunc (%s %s) UnmarshalJSON(bts []byte) (err error) {", selfvar, receiver)
	u.p.print("\nl := jlexer.Lexer{Data: bts}")
	u.p.printf("\n%s.UnmarshalEasyJSON(&l)", p.Varname())
	u.p.print("\nreturn l.Error()")
	u.p.closeblock()

	u.p.printf("\nfunc (%s %s) UnmarshalEasyJSON(l *jlexer.Lexer) {", selfvar, receiver)
	/*u.p.print("\n isTopLevel := l.IsStart()")
	u.p.print("\n if l.IsNull() {")
	u.p.print("\n if isTopLevel {")
	u.p.print("\n  l.Consumed()")
	u.p.print("\n }")
	u.p.print("\n l.Skip()")
	u.p.print("\n  return")
	u.p.print("\n }")
	u.p.print("\n l.Delim('{')")*/
	next(u, p)
	/*u.p.print("\n l.Delim('}')")
	u.p.print("\n if isTopLevel {")
	u.p.print("\n  l.Consumed()")
	u.p.print("\n }")*/
	u.p.closeblock()
	unsetReceiver(p)
	return u.p.err
}

// does assignment to the variable "name" with the type "base"
func (u *unmarshalJSONGen) assignAndCheck(name string, base string) {
	if !u.p.ok() {
		return
	}
	u.p.printf("\n%s, bts, err = msgp.Read%sBytes(bts)", name, base)
	u.p.print(errcheck)
}

func (u *unmarshalJSONGen) gStruct(s *Struct) {
	if !u.p.ok() {
		return
	}
	if s.AsTuple {
		u.tuple(s)
	} else {
		u.mapstruct(s)
	}
	return
}

func (u *unmarshalJSONGen) tuple(s *Struct) {
	u.p.comment("TODO: tuple")
	/*
		// open block
		sz := randIdent()
		u.p.declare(sz, u32)
		u.assignAndCheck(sz, arrayHeader)
		u.p.arrayCheck(strconv.Itoa(len(s.Fields)), sz)
		for i := range s.Fields {
			if !u.p.ok() {
				return
			}
			next(u, s.Fields[i].FieldElem)
		}
	*/
}

func (u *unmarshalJSONGen) mapstruct(s *Struct) {
	u.needsField()
	u.p.print("\n l.Delim('{')")
	u.p.print("\n for !l.IsDelim('}') {")
	// Read field name
	u.p.print("\n  field := l.UnsafeString()")
	u.p.print("\n  l.WantColon()")
	u.p.print("\n  if l.IsNull() {")
	u.p.print("\n 	l.Skip()")
	u.p.print("\n 	l.WantComma()")
	u.p.print("\n 	continue")
	u.p.print("\n  }")
	u.p.print("\n  switch field {")
	for i := range s.Fields {
		if !u.p.ok() {
			return
		}
		u.p.printf("\ncase \"%s\":", s.Fields[i].FieldTag)
		next(u, s.Fields[i].FieldElem)
	}
	u.p.print("\n   default:")
	u.p.print("\n    l.SkipRecursive()")
	u.p.print("\n  }")
	u.p.print("\n  l.WantComma()")
	// Close loop
	u.p.print("\n }")
	u.p.print("\n l.Delim('}')")
}

func (u *unmarshalJSONGen) gBase(b *BaseElem) {
	if !u.p.ok() {
		return
	}

	refname := b.Varname() // assigned to
	lowered := b.Varname() // passed as argument
	if b.Convert {
		// begin 'tmp' block
		refname = randIdent()
		lowered = b.ToBase() + "(" + lowered + ")"
		u.p.printf("\n{\nvar %s %s", refname, b.BaseType())
	}

	switch b.Value {
	case Bytes:
		u.p.printf("\n%s = l.Bytes()", refname)
	case Intf, Ext:
		u.p.printf("\nmsgp.Read%sJSON(l, %s)", b.BaseName(), refname)
	case IDENT:
		u.p.printf("\n%s.UnmarshalEasyJSON(l)", lowered)
	default:
		u.p.printf("\n%s = msgp.Read%sJSON(l) // %v", refname, b.BaseName(), b.Value)
	}

	if b.Convert {
		// close 'tmp' block
		if b.ShimMode == Cast {
			u.p.printf("\n%s = %s(%s)\n", b.Varname(), b.FromBase(), refname)
		} else {
			u.p.printf("\n%s, err = %s(%s)", b.Varname(), b.FromBase(), refname)
			u.p.print(errcheck)
		}
		u.p.printf("}")
	}
}

func (u *unmarshalJSONGen) gArray(a *Array) {
	if !u.p.ok() {
		return
	}

	// special case for [const]byte objects
	// see decode.go for symmetry
	if be, ok := a.Els.(*BaseElem); ok && be.Value == Byte {
		u.p.printf("\nmsgp.ReadExactBytesJSON(l, (%s)[:])", a.Varname())
		return
	}

	u.p.print("\n l.Delim('[')")
	// Read elements
	u.p.printf("\n for %s := 0; %s < int(%s); %s++ {", a.Index, a.Index, a.Size, a.Index)
	next(u, a.Els)
	u.p.print("\n  l.WantComma()")
	u.p.print("\n }")
	u.p.print("\n l.Delim(']')")
}

func (u *unmarshalJSONGen) gSlice(s *Slice) {
	if !u.p.ok() {
		return
	}
	u.p.print("\n l.Delim('[')")
	u.p.print("\n if !l.IsDelim(']') {")
	u.p.printf("\n  %s = make(%s, 0, 4)", s.Varname(), s.TypeName())
	u.p.print("\n } else {")
	u.p.printf("\n  %s = %s {}", s.Varname(), s.TypeName())
	u.p.print("\n }")
	// Read elements
	u.p.print("\n for !l.IsDelim(']') {")
	elem := s.Els.Copy()
	elem.SetVarname(randIdent())
	u.p.printf("\n  var %s %s", elem.Varname(), elem.TypeName())
	next(u, elem)
	u.p.printf("\n  %s = append(%s, %s)", s.Varname(), s.Varname(), elem.Varname())
	u.p.print("\n  l.WantComma()")
	u.p.print("\n }")
	u.p.print("\n l.Delim(']')")
}

func (u *unmarshalJSONGen) gMap(m *Map) {
	if !u.p.ok() {
		return
	}
	u.p.print("\n l.Delim('{')")
	u.p.print("\n if !l.IsDelim('}') {")
	u.p.printf("\n  %s = make(%s)", m.Varname(), m.TypeName())
	u.p.print("\n } else {")
	u.p.printf("\n  %s = nil", m.Varname())
	u.p.print("\n }")
	u.p.print("\n for !l.IsDelim('}') {")
	// Read field name
	u.p.printf("\n  %s := l.String()", m.Keyidx)
	u.p.print("\n  l.WantColon()")
	u.p.printf("\n  var %s %s", m.Validx, m.Value.TypeName())
	next(u, m.Value)
	u.p.mapAssign(m)
	u.p.print("\n }")
	u.p.print("\n l.Delim('}')")
}
func (u *unmarshalJSONGen) gPtr(p *Ptr) {
	u.p.printf("\nif l.IsNull() { l.Skip(); %s = nil; } else { ", p.Varname())
	u.p.initPtr(p)
	next(u, p.Value)
	u.p.closeblock()
}
