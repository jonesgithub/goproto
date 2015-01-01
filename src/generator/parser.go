package generator

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
)

type PacketKind int

const (
	SimplePacketKind  PacketKind = 0x00
	VLFPacketKind     PacketKind = 0x01
	GenericPacketKind PacketKind = 0x02
	StructKind        PacketKind = 0x04
)

type FieldKind int

const (
	SliceFieldKind  FieldKind = 1 << iota
	ArrayFieldKind  FieldKind = 1 << iota
	StructFieldKind FieldKind = 1 << iota
	ByteFieldKind   FieldKind = 1 << iota
	Uint8FieldKind  FieldKind = 1 << iota
	Uint16FieldKind FieldKind = 1 << iota
	Uint32FieldKind FieldKind = 1 << iota
	Uint64FieldKind FieldKind = 1 << iota
	Int8FieldKind   FieldKind = 1 << iota
	Int16FieldKind  FieldKind = 1 << iota
	Int32FieldKind  FieldKind = 1 << iota
	Int64FieldKind  FieldKind = 1 << iota
	StringFieldKind FieldKind = 1 << iota
)

var FieldKindLengthMap = map[FieldKind]string{
	ByteFieldKind:   "1",
	Uint8FieldKind:  "1",
	Uint16FieldKind: "2",
	Uint32FieldKind: "4",
	Uint64FieldKind: "8",
	Int8FieldKind:   "1",
	Int16FieldKind:  "2",
	Int32FieldKind:  "4",
	Int64FieldKind:  "8",
}

type ProtoParser struct {
	astFile *ast.File
	packets []*PacketLayout
}

func NewProtoParser(file string) (*ProtoParser, error) {
	fileSet := token.NewFileSet()
	astFile, err := goparser.ParseFile(fileSet, file, nil, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	return &ProtoParser{
		astFile: astFile,
	}, nil
}

func (this *ProtoParser) Parse() error {
	var err error
	for _, decl := range this.astFile.Decls {
		var layout PacketLayout
		layout.kind, layout.idname, layout.id, err = this.parsePacketType(decl)
		if err != nil {
			break
		}
		layout.name, layout.structType = this.parseStructInfo(decl)
		if err = layout.parseField(); err != nil {
			break
		}
		this.packets = append(this.packets, &layout)
	}
	return err
}

func (this *ProtoParser) parsePacketType(decl ast.Decl) (kind PacketKind, IDName string, ID int, err error) {
	kind = StructKind

	genDecl := decl.(*ast.GenDecl)
	if genDecl.Doc != nil {
		// check the comment whether if SimplePacket, Packet, or VLFPacket
		for _, comment := range genDecl.Doc.List {
			text := strings.ToLower(comment.Text)
			text = strings.Replace(text, "/", "", -1)
			text = strings.Replace(text, " ", "", -1)
			typeComment := strings.Split(text, ":")
			if len(typeComment) < 2 {
				continue
			}

			switch typeComment[0] {
			case "@simplepacket":
				kind = SimplePacketKind
			case "@packet":
				kind = GenericPacketKind
			case "@vlfpacket":
				kind = VLFPacketKind
			}
			params := strings.Split(typeComment[1], ",")
			if len(params) < 2 {
				continue
			}
			IDName = params[0]
			var value int64
			value, err = strconv.ParseInt(params[1], 0, 64)
			ID = int(value)
		}
	}
	return
}

func (this *ProtoParser) parseStructInfo(decl ast.Decl) (name string, structType *ast.StructType) {
	genDecl := decl.(*ast.GenDecl)
	for _, spec := range genDecl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			if structType, ok = typeSpec.Type.(*ast.StructType); ok {
				name = typeSpec.Name.Name
				break
			}
		}
	}
	return
}

type PacketLayout struct {
	structType *ast.StructType
	kind       PacketKind
	name       string
	id         int
	idname     string
	fields     []*FieldLayout
}

type FieldLayout struct {
	field          *ast.Field
	kind           FieldKind
	name           string
	subElementKind FieldKind
	fieldType      string
}

