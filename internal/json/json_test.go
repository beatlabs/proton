package json

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/beatlabs/proton/internal/protoparser"
	another_tutorial "github.com/beatlabs/proton/testdata"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	json "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Test_Converter(t *testing.T) {
	addressBook := genAddressBook()

	tests := []struct {
		name      string
		converter func() Converter
		message   func() io.Reader
		assert    func([]byte, error)
	}{
		{
			name: "Wrong package",
			converter: func() Converter {
				parser, filename, _ := protoparser.NewFile("../../testdata/addressbook.proto")

				return Converter{
					Parser:      parser,
					Filename:    filename,
					Package:     "tutorial2",
					MessageType: "AddressBook",
				}
			},
			message: func() io.Reader {
				return nil
			},
			assert: func(bytes []byte, err error) {
				assert.Empty(t, bytes)
				assert.Error(t, err)
				assert.EqualError(t, err, "can't find AddressBook in tutorial2 package")
			},
		},
		{
			name: "Wrong type",
			converter: func() Converter {
				parser, filename, _ := protoparser.NewFile("../../testdata/addressbook.proto")

				return Converter{
					Parser:      parser,
					Filename:    filename,
					Package:     "tutorial",
					MessageType: "AddressBook2",
				}
			},
			message: func() io.Reader {
				return nil
			},
			assert: func(bytes []byte, err error) {
				assert.Empty(t, bytes)
				assert.Error(t, err)
				assert.EqualError(t, err, "can't find AddressBook2 in tutorial package")
			},
		},
		{
			name: "No package provided, defaults to package of proto file",
			converter: func() Converter {
				parser, filename, _ := protoparser.NewFile("../../testdata/addressbook.proto")
				return Converter{
					Parser:      parser,
					Filename:    filename,
					MessageType: "AddressBook",
				}
			},
			message: func() io.Reader {
				protoBytes, err := proto.Marshal(addressBook)
				assert.NoError(t, err)
				return bytes.NewReader(protoBytes)
			},
			assert: func(bytes []byte, err error) {
				assert.NotEmpty(t, bytes)
				assert.NoError(t, err)
				addressBookAsByte, err := json.Marshal(addressBook)
				assert.NoError(t, err)
				assert.JSONEq(t, string(addressBookAsByte), string(bytes))
			},
		},
		{
			name: "No message type provided, defaults to first message type",
			converter: func() Converter {
				parser, filename, _ := protoparser.NewFile("../../testdata/addressbook.proto")
				return Converter{
					Parser:   parser,
					Filename: filename,
					Package:  "tutorial",
				}
			},
			message: func() io.Reader {
				protoBytes, err := proto.Marshal(addressBook)
				assert.NoError(t, err)
				return bytes.NewReader(protoBytes)
			},
			assert: func(bytes []byte, err error) {
				assert.NotEmpty(t, bytes)
				assert.NoError(t, err)
				addressBookAsByte, err := json.Marshal(addressBook)
				assert.NoError(t, err)
				assert.JSONEq(t, string(addressBookAsByte), string(bytes))
			},
		},
		{
			name: "Parse message",
			converter: func() Converter {
				parser, filename, err := protoparser.NewFile("../../testdata/addressbook.proto")
				assert.NoError(t, err)
				return Converter{
					Parser:      parser,
					Filename:    filename,
					Package:     "tutorial",
					MessageType: "AddressBook",
				}
			},
			message: func() io.Reader {
				protoBytes, err := proto.Marshal(addressBook)
				assert.NoError(t, err)
				return bytes.NewReader(protoBytes)
			},
			assert: func(bytes []byte, err error) {
				assert.NotEmpty(t, bytes)
				assert.NoError(t, err)
				addressBookAsByte, err := json.Marshal(addressBook)
				assert.NoError(t, err)
				assert.JSONEq(t, string(addressBookAsByte), string(bytes))
			},
		},
		{
			name: "Parse message with indent",
			converter: func() Converter {
				parser, filename, err := protoparser.NewFile("../../testdata/addressbook.proto")
				assert.NoError(t, err)
				return Converter{
					Parser:      parser,
					Filename:    filename,
					Package:     "tutorial",
					MessageType: "AddressBook",
					Indent:      true,
				}
			},
			message: func() io.Reader {
				protoBytes, err := proto.Marshal(addressBook)
				assert.NoError(t, err)
				return bytes.NewReader(protoBytes)
			},
			assert: func(bytes []byte, err error) {
				assert.NotEmpty(t, bytes)
				assert.NoError(t, err)
				marshaller := json.MarshalOptions{Indent: " "}
				addressBookAsByte, err := marshaller.Marshal(addressBook)
				assert.NoError(t, err)
				assert.JSONEq(t, string(addressBookAsByte), string(bytes))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := test.converter()
			b, err := c.Convert(test.message())
			test.assert(b, err)
		})
	}
}

