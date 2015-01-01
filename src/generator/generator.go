package generator

import (
	"fmt"
	"go/ast"
	"go/format"
	"strconv"
	"strings"
)

func Generate(filePath string) (data []byte, err error) {
	parser, err := NewProtoParser(filePath)
	if err != nil {
		return nil, err
	}
	if err = parser.Parse(); err != nil {
		return nil, err
	}

	packageName := "\npackage protocol"
	if parser.astFile.Name != nil {
		packageName = "package " + parser.astFile.Name.Name
	}

	content := packageName + "\n"
	content += addImportPackageCode() + "\n"
	content += addPacketID(parser.packets) + "\n"
	content += addPacketInterfaceCode() + "\n"
	content += addPacketHeaderCode() + "\n"

	for _, p := range parser.packets {
		var packetContent string
		var err error
		switch p.kind {
		case StructKind:
			packetContent, err = generateStruct(p)
		case GenericPacketKind:
			packetContent, err = generateGenericPacket(p)
		case SimplePacketKind:
			packetContent, err = generateSimplePacket(p)
		case VLFPacketKind:
			packetContent, err = generateVLFPacket(p)
		}
		if err != nil {
			return nil, err
		}
		content += packetContent + "\n"
	}

	packetFactoryCode, err := generatePacketFactory(parser.packets)
	if err != nil {
		return nil, err
	}
	content += packetFactoryCode + "\n"
	return format.Source([]byte(content))
}

func addPacketHeaderCode() string {
	return `
	type PacketHeader struct {		
		ID         uint32
		PacketType uint32
		Len        uint32
		Version    uint32
		Ack        uint32
		Token      uint32
	}

	func (p *PacketHeader) GetID() uint32 { return p.ID }

	func (p *PacketHeader) SetID(id uint32) { p.ID = id }

	func (p *PacketHeader) GetToken() uint32 { return p.Token }

	func (p *PacketHeader) SetToken(token uint32) { p.Token = token }

	func (p *PacketHeader) GetAck() uint32 { return p.Ack }

	func (p *PacketHeader) SetAck(ack uint32) { p.Ack = ack }

	func (p *PacketHeader) GetPacketType() uint32 { return p.PacketType }

	func (p *PacketHeader) Length() int { return 36 }
	
	func (p *PacketHeader) AdjustLength() { p.Len = uint32(p.Length()) }

	func (p *PacketHeader) Read(stream ReadStream) error {
		var err error
		if p.ID, err = stream.ReadUint32(); err != nil {
			return err
		}
		if p.PacketType, err = stream.ReadUint32(); err != nil {
			return err
		}
		if p.Len, err = stream.ReadUint32(); err != nil {
			return err
		}
		if p.Version, err = stream.ReadUint32(); err != nil {
			return err
		}
		if p.Ack, err = stream.ReadUint32(); err != nil {
			return err
		}
		if p.Token, err = stream.ReadUint32(); err != nil {
			return err
		}
		return nil
	}

	func (w *PacketHeader) Write(stream WriteStream) error {
		var err error
		if err = stream.WriteUint32(w.ID); err != nil {
			return err
		}
		if err = stream.WriteUint32(w.PacketType); err != nil {
			return err
		}
		if err = stream.WriteUint32(w.Len); err != nil {
			return err
		}
		if err = stream.WriteUint32(w.Version); err != nil {
			return err
		}
		if err = stream.WriteUint32(w.Ack); err != nil {
			return err
		}
		if err = stream.WriteUint32(w.Token); err != nil {
			return err
		}
		return nil
	}`
}

func addImportPackageCode() string {
	return `
		import "errors"
		
		var ErrUnknownPacket = errors.New("unknown packet")
	`
}

func addPacketID(packets []*PacketLayout) string {
	if len(packets) != 0 {
		content := "const (\n"
		for _, p := range packets {
			if p.kind != StructKind {
				content += fmt.Sprintf("%s = 0x%08x\n", strings.ToUpper(p.idname), p.id)
			}
		}
		content += ")"
		return content

	}
	return ""
}

func addPacketInterfaceCode() string {
	return `
	type Packet interface {
		GetID() uint32
		SetID(uint32)
		GetToken() uint32
		SetToken(uint32)
		GetAck() uint32
		SetAck(uint32)
		GetPacketType() uint32
		Length() int
		AdjustLength()
		Read(stream ReadStream) error
		Write(stream WriteStream) error
	}`
}

