package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"strings"
)

type TypeKind int

const (
	VOID TypeKind = iota
	PTR
	I8
	I16
	I32
	I64
	U8
	U16
	U32
	U64
	F32
	F64
)

type Type struct {
	name    string
	kind    TypeKind
	ptrType *Type
}

type Function struct {
	name     string  // name of the Go func
	linkname string  // name to link to
	args     []*Type // the arguments to the func
	ret      *Type   // what if anything it returns
}

func main() {
	if len(os.Args) <= 1 {
		log.Fatal("no files specified")
		return
	}
	var files = os.Args[1:]
	fs := token.NewFileSet()
	var fileName = files[0]
	var fileNameNoExt = fileName[:strings.IndexRune(fileName, '.')]
	open, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	all, err := io.ReadAll(open)
	if err != nil {
		panic(err)
	}
	file, err := parser.ParseFile(fs, fileName, string(all), parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var package_ = file.Name.Name
	var functions []Function           // the functions to generate
	var libs = make(map[string]string) // the os -> shared object file
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if !strings.HasPrefix(c.Text, "//onlygo:open") {
				continue
			}
			args := strings.Split(c.Text, " ")
			if len(args) != 3 {
				log.Println("incorrect format for //onlygo:open GOOS LIB")
				continue
			}
			system := args[1]
			lib := args[2]
			libs[system] = lib
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				return false
			}
			var (
				name, linkname string
				args           []*Type
				ret            *Type
			)
			{
				typ := n.Type
				name = n.Name.Name
				linkname = name // linkname is guessed to be the same as the func name unless a go:linkname directive exists
				comments := n.Doc.List
				for _, c := range comments {
					if !strings.HasPrefix(c.Text, "//onlygo:linkname") {
						continue
					}
					linkname = strings.Split(c.Text, " ")[1]
				}
				for _, v := range typ.Params.List {
					for _, n := range v.Names {
						ty := getType(v.Type)
						ty.name = n.Name
						args = append(args, ty)
					}
				}
				if typ.Results != nil && len(typ.Results.List) == 1 {
					ret = getType(typ.Results.List[0].Type)
				} else {
					ret = &Type{}
				}
			}
			functions = append(functions, Function{
				name, linkname, args, ret,
			})
		}
		return true
	})
	var buf = &bytes.Buffer{}
	for sys, lib := range libs {
		buf.Reset()
		buf.WriteString("// File generated using onlygo. DO NOT EDIT!!!\n\n")
		buf.WriteString(fmt.Sprintf("package %s\n", package_))

		// import generation
		buf.WriteString(`
import (
	"github.com/totallygamerjet/pure-gl/internal/dyld"
)
`)

		//variable generation
		buf.WriteString("var (\n")
		for _, f := range functions {
			buf.WriteString(fmt.Sprintf("\t_%s uintptr\n", f.name))
		}
		buf.WriteString(")\n")

		// Init function generation
		buf.WriteString("func Init() error {\n")
		buf.WriteString(fmt.Sprintf("\tlib, err := dyld.Open(\"%s\", dyld.ScopeGlobal)\n", lib))
		buf.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
		for _, f := range functions {
			buf.WriteString(fmt.Sprintf("\t_%s, err = lib.Lookup(\"%s\")\n", f.name, f.linkname))
			buf.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
		}
		buf.WriteString("\treturn nil\n")
		buf.WriteString("}\n")
		init, err := os.Create(fileNameNoExt + "_" + sys + "_init.go")
		if err != nil {
			return
		}
		init.Write(buf.Bytes())
	}
	for sys := range libs {
		buf.Reset()
		buf.WriteString("// File generated using onlygo. DO NOT EDIT!!!\n")
		buf.WriteString("#include \"textflag.h\"\n\n")
		for _, f := range functions {
			gen := NewArm64FuncGen()
			buf.WriteString(fmt.Sprintf("TEXT ·%s(SB), NOSPLIT, $0-0\n", f.name)) //TODO: calc proper stacksize
			buf.WriteString("\tBL runtime·entersyscall(SB)\n")
			for i, arg := range f.args {
				buf.WriteString(fmt.Sprintf("%s", gen.MovInst(arg, i)))
			}
			buf.WriteString(fmt.Sprintf("\tMOVD ·_%s(SB), R16\n", f.name))
			buf.WriteString("\tCALL R16\n")
			if f.ret.kind != VOID {
				buf.WriteString(fmt.Sprintf("\t%s\n", gen.RetInst(f.ret)))
			}
			buf.WriteString("\tBL runtime·exitsyscall(SB)\n")
			buf.WriteString("\tRET\n\n")
		}
		create, err := os.Create(fileNameNoExt + "_" + sys + "_arm64.s") // TODO: other archs
		if err != nil {
			return
		}
		create.Write(buf.Bytes())
	}
}

func getType(expr ast.Expr) (ty *Type) {
	ty = &Type{}
	if star, ok := expr.(*ast.StarExpr); ok {
		ty.kind = PTR
		ty.ptrType = getType(star.X)
		return
	}
	ident := expr.(*ast.Ident)
	switch ident.Name {
	case "bool":
		ty.kind = U8
	case "int8":
		ty.kind = I8
	case "int16":
		ty.kind = I16
	case "int32":
		ty.kind = I32
	case "int64":
		ty.kind = I64
	case "uint8":
		ty.kind = U8
	case "uint16":
		ty.kind = U16
	case "uint32":
		ty.kind = U32
	case "uint64":
		ty.kind = U64
	case "float32":
		ty.kind = F32
	case "float64":
		ty.kind = F64
	default:
		panic(ident.Name)
	}
	return ty
}