func Test_ConvertStream(t *testing.T) {
	addressBook := genAddressBook()
	protoBytes, err := proto.Marshal(addressBook)
	assert.NoError(t, err)

	addressBookAsJsonByte, err := json.MarshalOptions{}.Marshal(addressBook)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		input   func() *strings.Reader
		results [][]byte
		errors  []error
	}{
		{
			name: "three addressbook messages",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.WriteString(string(protoBytes))
				b.WriteString(DefaultEndOfMessageMarker)
				b.WriteString(string(protoBytes))
				b.WriteString(DefaultEndOfMessageMarker)
				b.WriteString(string(protoBytes))
				b.WriteString(DefaultEndOfMessageMarker)
				return strings.NewReader(b.String())
			},
			results: [][]byte{
				addressBookAsJsonByte,
				addressBookAsJsonByte,
				addressBookAsJsonByte,
			},
		},
		{
			name: "addressbook messages without last marker",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.WriteString(string(protoBytes))
				b.WriteString(DefaultEndOfMessageMarker)
				b.WriteString(string(protoBytes))
				return strings.NewReader(b.String())
			},
			results: [][]byte{
				addressBookAsJsonByte,
				addressBookAsJsonByte,
			},
		},
		{
			name: "single addressbook message",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.WriteString(string(protoBytes))
				return strings.NewReader(b.String())
			},
			results: [][]byte{
				addressBookAsJsonByte,
			},
		},
		{
			name: "invalid first message doesn't stop processing",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.WriteString("\n")
				b.WriteString(DefaultEndOfMessageMarker)
				b.WriteString(string(protoBytes))
				b.WriteString(DefaultEndOfMessageMarker)
				b.WriteString(string(protoBytes))
				return strings.NewReader(b.String())
			},
			results: [][]byte{
				addressBookAsJsonByte,
				addressBookAsJsonByte,
			},
			errors: []error{
				errors.New("unexpected EOF"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser, filename, err := protoparser.NewFile("../../testdata/addressbook.proto")
			assert.NoError(t, err)
			c := Converter{
				Parser:   parser,
				Filename: filename,
			}
			resultCh, errorCh := c.ConvertStream(test.input())

			var results []string
			var errors []error
			for {
				done := false
				select {
				case m, ok := <-resultCh:
					if !ok {
						done = true
						break
					}
					results = append(results, string(m))
				case e, ok := <-errorCh:
					if !ok {
						done = true
						break
					}
					errors = append(errors, e)
				}
				if done {
					break
				}
			}

			assert.Equal(t, len(test.results), len(results))
			for i, r := range results {
				assert.JSONEq(t, string(test.results[i]), r)
			}
			assert.Equal(t, len(test.errors), len(errors))
			for i, e := range errors {
				assert.EqualError(t, e, test.errors[i].Error())
			}
		})
	}
}

func Test_ConvertStream_WithInvalidProtoFile(t *testing.T) {
	parser, filename, err := protoparser.NewFile("../../testdata/not-a-file.proto")
	assert.NoError(t, err)
	c := Converter{
		Parser:   parser,
		Filename: filename,
	}

	resultCh, errorCh := c.ConvertStream(strings.NewReader(""))
	err = <-errorCh
	res := <-resultCh
	assert.Error(t, err)
	assert.Nil(t, res)

}

