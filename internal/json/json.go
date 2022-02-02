package json

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

// DefaultLineSeparator is the default line separator that is used when converting streams, unless otherwise specified.
const DefaultLineSeparator = "--END--"

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
	LineSeparator        string
}

// Convert converts proto message to json.
func (c Converter) Convert(r io.Reader) ([]byte, error) {
	md, err := c.createProtoMessageDescriptor()
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	parsed, err := c.unmarshalProtoBytesToJson(md, b)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

// ConvertStream converts multiple proto messages to json.
// It returns a result channel and error channel which both can return multiple messages (a result or error for each message)
// Because proto messages often contain newlines, we can't rely on new lines for knowing when one message ends and the
// next begins, so instead it looks for a line containing only a specified LineSeparator (defaults to DefaultLineSeparator).
// Although unlikely, it is possible that the LineSeparator can be part of the proto binary message, in which case the
// parsing of that message will fail. If this happens, use a more complex LineSeparator.
func (c Converter) ConvertStream(r io.Reader) (resultCh chan []byte, errorCh chan error) {
	resultCh = make(chan []byte)
	errorCh = make(chan error)
	if c.LineSeparator == "" {
		c.LineSeparator = DefaultLineSeparator
	}

	md, err := c.createProtoMessageDescriptor()
	if err != nil {
		go func() {
			errorCh <- err
			close(resultCh)
			close(errorCh)
		}()
		return
	}

	go func() {
		reader := bufio.NewReader(r)
		var buf bytes.Buffer
		for {
			// Go over the stream line by line, as streams like Kafka send messages on next lines
			line, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			// If the line is equal to c.LineSeparator (and a newline as reader.ReadBytes does not strip that), we know
			// the message is finished, so we can start processing it.
			if bytes.Equal(line, []byte(c.LineSeparator+"\n")) {
				parsed, err := c.unmarshalProtoBytesToJson(md, stripTrailingNewline(buf.Bytes()))
				if err != nil {
					errorCh <- err
				} else {
					resultCh <- parsed
				}
				buf.Reset()
			} else {
				buf.Write(line)
			}
		}

		// Process whatever is remaining on the read buffer
		b := stripTrailingNewline(buf.Bytes())
		if len(b) > 0 {
			parsed, err := c.unmarshalProtoBytesToJson(md, b)
			if err != nil {
				errorCh <- err
			} else {
				resultCh <- parsed
			}
		}
		close(resultCh)
		close(errorCh)
	}()
	return
}

func (c Converter) createProtoMessageDescriptor() (*desc.MessageDescriptor, error) {
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

	return symbol.(*desc.MessageDescriptor), nil
}

func (c Converter) unmarshalProtoBytesToJson(md *desc.MessageDescriptor, rawMessage []byte) ([]byte, error) {
	dm := dynamic.NewMessage(md)
	err := dm.Unmarshal(rawMessage)
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

func stripTrailingNewline(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		return b[:len(b)-1]
	}
	return b
}
