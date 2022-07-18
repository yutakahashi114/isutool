package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"strings"
)

var (
	fset = token.NewFileSet()

	targetStruct    = flag.String("s", "", "target struct")
	targetFile      = flag.String("f", "", "target file")
	outputFile      = flag.String("o", "", "output file")
	pointerReceiver = flag.Bool("p", true, "receiver is pointer")

	packageMap     = map[string]string{}
	usedPackageMap = map[string]string{}
)

func init() {
	flag.Parse()
}

func main() {
	targets := strings.Split(*targetStruct, ",")
	targetMap := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		targetMap[t] = struct{}{}
	}
	receiverIsPointer := *pointerReceiver
	outFile := *outputFile
	if outFile == "" {
		outFile = strings.TrimSuffix(*targetFile, ".go") + "_gen.go"
	}

	out, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}

	af, err := parser.ParseFile(fset, *targetFile, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	for _, is := range af.Imports {
		var name string
		if is.Name != nil {
			name = is.Name.Name
		} else {
			ss := strings.Split(strings.TrimSuffix(is.Path.Value, "\""), "/")
			name = ss[len(ss)-1]
		}
		packageMap[name] = is.Path.Value
	}

	param := param{
		PackageName:       af.Name.Name,
		ReceiverIsPointer: receiverIsPointer,
	}

	for _, d := range af.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		if gd.Tok != token.TYPE {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := targetMap[ts.Name.Name]; !ok {
				continue
			}
			f := parseTypeSpec(ts)
			ef := f
			if receiverIsPointer {
				ef = field{
					fieldType: pointerType{
						value: ef.fieldType,
					},
				}
			}

			param.StructData = append(param.StructData, structParam{
				StructName:   ts.Name.Name,
				ReceiverName: strings.ToLower(ts.Name.Name)[:1],
				EncodeField:  ef,
				DecodeField:  f,
			})
		}
	}

	delete(usedPackageMap, "time")
	param.ImportPackageMap = usedPackageMap

	var content bytes.Buffer
	err = Template.Execute(&content, &param)
	if err != nil {
		log.Fatal(err)
	}
	// out.Write(content.Bytes())
	// fmt.Println(string(content.Bytes()))
	re, err := format.Source(content.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(string(re))
	out.Write(re)
}

type field struct {
	name      string
	fieldType fieldType
}

func (f field) MaxSize(receiverName string) template.HTML {
	return template.HTML(fieldTypeToSize(f.fieldType, receiverName))
}

func (f field) comment() string {
	return fmt.Sprintf("// %s\n", f.name)
}

func fieldTypeToSize(ft fieldType, varName string) string {
	switch t := ft.(type) {
	case aliasType:
		return fieldTypeToSize(t.value, varName)
	case primitiveType:
		switch t.name {
		case "string":
			return fmt.Sprintf(`_size += binary.MaxVarintLen64 + len(%s)`, varName)
		case "int8", "uint8", "byte", "bool":
			return "_size += 1"
		case "int16", "uint16":
			return "_size += binary.MaxVarintLen16"
		case "int32", "uint32", "float32", "rune":
			return "_size += binary.MaxVarintLen32"
		case "int", "int64", "uint", "uint64", "float64":
			return "_size += binary.MaxVarintLen64"
		}
	case structType:
		buf := &strings.Builder{}
		for _, f := range t.fields {
			buf.WriteString(f.comment())
			buf.WriteString(fieldTypeToSize(f.fieldType, fmt.Sprintf("%s.%s", varName, f.name)))
			buf.WriteString("\n")
		}
		return buf.String()
	case mapType:
		return fmt.Sprintf(`_size += binary.MaxVarintLen64
		_size += 1  // is nil
		for _k, _v := range %s {
			_, _ = _k, _v
			%s
			%s
		}`, varName, fieldTypeToSize(t.key, "_k"), fieldTypeToSize(t.value, "_v"))
	case arrayType:
		return fmt.Sprintf(`for _, _e := range %s {
			_ = _e
			%s
		}`, varName, fieldTypeToSize(t.elem, "_e"))
	case sliceType:
		return fmt.Sprintf(`_size += binary.MaxVarintLen64
		_size += 1  // is nil
		for _, _e := range %s {
			_ = _e
			%s
		}`, varName, fieldTypeToSize(t.elem, "_e"))
	case pointerType:
		return fmt.Sprintf(`_size += 1  // is nil
		if %s != nil {
			%s}`, varName, fieldTypeToSize(t.value, "(*"+varName+")"))
	case externalStructType:
		if t.packageName == "time" && t.name == "Time" {
			return "_size += 16"
		}
		return fmt.Sprintf("_size += %s.MaxSize()", varName)
	}
	return ""
}

