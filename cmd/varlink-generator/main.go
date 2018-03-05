package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/varlink/go/varlink"
)

func help(name string) {
	fmt.Printf("Usage: %s <package> <file>\n", name)
	os.Exit(1)
}

func writeTypeString(b *bytes.Buffer, t *varlink.IDLType) {
	switch t.Kind {
	case varlink.IDLTypeBool:
		b.WriteString("bool")

	case varlink.IDLTypeInt:
		b.WriteString("int64")

	case varlink.IDLTypeFloat:
		b.WriteString("float64")

	case varlink.IDLTypeString, varlink.IDLTypeEnum:
		b.WriteString("string")

	case varlink.IDLTypeArray:
		b.WriteString("[]")
		writeTypeString(b, t.ElementType)

	case varlink.IDLTypeAlias:
		b.WriteString(t.Alias)

	case varlink.IDLTypeStruct:
		b.WriteString("struct {")
		for i, field := range t.Fields {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(field.Name + " ")
			writeTypeString(b, field.Type)
		}
		b.WriteString("}")
	}
}

func writeType(b *bytes.Buffer, name string, t *varlink.IDLType) {
	if len(t.Fields) == 0 {
		return
	}

	b.WriteString("type " + name + " struct {\n")
	for _, field := range t.Fields {
		name := strings.Title(field.Name)
		b.WriteString("\t" + name + " ")
		writeTypeString(b, field.Type)
		b.WriteString(" `json:\"" + field.Name)

		switch field.Type.Kind {
		case varlink.IDLTypeStruct, varlink.IDLTypeString, varlink.IDLTypeEnum, varlink.IDLTypeArray:
			b.WriteString(",omitempty")
		}

		b.WriteString("\"`\n")
	}
	b.WriteString("}\n\n")
}

func main() {
	if len(os.Args) < 2 {
		help(os.Args[0])
	}

	varlinkFile := os.Args[1]

	file, err := ioutil.ReadFile(varlinkFile)
	if err != nil {
		fmt.Printf("Error reading file '%s': %s\n", varlinkFile, err)
	}

	description := strings.TrimRight(string(file), "\n")
	idl, err := varlink.NewIDL(description)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	pkgname := strings.Replace(idl.Name, ".", "", -1)

	var b bytes.Buffer
	b.WriteString("// Generated with varlink-generator -- https://github.com/varlink/go/cmd/varlink-generator\n\n")
	b.WriteString("package " + pkgname + "\n\n")
	b.WriteString(`import "github.com/varlink/go/varlink"` + "\n\n")

	for _, member := range idl.Members {
		switch member.(type) {
		case *varlink.IDLAlias:
			alias := member.(*varlink.IDLAlias)
			writeType(&b, alias.Name, alias.Type)

		case *varlink.IDLMethod:
			method := member.(*varlink.IDLMethod)
			writeType(&b, method.Name+"_In", method.In)
			writeType(&b, method.Name+"_Out", method.Out)

		case *varlink.IDLError:
			err := member.(*varlink.IDLError)
			writeType(&b, err.Name+"_Error", err.Type)
		}
	}

	b.WriteString("func NewInterfaceDefinition() varlink.InterfaceDefinition {\n" +
		"\treturn varlink.InterfaceDefinition{\n" +
		"\t\tName:        `" + idl.Name + "`,\n" +
		"\t\tDescription: `" + idl.Description + "`,\n" +
		"\t\tMethods: map[string]struct{}{\n")
	for m := range idl.Methods {
		b.WriteString("\t\t\t\"" + m + `": {},` + "\n")
	}
	b.WriteString("\t\t},\n\t}\n}\n")

	filename := path.Dir(varlinkFile) + "/" + pkgname + ".go"
	err = ioutil.WriteFile(filename, b.Bytes(), 0660)
	if err != nil {
		fmt.Printf("Error writing file '%s': %s\n", filename, err)
	}
}
