package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/types"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// I believe there are no types in Go with alignment 16.
var alignments = map[int64]string{
	1: "uint8",
	2: "uint16",
	4: "uint32",
	8: "uint64",
}

// FindUnion finds the struct that should be used as a template for
// the union.
func FindUnion(pkg *packages.Package, name string) *types.Struct {
	for _, d := range pkg.TypesInfo.Defs {
		if d != nil && d.Name() == name {
			s, ok := d.Type().Underlying().(*types.Struct)
			if ok {
				return s
			}
		}
	}

	return nil
}

// UnionSize returns the size and alignment necessary for the underyling union
// buffer given the template struct.
func UnionSize(s *types.Struct, lookup types.Sizes) (int64, int64) {
	var maxsz int64
	var maxalign int64

	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		sz := lookup.Sizeof(f.Type())
		align := lookup.Alignof(f.Type())
		if sz > maxsz {
			maxsz = sz
		}
		if align > maxalign {
			maxalign = align
		}
	}

	if maxsz%maxalign != 0 {
		maxsz = maxsz - maxsz%maxalign + maxalign
	}
	return maxsz, maxalign
}

// Field represents a field of the union.
type Field struct {
	name string
	typ  types.Type
}

// UnionFields returns the fields of the union given the template struct.
func UnionFields(s *types.Struct) []Field {
	fields := make([]Field, s.NumFields())
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		fields[i] = Field{
			name: f.Name(),
			typ:  f.Type(),
		}
	}
	return fields
}

// GetImports returns the names of any packages that are needed to access
// the types in the union fields.
func GetImports(fields []Field, pkg *types.Package) []string {
	imports := make([]string, 0)
	for _, f := range fields {
		if t, ok := f.typ.(*types.Named); ok {
			if pkg != t.Obj().Pkg() {
				imports = append(imports, "\""+t.Obj().Pkg().Path()+"\"")
			}
		}
	}
	return imports
}

func qual(pkg *types.Package) types.Qualifier {
	if pkg == nil {
		return nil
	}
	return func(other *types.Package) string {
		if pkg == other {
			return ""
		}
		return other.Name()
	}
}

// StringUnion builds the source code for the union.
func StringUnion(name string, size, align int64, fields []Field, pkg *types.Package) string {
	// This is a little bit of a hack in order to make the union type
	// properly aligned. The union must be aligned to the largest member, and
	// using a byte array will have alignment 1, so we use an array of type
	// T, where T has the correct alignment.  The `alignments` map contains
	// primitive types with alignments up to 8. We also have to adjust the
	// size, since the primitive type of the buffer will be modified.
	typ := alignments[align]
	size /= align
	s := fmt.Sprintf(structTemplate, name, size, typ)

	for _, f := range fields {
		s += fmt.Sprintf(fieldTemplate, name, f.name, types.TypeString(f.typ, qual(pkg)))
	}

	return s
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage of unionize:")
		fmt.Println("\tunionize [flags] T [directory]")
		fmt.Println("\tunionize [flags] T files...")
		fmt.Println("For more information, see:")
		fmt.Println("\thttps://github.com/zyedidia/unionize")
		fmt.Println("Flags:")

		flag.PrintDefaults()
	}

	flagPkg := flag.String("pkg", "main", "output package name")
	flagUnion := flag.String("otype", "", "output union type name")
	flagFile := flag.String("output", "", "output file name")

	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Invalid number of arguments: need union and package.\n")
		flag.Usage()
		os.Exit(1)
	}

	cfg := &packages.Config{Mode: packages.LoadTypes | packages.LoadSyntax | packages.LoadImports}
	pkgs, err := packages.Load(cfg, args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	if len(pkgs) <= 0 {
		fmt.Fprintf(os.Stderr, "Error: no package found\n")
		os.Exit(1)
	}

	pkg := pkgs[0]
	strct := FindUnion(pkg, args[0])
	if strct == nil {
		fmt.Fprintf(os.Stderr, "Error: could not find struct to unionize\n")
		os.Exit(1)
	}

	var unionName string
	if flagUnion != nil && *flagUnion != "" {
		unionName = *flagUnion
	} else {
		unionName = args[0] + "Union"
	}

	sz, align := UnionSize(strct, pkg.TypesSizes)

	if _, ok := alignments[align]; !ok {
		fmt.Printf("Warning: alignment of %d cannot be satisfied with a primitive type, using alignment of %d instead\n", align, 8)
		align = 8
	}

	fields := UnionFields(strct)
	imports := GetImports(fields, pkg.Types)

	buf := &bytes.Buffer{}
	buf.WriteString(header)
	buf.WriteString(fmt.Sprintf(packageTemplate, *flagPkg))
	buf.WriteString(fmt.Sprintf(importTemplate, strings.Join(imports, "\n")))
	buf.WriteString(StringUnion(unionName, sz, align, fields, pkg.Types))

	output, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Print(buf.String())
		fmt.Fprintf(os.Stderr, "Error with union generation:\n")
		fmt.Fprintf(os.Stderr, "format: %v\n", err)
		os.Exit(1)
	}

	if flagFile != nil && *flagFile != "" {
		ioutil.WriteFile(*flagFile, output, 0666)
	} else {
		fmt.Print(string(output))
	}
}
