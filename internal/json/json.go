package json

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

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
	EndOfMessageMarker   string
}

// ConvertStream converts multiple proto messages to json.
// It returns a result channel and error channel which both can return multiple messages (a result or error for each message)
// Because proto messages often contain newlines, we can't rely on new lines for knowing when one message ends and the
// next begins, so instead it looks for a line containing only a specified marker (defaults to DefaultEndOfMessageMarker).
// Although unlikely, it is possible that the EndOfMessageMarker can be part of the proto binary message, in which case the
// parsing of that message will fail. If this happens, use a more complex EndOfMessageMarker.
func (c Converter) ConvertStream(r io.Reader) (resultCh chan []byte, errorCh chan error) {
	resultCh = make(chan []byte)
	errorCh = make(chan error)

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
		scanner := bufio.NewScanner(r)
		// Don't set an initial buffer, as the default scanner doesn't do so either
		scanner.Buffer(nil, 1024*1024)
		scanner.Split(splitMessagesOnMarker([]byte(c.EndOfMessageMarker)))
		for scanner.Scan() {
			rawBytes := scanner.Bytes()
			parsed, err := c.unmarshalProtoBytesToJson(md, rawBytes)
			if err != nil {
				errorCh <- err
			} else {
				resultCh <- parsed
			}
		}
		if err := scanner.Err(); err != nil {
			errorCh <- err
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

// splitMessagesOnMarker is a split function for a Scanner that returns each msg
// in a byte stream stripped of any trailing msgMarker. The returned byte-stream
// may be empty. The last non-empty byte-slice of input will be returned even if
// it has no marker.
func splitMessagesOnMarker(marker []byte) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if len(marker) > 0 {
			if i := bytes.Index(data, marker); i >= 0 {
				// We have a full msg.
				return i + len(marker), data[0:i], nil
			}
		}
		// If we're at EOF, we have a final msg (without marker). Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	}
}
