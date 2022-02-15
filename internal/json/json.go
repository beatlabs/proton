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
	StartOfMessageMarker []byte
	EndOfMessageMarker   []byte
}

// ConvertStream converts multiple proto messages to json.
// It returns a result channel and error channel which both can return multiple messages (a result or error for each message)
// Because proto messages often contain newlines, we can't rely on new lines for knowing when one message ends and the
// next begins, so instead it looks for specified markers of the start and the end.
// Although unlikely, it is possible that the one the markers can be part of the proto binary message, in which case the
// parsing of that message will fail. If this happens, use more complex markers.
func (c Converter) ConvertStream(r io.Reader) (resultCh chan string, errorCh chan error) {
	resultCh = make(chan string)
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
		scanner.Split(splitMessagesOnMarkers(c.StartOfMessageMarker, c.EndOfMessageMarker))
		for scanner.Scan() {
			rawBytes := scanner.Bytes()
			parsed, err := c.unmarshalProtoBytesToJSON(md, rawBytes)
			if err != nil {
				// can't parse it, just output whatever that is
				resultCh <- string(rawBytes)
			} else {
				// could parse it, proto message, yahoo!
				resultCh <- string(parsed)
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

func (c Converter) unmarshalProtoBytesToJSON(md *desc.MessageDescriptor, rawMessage []byte) ([]byte, error) {
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

// splitMessagesOnMarkers is a split function for a Scanner that returns each msg
// in a byte stream stripped of any leading or trailing msgMarker(s).
// The data between two markers is seen as a chunk of binary data
// The data outside of START and END is seen as some random data and also just returned.
// The returned byte-stream may be empty. The last non-empty byte-slice of input will be returned even if
// it has no endMarker.
func splitMessagesOnMarkers(startMarker, endMarker []byte) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if isDataEmpty(data, atEOF) {
			return 0, nil, nil
		}

		if markersDefined(startMarker, endMarker) {
			startMarkerLength := len(startMarker)
			endMarkerLength := len(endMarker)

			// example message:
			// Hello world --START-- binary awesomeness --END-- Bye world

			startI := bytes.Index(data, startMarker)
			// if START is not at index 0, then there is some data before binary message
			if startI > 0 {
				// we just return that data
				// example: Hello world
				return startI, data[0:startI], nil
			}

			// if START is at index 0, then this is a start of a binary message
			// example: --START-- binary awesomeness --END-- Bye world
			if startI == 0 {
				// find END marker
				endI := bytes.Index(data, endMarker)

				// if it's found, return everything in between, this is a complete binary message
				if endI >= 0 {
					// advance will skip the data until the end of END marker for the next chunk
					advance := endI + endMarkerLength
					message := data[startMarkerLength:endI]

					// example: binary awesomeness
					return advance, message, nil
				}
			}

			// if we can't find a START, then there are no more binary messages
			if startI == -1 {
				// so simply return what we have left
				return len(data), data, nil
			}
		}

		// If we're at EOF, we have a final msg (without marker). Return it.
		if atEOF {
			return len(data), data, nil
		}

		// if we can't find either of START or END markers, we need more data, we're somewhere in between
		return 0, nil, nil
	}
}

func markersDefined(startMarker []byte, endMarker []byte) bool {
	return len(startMarker) > 0 && len(endMarker) > 0
}

func isDataEmpty(data []byte, atEOF bool) bool {
	return atEOF && len(data) == 0
}