func getArrayTypeLength(field *ast.Field) (i int, err error) {
	if a, ok := field.Type.(*ast.ArrayType); ok {
		if b, ok := a.Len.(*ast.BasicLit); ok {
			return strconv.Atoi(b.Value)
		}
	}

	return 0, fmt.Errorf("invalid field")
}

func generateStructData(p *PacketLayout) (s string, err error) {
	structContent := fmt.Sprintf("\ntype %s struct {\n", p.name)
	if p.kind != StructKind {
		structContent += "    PacketHeader\n"
	}
	for _, f := range p.fields {
		if f.kind == ArrayFieldKind {
			index, err := getArrayTypeLength(f.field)
			if err != nil {
				return "", err
			}
			structContent += fmt.Sprintf("    %s [%d]%s\n", f.name, index, f.fieldType)
		} else if f.kind == SliceFieldKind {
			structContent += fmt.Sprintf("    %s []%s\n", f.name, f.fieldType)
		} else {
			structContent += fmt.Sprintf("    %s %s\n", f.name, f.fieldType)
		}
	}
	structContent += fmt.Sprintf("\n}")
	return structContent, nil
}

func generateStruct(p *PacketLayout) (s string, err error) {
	structContent, err := generateStructData(p)
	if err != nil {
		return "", err
	}

	code, err := generatePacketLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generatePacketAdjustLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateReadCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateWriteCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	return structContent, nil
}

func generateSimplePacket(p *PacketLayout) (s string, err error) {
	structContent, err := generateStructData(p)
	if err != nil {
		return "", err
	}

	structContent += fmt.Sprintf("\nfunc (s *%s) Length() int { return s.PacketHeader.Length() }", p.name)
	structContent += fmt.Sprintf("\nfunc (s *%s) AdjustLength() { s.Len = uint32(s.Length()) }", p.name)
	structContent += fmt.Sprintf("\nfunc (s *%s) Read(stream ReadStream) error { return nil}", p.name)
	structContent += fmt.Sprintf("\nfunc (s *%s) Write(stream WriteStream) error { return s.PacketHeader.Write(stream) }", p.name)

	code, err := generateNewPacketFunc(p)
	if err != nil {
		return "", err
	}
	structContent += code

	return structContent, nil
}