func (p *PacketLayout) parseField() error {
	if p.kind == SimplePacketKind {
		return nil
	} else if p.kind == VLFPacketKind {
		if p.structType.Fields.NumFields() == 0 {
			return fmt.Errorf("VLFPacket Must have a slice field,", p.name)
		}
		field := p.structType.Fields.List[0]
		fieldLayout, err := NewFieldLayout(field)
		if err != nil {
			return err
		}
		if fieldLayout.kind != SliceFieldKind {
			return fmt.Errorf("VLFPacket Must have a slice field,", p.name)
		}
		p.fields = append(p.fields, fieldLayout)

	} else {
		for index := 0; index < len(p.structType.Fields.List); index++ {
			field := p.structType.Fields.List[index]
			if len(field.Names) == 0 {
				return fmt.Errorf("disallow anonymouse field except SimplePacketProperty, VLFPacketProperty, PacketProperty",
					"name:", p.name, "pos:", p.structType.Pos())
			}
			fieldLayout, err := NewFieldLayout(field)
			if err != nil {
				return err
			}
			p.fields = append(p.fields, fieldLayout)
		}
	}

	return nil
}

func NewFieldLayout(field *ast.Field) (*FieldLayout, error) {
	var fieldLayout FieldLayout
	fieldLayout.name = field.Names[0].Name
	fieldLayout.field = field
	if err := fieldLayout.parseFieldKind(); err != nil {
		return nil, err
	}
	if err := fieldLayout.parseFieldType(); err != nil {
		return nil, err
	}
	return &fieldLayout, nil
}

func (f *FieldLayout) parseFieldKind() error {
	switch t := f.field.Type.(type) {
	case *ast.StructType:
		{
			f.kind = StructFieldKind
		}
	case *ast.ArrayType:
		{
			if t.Len == nil {
				f.kind = SliceFieldKind
			} else {
				f.kind = ArrayFieldKind
			}
			return f.parseFieldType()
		}
	case *ast.Ident:
		{
			switch t.Name {
			case "byte":
				f.kind = ByteFieldKind
			case "uint8":
				f.kind = Uint8FieldKind
			case "uint16":
				f.kind = Uint16FieldKind
			case "uint32":
				f.kind = Uint32FieldKind
			case "uint64":
				f.kind = Uint64FieldKind
			case "int8":
				f.kind = Int8FieldKind
			case "int16":
				f.kind = Int16FieldKind
			case "int32":
				f.kind = Int32FieldKind
			case "int64":
				f.kind = Int64FieldKind
			case "string":
				f.kind = StringFieldKind
			default:
				f.kind = StructFieldKind
			}
		}
	default:
		{
			return fmt.Errorf("invalid type, pos:%d", f.field.Pos())
		}
	}
	return nil
}

func (f *FieldLayout) parseFieldType() error {
	f.fieldType = parseNameByType(f.field.Type)
	switch f.fieldType {
	case "struct":
		f.subElementKind = StructFieldKind
	case "byte":
		f.subElementKind = ByteFieldKind
	case "uint8":
		f.subElementKind = Uint8FieldKind
	case "uint16":
		f.subElementKind = Uint16FieldKind
	case "uint32":
		f.subElementKind = Uint32FieldKind
	case "uint64":
		f.subElementKind = Uint64FieldKind
	case "int8":
		f.subElementKind = Int8FieldKind
	case "int16":
		f.subElementKind = Int16FieldKind
	case "int32":
		f.subElementKind = Int32FieldKind
	case "int64":
		f.subElementKind = Int64FieldKind
	case "string":
		f.subElementKind = StringFieldKind
	default:
		f.subElementKind = StructFieldKind
	}
	return nil
}

func parseNameByType(exp ast.Expr) string {
	switch t := exp.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.SliceExpr:
		return parseNameByType(t.X)
	case *ast.ArrayType:
		return parseNameByType(t.Elt)
	case *ast.Ident:
		{
			return t.Name
		}
	}
	return "unknown"
}