func TestSplitting(t *testing.T) {
	tests := map[string]struct {
		input        []byte
		marker       []byte
		expectedMsgs [][]byte
	}{
		"empty": {
			input:        []byte{},
			marker:       []byte{},
			expectedMsgs: [][]byte{},
		},
		"empty marker returns original input": {
			input:  appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte("#marker#"), []byte{4, 5, '\n', 6, 7, 'p'}),
			marker: []byte{},
			expectedMsgs: [][]byte{
				appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte("#marker#"), []byte{4, 5, '\n', 6, 7, 'p'}),
			},
		},
		"missing end marker": {
			input:  appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte("#marker#"), []byte{4, 5, '\n', 6, 7, 'p'}),
			marker: []byte("#marker#"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r'}, //msg1
				{4, 5, '\n', 6, 7, 'p'},  //msg2
			},
		},
		"with end marker": {
			input:  appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte("#marker#"), []byte{4, 5, '\n', 6, 7, 'p'}, []byte("#marker#")),
			marker: []byte("#marker#"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r'},
				{4, 5, '\n', 6, 7, 'p'},
			},
		},
		"with incomplete marker": {
			input:  appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte("#marker#"), []byte{4, 5, '\n', 6, 7, 'p'}, []byte("#marker")),
			marker: []byte("#marker#"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r'},
				appendSlices([]byte{4, 5, '\n', 6, 7, 'p'}, []byte("#marker")),
			},
		},
		"three msgs ": {
			input: appendSlices(
				[]byte{1, 2, '\n', 3, 4, '\r'},     //msg1
				[]byte("--END--"),                  // marker
				[]byte{4, 5, '\n', 7, 'p'},         //msg2
				[]byte("--END--"),                  // marker
				[]byte{4, 5, '\n', 6, 7, 8, 9, 10}, //msg2
				[]byte("--END--"),                  // marker
			), //marker
			marker: []byte("--END--"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r'},
				{4, 5, '\n', 7, 'p'},
				{4, 5, '\n', 6, 7, 8, 9, 10},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(tt.input))
			s := bufio.NewScanner(r)
			s.Split(splitMessagesOnMarker(tt.marker))
			msgs := make([][]byte, 0)
			for s.Scan() {
				msgs = append(msgs, s.Bytes())
			}
			if err := s.Err(); err != nil {
				t.Errorf("Scan() error = %v", err)
			}
			if !reflect.DeepEqual(msgs, tt.expectedMsgs) {
				t.Errorf("Scan() = %v, want %v", msgs, tt.expectedMsgs)
			}
		})
	}
}

func genAddressBook() *another_tutorial.AddressBook {
	loc, _ := time.LoadLocation("UTC")
	d := time.Date(2013, 1, 2, 9, 22, 0, 0, loc)
	d2 := d.Add(27 * 24 * time.Hour).Add(23 * time.Minute)

	return &another_tutorial.AddressBook{
		People: []*another_tutorial.Person{{
			Name:  "ABC",
			Id:    1,
			Email: "abc@thebeat.co",
			Phones: []*another_tutorial.Person_PhoneNumber{{
				Number: "123456",
				Type:   another_tutorial.Person_HOME,
			}},
			LastUpdated: &timestamp.Timestamp{
				Seconds: d.Unix(),
			},
		}, {
			Name:  "DEF",
			Id:    2,
			Email: "def@thebeat.co",
			Phones: []*another_tutorial.Person_PhoneNumber{{
				Number: "789012",
				Type:   another_tutorial.Person_HOME,
			}},
			LastUpdated: &timestamp.Timestamp{
				Seconds: d2.Unix(),
			},
		}},
	}
}

func appendSlices(ss ...[]byte) []byte {
	res := []byte{}
	for _, s := range ss {
		res = append(res, s...)
	}
	return res
}