func generateGenericPacket(p *PacketLayout) (s string, err error) {
	structContent, err := generateStructData(p)
	if err != nil {
		return "", err
	}

	code, err := generateNewPacketFunc(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generatePacketLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generatePacketAdjustLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateReadCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateWriteCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	return structContent, nil
}

func generateVLFPacket(p *PacketLayout) (s string, err error) {
	structContent, err := generateStructData(p)
	if err != nil {
		return "", err
	}

	code, err := generateNewPacketFunc(p)
	if err != nil {
		return "", err
	}
	structContent += code

	code, err = generatePacketLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generatePacketAdjustLengthCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateReadCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	code, err = generateWriteCode(p)
	if err != nil {
		return "", err
	}
	structContent += "\n" + code

	return structContent, nil
}

func generatePacketAdjustLengthCode(p *PacketLayout) (s string, err error) {
	if p.kind != StructKind {
		s = fmt.Sprintf("func (s *%s) AdjustLength() { s.PacketHeader.Len = uint32(s.Length()) }", p.name)
	} else {
		s = fmt.Sprintf("func (s *%s) AdjustLength() {}", p.name)
	}
	return s, nil
}

func generatePacketLengthCode(p *PacketLayout) (s string, err error) {
	code := fmt.Sprintf("func (s *%s) Length() int {\nvar totalLength int", p.name)
	if p.kind != StructKind {
		code += "\ntotalLength +=s.PacketHeader.Length()"
	}
	for _, f := range p.fields {
		if v, ok := FieldKindLengthMap[f.kind]; ok {
			code += "\ntotalLength += " + v
		} else {
			if f.kind == StructFieldKind {
				code += fmt.Sprintf("\ntotalLength += s.%s.Length()", f.name)

			} else if f.kind == SliceFieldKind {
				code += fmt.Sprintf("\ntotalLength += 4")
				if f.subElementKind == StructFieldKind {
					code += fmt.Sprintf("\nfor i := 0; i < len(s.%s); i++ {\ntotalLength += s.%s[i].Length()\n}", f.name, f.name)
				} else if f.subElementKind == StringFieldKind {
					code += fmt.Sprintf("\nfor i := 0; i < len(s.%s); i++ {\ntotalLength += 4\ntotalLength += len(s.%s[i])\n}", f.name, f.name)
				} else {
					code += fmt.Sprintf("\ntotalLength += len(s.%s) * %s", f.name, FieldKindLengthMap[f.subElementKind])
				}

			} else if f.kind == ArrayFieldKind {
				arrayIndex, err := getArrayTypeLength(f.field)
				if err != nil {
					return "", nil
				}
				if f.subElementKind == StructFieldKind {
					code += fmt.Sprintf("\nfor i := 0; i < %d; i++ {\ntotalLength += s.%s[i].Length()\n}", arrayIndex, f.name)

				} else if f.subElementKind == StringFieldKind {
					code += fmt.Sprintf("\nfor i := 0; i < %d; i++ {\ntotalLength += 4\ntotalLength += len(s.%s[i])\n}", arrayIndex, f.name)
				} else {
					code += fmt.Sprintf("\ntotalLength += %d * %s", arrayIndex, FieldKindLengthMap[f.subElementKind])
				}
			} else if f.kind == StringFieldKind {
				code += fmt.Sprintf("\ntotalLength += 4")
				code += fmt.Sprintf("\ntotalLength += len(s.%s)", f.name)
			}
		}
	}
	code += "\nreturn totalLength\n}"
	return code, nil
}

func generateArrayFieldReadCode(fieldKind FieldKind, fieldName, fieldTypeName string, slice bool) string {
	code := "\n{\n"
	if slice {
		code += fmt.Sprintf("var size uint32\nif size, err = stream.ReadUint32(); err != nil {\nreturn err\n}\n")
		code += fmt.Sprintf("s.%s = make([]%s, size)\n", fieldName, fieldTypeName)
	}

	if fieldKind == ByteFieldKind || fieldKind == Int8FieldKind || fieldKind == Uint8FieldKind {
		code += fmt.Sprintf("if buff, err := stream.ReadBuff(int(size)); err != nil { return err } else { s.%s = []%s(buff[:]) }", fieldName, fieldTypeName)
		code += "\n}\n"
		return code
	}

	code += fmt.Sprintf("for i := 0; i < len(s.%s); i++{\n", fieldName)
	switch fieldKind {
	case StructFieldKind:
		{
			code += fmt.Sprintf("if err = s.%s[i].Read(stream); err != nil {\nreturn err\n}\n", fieldName)
		}
	case StringFieldKind:
		{
			code += fmt.Sprintf("var elementSize uint32\n"+
				"if elementSize, err = stream.ReadUint32(); err != nil{\nreturn err\n}\n"+
				"var buff []byte\n"+
				"if buff, err = stream.ReadBuff(int(elementSize)); err != nil {\nreturn err\n}\n"+
				"s.%s[i] = string(buff)\n", fieldName)
		}
	case Int8FieldKind:
		{
			code += fmt.Sprintf("if val, err := stream.ReadByte(); err != nil { returne err } else { s.%s[i] = int8(val) }", fieldName)
		}
	case Int16FieldKind:
		{
			code += fmt.Sprintf("if val, err := stream.ReadUint16(); err != nil { return err } else { s.%s[i] = int16(val) }", fieldName)
		}
	case Int32FieldKind:
		{
			code += fmt.Sprintf("if val, err := stream.ReadUint32(); err != nil { return err } else { s.%s[i] = int32(val) }", fieldName)
		}
	case Int64FieldKind:
		{
			code += fmt.Sprintf("if val, err := stream.ReadUint64(); err != nil { return err } else { s.%s[i] = int64(val) }", fieldName)
		}
	case Uint8FieldKind:
		{
			code += fmt.Sprintf("if val, err := stream.ReadByte(); err != nil { returne err } else { s.%s[i] = uint8(val) }", fieldName)
		}
	case Uint16FieldKind:
		{
			code += fmt.Sprintf("if s.%s[i], err = stream.ReadUint16(); err != nil { return err } ", fieldName)
		}
	case Uint32FieldKind:
		{
			code += fmt.Sprintf("if s.%s[i], err = stream.ReadUint32(); err != nil { return err } ", fieldName)
		}
	case Uint64FieldKind:
		{
			code += fmt.Sprintf("if s.%s[i], err = stream.ReadUint64(); err != nil { return err } ", fieldName)
		}

	}
	code += "\n}\n}\n"
	return code
}

func generateReadCode(p *PacketLayout) (s string, err error) {
	code := fmt.Sprintf("func (s *%s) Read(stream ReadStream) error {\nvar err error\n", p.name)
	for _, f := range p.fields {
		switch f.kind {
		case StructFieldKind:
			{
				code += fmt.Sprintf("\nif err = s.%s.Read(stream); err != nil { return err }", f.name)
			}
		case SliceFieldKind:
			{
				code += generateArrayFieldReadCode(f.subElementKind, f.name, f.fieldType, true)
			}
		case ArrayFieldKind:
			{
				code += generateArrayFieldReadCode(f.subElementKind, f.name, f.fieldType, false)
			}
		case StringFieldKind:
			{
				code += fmt.Sprintf("{\nvar size uint32"+
					"\nif size, err = stream.ReadUint32(); err != nil { return err }"+
					"\nvar buff []byte"+
					"\nif buff, err = stream.ReadBuff(int(size)); err != nil { return err }"+
					"\ns.%s = string(buff)\n}\n", f.name)
			}
		case ByteFieldKind:
			{
				code += fmt.Sprintf("if s.%s, err = stream.ReadByte(); err != nil { return err }\n", f.name)
			}
		case Int8FieldKind:
			{
				code += fmt.Sprintf("if val, err := stream.ReadByte(); err != nil { return err } else { s.%s = int8(val) }\n", f.name)
			}
		case Int16FieldKind:
			{
				code += fmt.Sprintf("if val, err := stream.ReadUint16(); err != nil { return err } else { s.%s = int16(val) }\n", f.name)
			}
		case Int32FieldKind:
			{
				code += fmt.Sprintf("if val, err := stream.ReadUint32(); err != nil { return err } else { s.%s = int32(val) }\n", f.name)
			}
		case Int64FieldKind:
			{
				code += fmt.Sprintf("if val, err := stream.ReadUint64(); err != nil { return err } else { s.%s = int64(val) }\n", f.name)
			}
		case Uint8FieldKind:
			{
				code += fmt.Sprintf("if s.%s, err = stream.ReadByte(); err != nil { return err }\n ", f.name)
			}
		case Uint16FieldKind:
			{
				code += fmt.Sprintf("if s.%s, err = stream.ReadUint16(); err != nil { return err }\n ", f.name)
			}
		case Uint32FieldKind:
			{
				code += fmt.Sprintf("if s.%s, err = stream.ReadUint32(); err != nil { return err }\n ", f.name)
			}
		case Uint64FieldKind:
			{
				code += fmt.Sprintf("if s.%s, err = stream.ReadUint64(); err != nil { return err }\n ", f.name)
			}
		}
	}
	code += "\nreturn nil\n}"
	return code, nil
}

func generateArrayFieldWriteCode(fieldKind FieldKind, fieldName string, slice bool) string {
	code := "\n{\n"
	if slice {
		code += fmt.Sprintf("if err = stream.WriteUint32(uint32(len(s.%s))); err != nil { return err }\n", fieldName)
	}

	if fieldKind == ByteFieldKind {
		code += fmt.Sprintf("if err = stream.WriteBuff(s.%s); err != nil { return err }", fieldName)
		code += "\n}\n"
		return code
	}

	code += fmt.Sprintf("for i := 0; i < len(s.%s); i++{\n", fieldName)
	switch fieldKind {
	case StructFieldKind:
		{
			code += fmt.Sprintf("if err = s.%s[i].Write(stream); err != nil { return err }", fieldName)
		}
	case StringFieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint32(uint32(len(s.%s[i]))); err != nil { return err }\n", fieldName)
			code += fmt.Sprintf("if err = stream.WriteBuff([]byte(s.%s[i])); err != nil { return err }", fieldName)
		}
	case Int8FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteByte(byte(s.%s[i])); err != nil { return err } ", fieldName)
		}
	case Int16FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint16(uint16(s.%s[i])); err != nil { return err }", fieldName)
		}
	case Int32FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint32(uint32(s.%s[i])); err != nil { return err }", fieldName)
		}
	case Int64FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint64(uint64(s.%s[i])); err != nil { return err } ", fieldName)
		}
	case Uint8FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteByte(s.%s[i]); err != nil { return err } ", fieldName)
		}
	case Uint16FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint16(s.%s[i]); err != nil { return err } ", fieldName)
		}
	case Uint32FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint32(s.%s[i]); err != nil { return err } ", fieldName)
		}
	case Uint64FieldKind:
		{
			code += fmt.Sprintf("if err = stream.WriteUint64(s.%s[i]); err != nil { return err } ", fieldName)
		}
	}
	code += "\n}\n}\n"
	return code
}

