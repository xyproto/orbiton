package iferr

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

const noname = "(no name)"

var isNum = map[string]struct{}{
	"int":     struct{}{},
	"int16":   struct{}{},
	"int32":   struct{}{},
	"int64":   struct{}{},
	"uint":    struct{}{},
	"uint16":  struct{}{},
	"uint32":  struct{}{},
	"uint64":  struct{}{},
	"float":   struct{}{},
	"float32": struct{}{},
	"float64": struct{}{},
}

type Visitor struct {
	pos token.Pos
	err error
	ft  *ast.FuncType
	fn  string
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch x := node.(type) {
	case *ast.FuncDecl:
		x, ok := node.(*ast.FuncDecl)
		if !ok {
			return v
		}
		fname := x.Name.Name
		if v.pos < x.Pos() || v.pos > x.End() {
			return nil
		}
		if x.Type == nil {
			return v
		}
		v.fn = fname
		v.ft = x.Type
		return v
	case *ast.FuncLit:
		if x.Type == nil || x.Body == nil {
			return nil
		}
		if v.pos < x.Pos() || v.pos > x.End() {
			return nil
		}
		v.fn = noname
		v.ft = x.Type
		return v
	default:
		return v
	}
}

func ToTypes(fl *ast.FieldList) []ast.Expr {
	if fl == nil || len(fl.List) == 0 {
		return nil
	}
	types := make([]ast.Expr, 0, len(fl.List))
	for _, f := range fl.List {
		types = append(types, f.Type)
	}
	return types
}

func TypeString(x ast.Expr) string {
	switch t := x.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if _, ok := t.X.(*ast.Ident); ok {
			return TypeString(t.X) + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + TypeString(t.X)
	case *ast.ArrayType:
		return "[]" + TypeString(t.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.MapType:
		return "map[" + TypeString(t.Key) + "]" + TypeString(t.Value)
	case *ast.StructType:
		return "struct{}"
	case *ast.ChanType:
		return "chan " + TypeString(t.Value)
	default:
		return ""
	}
	return ""
}

func GetIfErr(types []ast.Expr) (string, error) {
	if len(types) == 0 {
		return "if err != nil {\n\treturn\n}\n", nil
	}
	var sb strings.Builder
	sb.WriteString("if err != nil {\n\treturn ")
	for i, t := range types {
		if i > 0 {
			sb.WriteString(", ")
		}
		ts := TypeString(t)
		if ts == "bool" {
			sb.WriteString("false")
			continue
		}
		if ts == "error" {
			sb.WriteString("err")
			continue
		}
		if ts == "string" {
			sb.WriteString(`""`)
			continue
		}
		if ts == "interface{}" {
			sb.WriteString("nil")
			continue
		}
		if ts == "time.Time" {
			sb.WriteString("time.Time{}")
			continue
		}
		if ts == "time.Duration" {
			sb.WriteString("time.Duration(0)")
			continue
		}
		if _, ok := isNum[ts]; ok {
			sb.WriteString("0")
			continue
		}
		if strings.HasPrefix(ts, "[]") {
			sb.WriteString("nil")
			continue
		}
		if strings.HasPrefix(ts, "map[") {
			sb.WriteString("nil")
			continue
		}
		if strings.HasPrefix(ts, "chan ") {
			sb.WriteString("nil")
			continue
		}
		if strings.HasPrefix(ts, "*") {
			sb.WriteString("nil")
			continue
		}
		// treat it as an interface when type name has "."
		if strings.Contains(ts, ".") {
			sb.WriteString("nil")
			continue
		}
		// TODO: support more types.
		sb.WriteString(ts)
		sb.WriteString("{}")
	}
	sb.WriteString("\n}\n")
	return sb.String(), nil
}

func IfErr(data []byte, lineIndex int) (string, error) {
	r := bytes.NewReader(data)
	file, err := parser.ParseFile(token.NewFileSet(), "iferr.go", r, 0)
	if err != nil {
		return "", err
	}
	v := &Visitor{pos: token.Pos(lineIndex)}
	ast.Walk(v, file)
	if v.err != nil {
		return "", err
	}
	if v.ft == nil {
		return "", fmt.Errorf("no functions at %d", lineIndex)
	}
	types := ToTypes(v.ft.Results)
	return GetIfErr(types)
}
