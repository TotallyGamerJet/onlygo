package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
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
	INT
	U8
	U16
	U32
	U64
	UINT
	F32
	F64
	STRUCT // struct
	ARRAY
)

type Type struct {
	name           string
	kind           TypeKind
	underlyingType *Type   // Underlying type if pointer or array
	fields         []*Type // used only if kind == STRUCT
	length         int     // only used if kind == ARRAY
	padding        int     // any padding this type receives
}

type Function struct {
	name     string  // name of the Go func
	linkname string  // name to link to
	sig      string  // the signature as written in go
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
	var functions []Function                      // the functions to generate
	var libs = make(map[string]map[string]string) // the os -> arch -> shared object file
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if !strings.HasPrefix(c.Text, "//onlygo:open") {
				continue
			}
			args := strings.Split(c.Text, " ")
			if len(args) != 4 {
				log.Println("incorrect format for //onlygo:open GOOS GOARCH LIB")
				continue
			}
			system := args[1]
			arch := args[2]
			lib := args[3]
			archs := libs[system]
			if archs == nil {
				archs = make(map[string]string)
				libs[system] = archs
			}
			archs[arch] = lib
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				return false
			}
			var (
				name, linkname, sig string
				args                []*Type
				ret                 *Type
			)
			{
				typ := n.Type
				name = n.Name.Name
				linkname = name // linkname is guessed to be the same as the func name unless a go:linkname directive exists
				var comments []*ast.Comment
				if n.Doc != nil {
					comments = n.Doc.List
				}
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
				name, linkname, sig, args, ret,
			})
		}
		return true
	})
	var buf = &bytes.Buffer{}
	for sys, archs := range libs {
		for arch, lib := range archs {
			create, err := os.Create(fileNameNoExt + "_so_" + sys + "_" + arch + ".go")
			if err != nil {
				panic(err)
			}
			_, _ = create.WriteString(fmt.Sprintf("package %s\n\nconst _%s_SharedObject = \"%s\"\n", package_, fileNameNoExt, lib))
			_ = create.Close()
		}
	}
	{ // Init function
		buf.Reset()
		buf.WriteString("// File generated using onlygo. DO NOT EDIT!!!\n\n")
		buf.WriteString(fmt.Sprintf("package %s\n", package_))

		// import generation
		buf.WriteString(`
import (
	"github.com/totallygamerjet/dl"
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
		buf.WriteString(fmt.Sprintf("\tlib, err := dl.Open(_%s_SharedObject, dl.ScopeGlobal)\n", fileNameNoExt))
		buf.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
		for _, f := range functions {
			buf.WriteString(fmt.Sprintf("\t_%s, err = lib.Lookup(\"%s\")\n", f.name, f.linkname))
			buf.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
		}
		buf.WriteString("\treturn nil\n")
		buf.WriteString("}\n")
		init, err := os.Create(fileNameNoExt + "_init.go")
		if err != nil {
			panic(err)
			return
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			panic(err)
		}
		_, _ = init.Write(formatted)
		err = init.Close()
		if err != nil {
			return
		}
	}
	for sys, archs := range libs {
		for arch := range archs {
			if genFn, ok := generators[sys][arch]; ok {
				buf.Reset()
				buf.WriteString("// File generated using onlygo. DO NOT EDIT!!!\n")
				buf.WriteString("#include \"textflag.h\"\n\n")
				for _, f := range functions {
					gen := genFn(buf, f)
					buf.WriteString(fmt.Sprintf("//%s\n", f.sig))
					buf.WriteString(fmt.Sprintf("TEXT ·%s(SB), NOSPLIT, $0-0\n", f.name)) //TODO: calc proper stacksize
					gen.PreCall()
					for _, arg := range f.args {
						gen.MovInst(arg)
					}
					gen.GenCall(f.name)
					if f.ret.kind != VOID {
						gen.RetInst(f.ret)
					}
					gen.PostCall()
					buf.WriteString("\tRET\n\n")
				}
				create, err := os.Create(fileNameNoExt + "_" + sys + "_" + arch + ".s") // TODO: other archs
				if err != nil {
					panic(err)
					return
				}
				_, _ = create.Write(buf.Bytes())
			} else {
				log.Println(fmt.Sprintf("the GOOS and GOARCH combo (%s, %s) is not supported.", sys, arch))
			}
		}
	}
}

func getType(expr ast.Expr) (ty *Type) {
	ty = &Type{}
	if sel, ok := expr.(*ast.ArrayType); ok {
		panic(sel)
	}
	if sel, ok := expr.(*ast.StructType); ok {
		ty.kind = STRUCT
		ty.fields = make([]*Type, sel.Fields.NumFields())
		for i, f := range sel.Fields.List {
			ty.fields[i] = getType(f.Type)
		}
		return
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if sel.Sel.String() != "Pointer" {
			panic(fmt.Sprintf("unknown selector: %s", sel.Sel.String()))
		}
		ty.kind = PTR
		return
	}
	if star, ok := expr.(*ast.StarExpr); ok {
		ty.kind = PTR
		ty.underlyingType = getType(star.X)
		return
	}
	ident := expr.(*ast.Ident)
	switch ident.Name {
	case "bool":
		ty.kind = U8
	case "uintptr":
		ty.kind = PTR
	case "int8":
		ty.kind = I8
	case "int16":
		ty.kind = I16
	case "int32":
		ty.kind = I32
	case "int64":
		ty.kind = I64
	case "int":
		ty.kind = INT
	case "uint8":
		ty.kind = U8
	case "uint16":
		ty.kind = U16
	case "uint32":
		ty.kind = U32
	case "uint64":
		ty.kind = U64
	case "uint":
		ty.kind = UINT
	case "float32":
		ty.kind = F32
	case "float64":
		ty.kind = F64
	default:
		panic(ident.Name)
	}
	return ty
}