func generateWriteCode(p *PacketLayout) (s string, err error) {
	code := fmt.Sprintf("func (s *%s) Write(stream WriteStream) error {\nvar err error\n", p.name)
	if p.kind != StructKind {
		code += "if err = s.PacketHeader.Write(stream); err != nil { return err }\n"
	}
	for _, f := range p.fields {
		switch f.kind {
		case StructFieldKind:
			{
				code += fmt.Sprintf("if err = s.%s.Write(stream); err != nil { return err }", f.name)
			}
		case SliceFieldKind:
			{
				code += generateArrayFieldWriteCode(f.subElementKind, f.name, true)
			}
		case ArrayFieldKind:
			{
				code += generateArrayFieldWriteCode(f.subElementKind, f.name, false)
			}
		case StringFieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint32(uint32(len(s.%s))); err != nil { return err }"+
					"\nif err = stream.WriteBuff([]byte(s.%s)); err != nil { return err }", f.name, f.name)
			}
		case ByteFieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteByte(s.%s); err != nil { return err }", f.name)
			}
		case Int8FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteByte(byte(s.%s)); err != nil { return err }", f.name)
			}
		case Int16FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint16(uint16(s.%s)); err != nil { return err }", f.name)
			}
		case Int32FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint32(uint32(s.%s)); err != nil { return err }", f.name)
			}
		case Int64FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint64(uint64(s.%s)); err != nil { return err }", f.name)
			}
		case Uint8FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteByte(byte(s.%s)); err != nil { return err } ", f.name)
			}
		case Uint16FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint16(s.%s); err != nil { return err } ", f.name)
			}
		case Uint32FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint32(s.%s); err != nil { return err } ", f.name)
			}
		case Uint64FieldKind:
			{
				code += fmt.Sprintf("\nif err = stream.WriteUint64(s.%s); err != nil { return err } ", f.name)
			}

		}
	}
	code += "\nreturn nil\n}"
	return code, nil
}