func (f field) Encode(receiverName string) template.HTML {
	return template.HTML(fieldTypeToEncode(f.fieldType, receiverName))
}

func fieldTypeToEncode(ft fieldType, varName string) string {
	switch t := ft.(type) {
	case aliasType:
		return fieldTypeToEncode(t.value, varName)
	case primitiveType:
		switch t.name {
		case "string":
			return fmt.Sprintf(`_n += binary.PutVarint(out[_n:], int64(len(%s)))
			_n += copy(out[_n:], %s)`, varName, varName)
		case "bool":
			return fmt.Sprintf(`if %s {
				out[_n] = 1
			} else {
				out[_n] = 0
			}
			_n += 1`, varName)
		case "int8", "uint8", "byte":
			return fmt.Sprintf(`out[_n] = byte(%s)
			_n += 1`, varName)
		case "int16":
			return fmt.Sprintf("_n += binary.PutVarint(out[_n:], int64(%s))", varName)
		case "uint16":
			return fmt.Sprintf("_n += binary.PutUvarint(out[_n:], uint64(%s))", varName)
		case "int32", "rune":
			return fmt.Sprintf("_n += binary.PutVarint(out[_n:], int64(%s))", varName)
		case "uint32":
			return fmt.Sprintf("_n += binary.PutUvarint(out[_n:], uint64(%s))", varName)
		case "float32":
			return fmt.Sprintf("_n += binary.PutUvarint(out[_n:], uint64(math.Float32bits(%s)))", varName)
		case "int", "int64":
			return fmt.Sprintf("_n += binary.PutVarint(out[_n:], int64(%s))", varName)
		case "uint", "uint64":
			return fmt.Sprintf("_n += binary.PutUvarint(out[_n:], uint64(%s))", varName)
		case "float64":
			return fmt.Sprintf("_n += binary.PutUvarint(out[_n:], uint64(math.Float64bits(%s)))", varName)
		}
	case structType:
		buf := &strings.Builder{}
		for _, f := range t.fields {
			buf.WriteString(f.comment())
			buf.WriteString(fieldTypeToEncode(f.fieldType, fmt.Sprintf("%s.%s", varName, f.name)))
			buf.WriteString("\n")
		}
		return buf.String()
	case mapType:
		return fmt.Sprintf(`if %s == nil {
			out[_n] = 0
			_n += 1
		} else {
			out[_n] = 1
			_n += 1
			_n += binary.PutVarint(out[_n:], int64(len(%s)))
			for _k, _v := range %s {
				_, _ = _k, _v
				%s
				%s
			}
		}`, varName, varName, varName, fieldTypeToEncode(t.key, "_k"), fieldTypeToEncode(t.value, "_v"))
	case arrayType:
		return fmt.Sprintf(`for _, _e := range %s {
			%s
		}`, varName, fieldTypeToEncode(t.elem, "_e"))
	case sliceType:
		return fmt.Sprintf(`if %s == nil {
			out[_n] = 0
			_n += 1
		} else {
			out[_n] = 1
			_n += 1
			_n += binary.PutVarint(out[_n:], int64(len(%s)))
			for _, _e := range %s {
				_ = _e
				%s
			}
		}`, varName, varName, varName, fieldTypeToEncode(t.elem, "_e"))
	case pointerType:
		return fmt.Sprintf(`if %s == nil {
			out[_n] = 0
			_n += 1
		} else {
			out[_n] = 1
			_n += 1
			%s
		}`, varName, fieldTypeToEncode(t.value, "(*"+varName+")"))
	case externalStructType:
		replacedVarName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(varName, ".", "_"), "*", "ç"), "(", "ƒ"), ")", "å")
		if t.packageName == "time" && t.name == "Time" {
			return fmt.Sprintf(`%s, err := _timeMarshalBinary(%s, out[_n:])
			if err != nil {
				return 0, err
			}
			_n += %s`, replacedVarName, varName, replacedVarName)
		}
		return fmt.Sprintf(`%s, err := %s.EncodeTo(out[_n:])
		if err != nil {
			return 0, err
		}
		_n += %s`, replacedVarName, varName, replacedVarName)
	}
	return ""
}

