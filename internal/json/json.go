package json

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

// ProtoParser defines the interface for parsing proto files dynamically.
type ProtoParser interface {
	ParseFiles(filenames ...string) ([]*desc.FileDescriptor, error)
}

// Converter converts proto message to json using definition provided by ProtoParser.
type Converter struct {
	Parser               ProtoParser
	Filename             string
	Package, MessageType string
	Indent               bool
}

// Convert converts proto message to json.
func (c Converter) Convert(r io.Reader) ([]byte, error) {
	files, err := c.Parser.ParseFiles(c.Filename)
	if err != nil {
		return nil, err
	}

	fd, err := desc.CreateFileDescriptor(files[0].AsFileDescriptorProto(), files[0].GetDependencies()...)
	if err != nil {
		return nil, err
	}

	if c.Package == "" {
		c.Package = fd.GetPackage()
	}
	if c.MessageType == "" && len(fd.GetMessageTypes()) > 0 {
		c.MessageType = fd.GetMessageTypes()[0].GetName()
	}

	symbol := fd.FindSymbol(fmt.Sprintf("%s.%s", c.Package, c.MessageType))
	if symbol == nil {
		return nil, fmt.Errorf("can't find %s in %s package", c.MessageType, c.Package)
	}

	md := symbol.(*desc.MessageDescriptor)
	dm := dynamic.NewMessage(md)

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = dm.Unmarshal(b)
	if err != nil {
		return nil, err
	}

	json, err := c.marshalJSON(dm)
	if err != nil {
		return nil, err
	}

	return json, nil
}

func (c Converter) marshalJSON(dm *dynamic.Message) ([]byte, error) {
	if c.Indent {
		return dm.MarshalJSONIndent()
	}

	return dm.MarshalJSON()
}
