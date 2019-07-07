package main

import "text/template"

func getConstantsTemplate() *template.Template {
	const constTemplate = `
// Package amqp for read, write, parse amqp frames
// Autogenerated code. Do not edit.
package amqp
{{range .}}
{{.Doc}}
const {{.GoName}} = {{.Value}}
{{end}}

// ConstantsNameMap map for mapping error codes into error messages
var ConstantsNameMap = map[uint16]string{
{{range .}}
{{if eq .IgnoreOnMap false}}
{{.Value}}: "{{.GoStr}}",{{end}}
{{end}}
}
`
	return template.Must(template.New("constTemplate").Parse(constTemplate))
}

func getMethodsTemplate() *template.Template {
	const methodsTemplate = `
// Package amqp for read, write, parse amqp frames
// Autogenerated code. Do not edit.
package amqp

import ( 
	"io"
	"fmt"
	"time"
)

// Method represents base method interface
type Method interface {
	Name() string
	FrameType() byte
	ClassIdentifier() uint16
	MethodIdentifier() uint16
	Read(reader io.Reader, protoVersion string) (err error)
	Write(writer io.Writer, protoVersion string) (err error)
	Sync() bool
}
{{range .}}
{{$classId := .ID}}
// {{.GoName}} methods

{{ if .Fields }}
// {{.GoName}}PropertyList represents properties for {{.GoName}} method
type {{.GoName}}PropertyList struct {
{{range .Fields}}{{.GoName}} {{if ne .GoType "*Table"}}*{{end}}{{.GoType}}
{{end}}
}
// {{.GoName}}PropertyList reads properties from io reader
func (pList *{{.GoName}}PropertyList) Read(reader io.Reader, propertyFlags uint16, protoVersion string) (err error) {
{{range .Fields}}
	if propertyFlags&(1<<{{.HeaderIndex}}) != 0 {
		value, err := Read{{.ReaderFunc}}(reader{{if eq .ReaderFunc "Table"}}, protoVersion{{end}})
		if err != nil {
			return err
		}
		pList.{{.GoName}} = {{if ne .GoType "*Table"}}&{{end}}value
	}
{{end}}
	return
}
// {{.GoName}}PropertyList wiretes properties into io writer
func (pList *{{.GoName}}PropertyList) Write(writer io.Writer, protoVersion string) (propertyFlags uint16, err error) {
{{range .Fields}}
	if pList.{{.GoName}} != nil {
		propertyFlags |= 1 << {{.HeaderIndex}}
		if err = Write{{.ReaderFunc}}(writer, {{if ne .GoType "*Table"}}*{{end}}pList.{{.GoName}}{{if eq .ReaderFunc "Table"}}, protoVersion{{end}}); err != nil {
			return
		}
	}
{{end}}
	return
}
{{end}}
{{range .Methods}}
{{.Doc}} 
type {{.GoName}} struct {
{{range .Fields}}{{.GoName}} {{.GoType}}
{{end}}
}
// Name returns method name as string, usefully for logging
func (method *{{.GoName}}) Name() string {
    return "{{.GoName}}"
}

// FrameType returns method frame type
func (method *{{.GoName}}) FrameType() byte {
    return 1
}

// ClassIdentifier returns method classID
func (method *{{.GoName}}) ClassIdentifier() uint16 {
    return {{$classId}}
}

// MethodIdentifier returns method methodID
func (method *{{.GoName}}) MethodIdentifier() uint16 {
    return {{.ID}}
}

// Sync is method should me sent synchronous
func (method *{{.GoName}}) Sync() bool {
    return {{if eq .Synchronous 1}}true{{else}}false{{end}}
} 

// Read method from io reader
func (method *{{.GoName}}) Read(reader io.Reader, protoVersion string) (err error) {
{{range .Fields}}
	{{if .IsBit }}
	{{if eq .BitOrder 0}}
		bits, err := ReadOctet(reader)
		if err != nil {
			return err
		}
	{{end}}
		method.{{.GoName}} = bits&(1<<{{.BitOrder}}) != 0 
	{{else}}
	method.{{.GoName}}, err = Read{{.ReaderFunc}}(reader{{if eq .ReaderFunc "Table"}}, protoVersion{{end}})
	if err != nil {
		return err
	}
	{{end}}
    
{{end}}
	return
}

// Write method from io reader
func (method *{{.GoName}}) Write(writer io.Writer, protoVersion string) (err error) {
{{$bitFieldsStarted := false}}
{{range .Fields}}
	{{if .IsBit }}
	{{$bitFieldsStarted := true}}
	{{if eq .BitOrder 0}}
	var bits byte
	{{end}}
	if method.{{.GoName}} {
		bits |= 1 << {{.BitOrder}}
	}
	{{if .LastBit}}
	if err = WriteOctet(writer, bits); err != nil {
		return err
	}
	{{end}}
	{{else}}
	if err = Write{{.ReaderFunc}}(writer, method.{{.GoName}}{{if eq .ReaderFunc "Table"}}, protoVersion{{end}}); err != nil {
		return err
	}
	{{end}}
{{end}}
	return
}
{{end}}
{{end}}

/* 
ReadMethod reads method from frame's payload

Method frames carry the high-level protocol commands (which we call "methods").
One method frame carries one command.  The method frame payload has this format:

  0          2           4
  +----------+-----------+-------------- - -
  | class-id | method-id | arguments...
  +----------+-----------+-------------- - -
     short      short    ...

*/
func ReadMethod(reader io.Reader, protoVersion string) (Method, error) {
	classID, err := ReadShort(reader)
	if err != nil {
		return nil, err 
	}

	methodID, err := ReadShort(reader)
	if err != nil {
		return nil, err 
	}
	switch classID {
		{{range .}}
	case {{.ID}}:
		switch methodID {
			{{range .Methods}}
		case {{.ID}}:
			var method = &{{.GoName}}{}
			if err := method.Read(reader, protoVersion); err != nil {
				return nil, err
			}
			return method, nil{{end}}
		}{{end}}
	}

	return nil, fmt.Errorf("unknown classID and methodID: [%d. %d]", classID, methodID)
}

// WriteMethod writes method into frame's payload
func WriteMethod(writer io.Writer, method Method, protoVersion string) (err error) {
	if err = WriteShort(writer, method.ClassIdentifier()); err != nil {
		return err
	}
	if err = WriteShort(writer, method.MethodIdentifier()); err != nil {
		return err
	}

	if err = method.Write(writer, protoVersion); err != nil {
		return err
	}
	
	return
}
`
	return template.Must(template.New("methodsTemplate").Parse(methodsTemplate))
}