func (f field) Decode(receiverName string, typeName string) template.HTML {
	return template.HTML(fieldTypeToDecode(f.fieldType, fmt.Sprintf("(*%s)", receiverName), typeName))
}

func fieldTypeToDecode(ft fieldType, varName string, typeName string) string {
	replacedVarName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(varName, ".", "_"), "*", "ç"), "(", "ƒ"), ")", "å")
	switch t := ft.(type) {
	case aliasType:
		return fieldTypeToDecode(t.value, varName, t.aliasName)
	case primitiveType:
		switch t.name {
		case "string":
			return fmt.Sprintf(`_%sLen, _%sLenSize := binary.Varint(in[_n:])
				_n += _%sLenSize
				%s = %s(in[_n : _n+int(_%sLen)])
				_n += int(_%sLen)`, replacedVarName, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "bool":
			return fmt.Sprintf(`if in[_n] == 1 {
					%s = true
				} else {
					%s = false
				}
				_n += 1`, varName, varName)
		case "int8":
			return fmt.Sprintf(`%s = %s(in[_n])
				_n += 1`, varName, typeName)
		case "uint8":
			return fmt.Sprintf(`%s = %s(in[_n])
				_n += 1`, varName, typeName)
		case "byte":
			return fmt.Sprintf(`%s = %s(in[_n])
				_n += 1`, varName, typeName)
		case "int16":
			return fmt.Sprintf(`_%s, _%sSize := binary.Varint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "uint16":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "int32":
			return fmt.Sprintf(`_%s, _%sSize := binary.Varint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "rune":
			return fmt.Sprintf(`_%s, _%sSize := binary.Varint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "uint32":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "float32":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(math.Float32frombits(uint32(_%s)))
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "int":
			return fmt.Sprintf(`_%s, _%sSize := binary.Varint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "int64":
			return fmt.Sprintf(`_%s, _%sSize := binary.Varint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "uint":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "uint64":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(_%s)
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		case "float64":
			return fmt.Sprintf(`_%s, _%sSize := binary.Uvarint(in[_n:])
				%s = %s(math.Float64frombits(_%s))
				_n += _%sSize`, replacedVarName, replacedVarName, varName, typeName, replacedVarName, replacedVarName)
		}
	case structType:
		buf := &strings.Builder{}
		for _, f := range t.fields {
			buf.WriteString(f.comment())
			buf.WriteString(fieldTypeToDecode(f.fieldType, fmt.Sprintf("%s.%s", varName, f.name), f.fieldType.typeName()))
			buf.WriteString("\n")
		}
		return buf.String()
	case mapType:
		kVarName := fmt.Sprintf("_k_%s", replacedVarName)
		vVarName := fmt.Sprintf("_v_%s", replacedVarName)
		return fmt.Sprintf(`if in[_n] == 0 {
				_n += 1
				%s = nil
			} else {
				_n += 1
				_%sLen, _%sLenSize := binary.Varint(in[_n:])
				_n += _%sLenSize
				%s = make(%s, _%sLen)
				for _i := int64(0); _i < _%sLen; _i++ {
					var %s %s
					var %s %s
					%s
					%s
					%s[%s] = %s
				}
			}`, varName, replacedVarName, replacedVarName, replacedVarName, varName, t.typeName(), replacedVarName, replacedVarName, kVarName, t.key.typeName(), vVarName, t.value.typeName(), fieldTypeToDecode(t.key, kVarName, t.key.typeName()), fieldTypeToDecode(t.value, vVarName, t.value.typeName()), varName, kVarName, vVarName)
	case arrayType:
		eVarName := fmt.Sprintf("_e_%s", replacedVarName)
		return fmt.Sprintf(`for _i := 0; _i < %s; _i++ {
				var %s %s
				%s
				%s[_i] = %s
			}`, t.len, eVarName, t.elem.typeName(), fieldTypeToDecode(t.elem, eVarName, t.elem.typeName()), varName, eVarName)
	case sliceType:
		eVarName := fmt.Sprintf("_e_%s", replacedVarName)
		return fmt.Sprintf(`if in[_n] == 0 {
				_n += 1
				%s = nil
			} else {
				_n += 1
				_%sLen, _%sLenSize := binary.Varint(in[_n:])
				_n += _%sLenSize
				%s = make(%s, _%sLen)
				for _i := int64(0); _i < _%sLen; _i++ {
					var %s %s
					%s
					%s[_i] = %s
				}
			}`, varName, replacedVarName, replacedVarName, replacedVarName, varName, t.typeName(), replacedVarName, replacedVarName, eVarName, t.elem.typeName(), fieldTypeToDecode(t.elem, eVarName, t.elem.typeName()), varName, eVarName)
	case pointerType:
		pVarName := fmt.Sprintf("_p_%s", replacedVarName)
		return fmt.Sprintf(`if in[_n] == 0 {
				_n += 1
				%s = nil
			} else {
				_n += 1
				var %s %s
				%s
				%s = &%s
			}`, varName, pVarName, t.value.typeName(), fieldTypeToDecode(t.value, pVarName, t.value.typeName()), varName, pVarName)
	case externalStructType:
		if t.packageName == "time" && t.name == "Time" {
			return fmt.Sprintf(`if in[_n] == 1 {
					err = %s.UnmarshalBinary(in[_n : _n+15])
				} else {
					err = %s.UnmarshalBinary(in[_n : _n+16])
				}
				if err != nil {
					return 0, err
				}
				_n += 16`, varName, varName)
		}
		eVarName := fmt.Sprintf("_e_%s", replacedVarName)
		return fmt.Sprintf(`var %s %s.%s
			_%sSize, err := %s.Decode%s(in[_n:], &%s)
			if err != nil {
				return 0, err
			}
			%s = %s
			_n += _%sSize`, eVarName, t.packageName, t.name, replacedVarName, t.packageName, t.name, eVarName, varName, eVarName, replacedVarName)
	}
	return ""
}

type fieldType interface {
	fieldName() string
	typeName() string
}

type primitiveType struct {
	name string
}

func (t primitiveType) fieldName() string {
	return t.name
}

func (t primitiveType) typeName() string {
	return t.name
}

type structType struct {
	name   string
	fields []field
}

func (t structType) fieldName() string {
	return t.name
}

func (t structType) typeName() string {
	return t.name
}

type mapType struct {
	key   fieldType
	value fieldType
}

func (t mapType) fieldName() string {
	return ""
}

func (t mapType) typeName() string {
	return fmt.Sprintf("map[%s]%s", t.key.typeName(), t.value.typeName())
}

type arrayType struct {
	len  string
	elem fieldType
}

func (t arrayType) fieldName() string {
	return ""
}

func (t arrayType) typeName() string {
	return fmt.Sprintf("[%s]%s", t.len, t.elem.typeName())
}

type sliceType struct {
	elem fieldType
}

func (t sliceType) fieldName() string {
	return ""
}

func (t sliceType) typeName() string {
	return fmt.Sprintf("[]%s", t.elem.typeName())
}

type pointerType struct {
	value fieldType
}

func (t pointerType) fieldName() string {
	return t.value.fieldName()
}

func (t pointerType) typeName() string {
	return fmt.Sprintf("*%s", t.value.typeName())
}

type externalStructType struct {
	packageName string
	name        string
}

func (t externalStructType) fieldName() string {
	return t.name
}

func (t externalStructType) typeName() string {
	return fmt.Sprintf("%s.%s", t.packageName, t.name)
}

type aliasType struct {
	aliasName string
	value     fieldType
}

func (t aliasType) fieldName() string {
	return t.aliasName
}

func (t aliasType) typeName() string {
	return t.aliasName
}

func parseTypeSpec(ts *ast.TypeSpec) field {
	return field{
		fieldType: parseField(ts.Type),
	}
}

func parseFieldList(fl *ast.FieldList) []field {
	fields := make([]field, 0, len(fl.List))
	for _, f := range fl.List {
		ft := parseField(f.Type)
		if ft == nil {
			continue
		}
		// unnamed parameters
		fieldName := ft.fieldName()
		if len(f.Names) > 0 {
			fieldName = f.Names[0].Name
		}
		fields = append(fields, field{
			name:      fieldName,
			fieldType: ft,
		})
	}
	return fields
}

func parseField(expr ast.Expr) fieldType {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Obj == nil {
			return primitiveType{
				name: t.Name,
			}
		}
		ts, ok := t.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			// func の場合等
			return nil
		}
		return parseStruct(ts, t.Obj.Name)
	case *ast.StarExpr:
		return pointerType{
			value: parseField(t.X),
		}
	case *ast.MapType:
		return mapType{
			key:   parseField(t.Key),
			value: parseField(t.Value),
		}
	case *ast.ArrayType:
		if t.Len != nil {
			bl, ok := t.Len.(*ast.BasicLit)
			if ok {
				return arrayType{
					len:  bl.Value,
					elem: parseField(t.Elt),
				}
			}
			// TODO 確認
			return nil
		}
		return sliceType{
			elem: parseField(t.Elt),
		}
	case *ast.SelectorExpr:
		p, ok := t.X.(*ast.Ident)
		if ok {
			usedPackageMap[p.Name] = packageMap[p.Name]
			return externalStructType{
				packageName: p.Name,
				name:        t.Sel.Name,
			}
		}
		// TODO 確認
		return nil
	case *ast.StructType:
		// unnamed struct
		var out strings.Builder
		if err := format.Node(&out, fset, t); err != nil {
			log.Fatalf("format.Node: %v", err)
		}
		return structType{
			name:   out.String(),
			fields: parseFieldList(t.Fields),
		}
	}
	// interfaceの場合等
	return nil
}

func parseStruct(ts *ast.TypeSpec, name string) fieldType {
	switch t := ts.Type.(type) {
	case *ast.StructType:
		return structType{
			name:   name,
			fields: parseFieldList(t.Fields),
		}
	}

	return aliasType{
		aliasName: name,
		value:     parseField(ts.Type),
	}
}

type param struct {
	PackageName       string
	ReceiverIsPointer bool
	ImportPackageMap  map[string]string
	StructData        []structParam
}

type structParam struct {
	StructName   string
	ReceiverName string
	EncodeField  field
	DecodeField  field
}

var Template = template.Must(template.New("").
	Funcs(template.FuncMap{
		"safehtml": func(text string) template.HTML { return template.HTML(text) },
	}).
	Parse(
		`// Code generated by github.com/yutakahashi114/isutool; DO NOT EDIT.
package {{ .PackageName }}

import (
	"encoding/binary"
	"errors"
	"math"
	"time"

{{ range $k, $v := .ImportPackageMap }}{{ $k|safehtml }} {{ $v|safehtml }}
{{ end }}
)

var (
	_ = math.MaxUint8
)

{{ $ReceiverIsPointer := .ReceiverIsPointer }}
{{ range .StructData }}
func ({{ .ReceiverName }} {{ if $ReceiverIsPointer }}*{{ end }}{{ .StructName }}) Encode() ([]byte, error) {
	out := make([]byte, {{ .ReceiverName }}.MaxSize())

	_n, err := {{ .ReceiverName }}.EncodeTo(out)
	if err != nil {
		return nil, err
	}

	return out[:_n], nil
}

func ({{ .ReceiverName }} {{ if $ReceiverIsPointer }}*{{ end }}{{ .StructName }}) Decode(in []byte) ({{ if $ReceiverIsPointer }}*{{ end }}{{ .StructName }}, error) {
	_, err := Decode{{ .StructName }}(in, {{ if not $ReceiverIsPointer }}&{{ end }}{{ .ReceiverName }})
	if err != nil {
		return {{ .ReceiverName }}, err
	}

	return {{ .ReceiverName }}, nil
}

func ({{ .ReceiverName }} {{ if $ReceiverIsPointer }}*{{ end }}{{ .StructName }}) MaxSize() int {
	_size := 0

{{ .EncodeField.MaxSize .ReceiverName }}
	return _size
}

func ({{ .ReceiverName }} {{ if $ReceiverIsPointer }}*{{ end }}{{ .StructName }}) EncodeTo(out []byte) (int, error) {
	_timeMarshalBinary := func(t time.Time, out []byte) (int, error) {
		var timeZero = time.Time{}.Unix()

		// cf. https://github.com/golang/go/blob/dc00aed6de101700fd02b30f93789b9e9e1fe9a1/src/time/time.go#L1206
		var offsetMin int16 // minutes east of UTC. -1 is UTC.
		var offsetSec int8
		version := 1

		if t.Location() == time.UTC {
			offsetMin = -1
		} else {
			_, offset := t.Zone()
			if offset%60 != 0 {
				version = 2
				offsetSec = int8(offset % 60)
			}

			offset /= 60
			if offset {{ "<"|safehtml }} -32768 || offset == -1 || offset > 32767 {
				return 0, errors.New("TimeMarshalBinary: unexpected zone offset")
			}
			offsetMin = int16(offset)
		}

		unix := t.Unix()
		sec := unix - timeZero
		nsec := t.UnixNano() - unix*1000000000
		out[0] = byte(version)   // byte 0 : version
		out[1] = byte(sec >> 56) // bytes 1-8: seconds
		out[2] = byte(sec >> 48)
		out[3] = byte(sec >> 40)
		out[4] = byte(sec >> 32)
		out[5] = byte(sec >> 24)
		out[6] = byte(sec >> 16)
		out[7] = byte(sec >> 8)
		out[8] = byte(sec)
		out[9] = byte(nsec >> 24) // bytes 9-12: nanoseconds
		out[10] = byte(nsec >> 16)
		out[11] = byte(nsec >> 8)
		out[12] = byte(nsec)
		out[13] = byte(offsetMin >> 8) // bytes 13-14: zone offset in minutes
		out[14] = byte(offsetMin)

		if version == 2 {
			out[15] = byte(offsetSec)
		}

		return 16, nil
	}
	_ = _timeMarshalBinary

	_n := 0

{{ .EncodeField.Encode .ReceiverName }}

	return _n, nil
}

func Decode{{ .StructName }}(in []byte, {{ .ReceiverName }} *{{ .StructName }}) (_n int, err error) {
{{ if $ReceiverIsPointer }}_n += 1
	if in[0] == 0 {
		return
	}
{{ end }}
{{ .DecodeField.Decode .ReceiverName .StructName }}
	return _n, nil
}
{{ end }}
`))