func generateNewPacketFunc(p *PacketLayout) (s string, err error) {
	return fmt.Sprintf("\nfunc New%s() *%s { return &%s{\nPacketHeader:PacketHeader{\nPacketType:%s,\n},\n}\n}", p.name, p.name, p.name, strings.ToUpper(p.idname)), nil
}

func generatePacketFactory(packets []*PacketLayout) (s string, err error) {
	code := `
	type PacketCacher interface {
		Get(id uint32, header *PacketHeader) Packet
		Put(id uint32, packet Packet)
	}
	
	type PacketFactory struct {
		Cacher PacketCacher
	}
	
	func NewPacketFactory(cacher PacketCacher) *PacketFactory {
		return &PacketFactory{
			Cacher: cacher,
		}
	}
	
	func (p *PacketFactory) CreatePacket(stream ReadStream) (newPacket Packet, err error) {
		var header PacketHeader
		if err = header.Read(stream); err != nil {
			return nil, err
		}
		if p.Cacher != nil {
			newPacket = p.Cacher.Get(header.PacketType, &header)
		}
		if newPacket == nil {
			switch header.PacketType {
	`
	for _, p := range packets {
		if p.kind == StructKind {
			continue
		}
		code += fmt.Sprintf("case %s:\n        {\n           newPacket = &%s{PacketHeader:header}\n        }\n", strings.ToUpper(p.idname), p.name)
	}
	code += `
				default:
					{
						return nil, ErrUnknownPacket
					}
				}
			}
			if err = newPacket.Read(stream); err != nil {
				return nil, err
			}
			return newPacket, nil
		}`
	return code, nil
}
