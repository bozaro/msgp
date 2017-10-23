package printer

import (
	"bytes"
	"fmt"
	"github.com/tinylib/msgp/gen"
	"github.com/tinylib/msgp/parse"
	"github.com/ttacon/chalk"
	"golang.org/x/tools/imports"
	"io/ioutil"
	"strings"
)

const (
	PackagePlaceholder = "${package}"
)

type FileSetResolver func([]string) (*parse.FileSet, error)

func infof(s string, v ...interface{}) {
	fmt.Printf(chalk.Magenta.Color(s), v...)
}

// PrintFile prints the methods for the provided list
// of elements to the given file name and canonical
// package path.
func PrintFile(file string, r FileSetResolver, mode gen.Method) error {
	out, tests, packageName, err := generate(r, mode)
	if err != nil {
		return err
	}

	// we'll run goimports on the main file
	// in another goroutine, and run it here
	// for the test file. empirically, this
	// takes about the same amount of time as
	// doing them in serial when GOMAXPROCS=1,
	// and faster otherwise.
	file = strings.Replace(file, PackagePlaceholder, packageName, -1)
	res := goformat(file, out.Bytes())
	if tests.Len() > 0 {
		testfile := strings.TrimSuffix(file, ".go") + "_test.go"
		err = format(testfile, tests.Bytes())
		if err != nil {
			return err
		}
		infof(">>> Wrote and formatted \"%s\"\n", testfile)
	}
	err = <-res
	if err != nil {
		return err
	}
	return nil
}

func format(file string, data []byte) error {
	out, err := imports.Process(file, data, nil)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, out, 0600)
}

func goformat(file string, data []byte) <-chan error {
	out := make(chan error, 1)
	go func(file string, data []byte, end chan error) {
		end <- format(file, data)
		infof(">>> Wrote and formatted \"%s\"\n", file)
	}(file, data, out)
	return out
}

func dedupImports(imp []string) []string {
	m := make(map[string]struct{})
	for i := range imp {
		m[imp[i]] = struct{}{}
	}
	r := []string{}
	for k := range m {
		r = append(r, k)
	}
	return r
}

func generate(r FileSetResolver, mode gen.Method) (*bytes.Buffer, *bytes.Buffer, string, error) {
	outbuf := bytes.NewBuffer(make([]byte, 0, 4096))
	testbuf := bytes.NewBuffer(make([]byte, 0, 4096))
	p := gen.NewPrinter(mode, outbuf, testbuf)

	cache := map[string]*parse.FileSet{}
	key := func(tags []string) string { return strings.Join(tags, ",") }
	packageName := ""
	// Parse metadata
	for _, g := range p.Gens {
		tags := g.Tags()
		k := key(tags)
		if _, ok := cache[k]; !ok {
			f, err := r(g.Tags())
			if err != nil {
				return nil, nil, "", err
			}
			cache[k] = f
			packageName = f.Package
		}
	}
	// Collect all imports
	importsGen := []string{}
	importsTest := []string{}
	for _, g := range p.Gens {
		f := cache[key(g.Tags())]
		if len(f.Identities) == 0 {
			continue
		}
		addImports := g.Imports()
		if g.IsTests() {
			importsTest = append(importsTest, "github.com/tinylib/msgp/msgp", "testing")
			importsTest = append(importsTest, addImports...)
		} else {
			importsGen = append(importsGen, "github.com/tinylib/msgp/msgp")
			importsGen = append(importsGen, addImports...)
		}
		for _, imp := range f.Imports {
			if imp.Name != nil {
				// have an alias, include it.
				importsGen = append(importsGen, imp.Name.Name+` `+imp.Path.Value)
			} else {
				importsGen = append(importsGen, imp.Path.Value)
			}
		}

	}
	// Write generator imports
	if len(importsGen) == 0 {
		return outbuf, testbuf, "", nil
	}
	writePkgHeader(outbuf, packageName)
	writeImportHeader(outbuf, dedupImports(importsGen)...)
	if len(importsTest) > 0 {
		writePkgHeader(testbuf, packageName)
		writeImportHeader(testbuf, dedupImports(importsTest)...)
	}
	// Generate code
	for _, g := range p.Gens {
		f := cache[key(g.Tags())]
		if len(f.Identities) == 0 {
			continue
		}
		err := f.PrintTo(g)
		if err != nil {
			return nil, nil, "", err
		}
	}
	return outbuf, testbuf, packageName, nil
}

func writePkgHeader(b *bytes.Buffer, name string) {
	b.WriteString("package ")
	b.WriteString(name)
	b.WriteByte('\n')
	b.WriteString("// NOTE: THIS FILE WAS PRODUCED BY THE\n// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)\n// DO NOT EDIT\n\n")
}

func writeImportHeader(b *bytes.Buffer, imports ...string) {
	b.WriteString("import (\n")
	for _, im := range imports {
		if im[len(im)-1] == '"' {
			// support aliased imports
			fmt.Fprintf(b, "\t%s\n", im)
		} else {
			fmt.Fprintf(b, "\t%q\n", im)
		}
	}
	b.WriteString(")\n\n")
}
