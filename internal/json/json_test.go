package json

import (
	"bufio"
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/beatlabs/proton/v2/internal/protoparser"
	another_tutorial "github.com/beatlabs/proton/v2/testdata"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	json "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var startMarker = []byte("--START--")
var endMarker = []byte("--END--")

func Test_ConvertStream(t *testing.T) {
	addressBook := genAddressBook()
	protoBytes, err := proto.Marshal(addressBook)
	assert.NoError(t, err)

	addressBookProtoParser, addressBookFilename, err := protoparser.NewFile("../../testdata/addressbook.proto")
	assert.NoError(t, err)

	defaultConverter := &Converter{
		Parser:               addressBookProtoParser,
		Filename:             addressBookFilename,
		StartOfMessageMarker: startMarker,
		EndOfMessageMarker:   endMarker,
	}

	addressBookAsJSONBytes, err := json.MarshalOptions{}.Marshal(addressBook)
	assert.NoError(t, err)
	addressBookAsIndentedJSONBytes, err := json.MarshalOptions{Indent: "	"}.Marshal(addressBook)
	assert.NoError(t, err)

	type result struct {
		isJSON bool
		val    []byte
	}
	tests := []struct {
		name      string
		input     func() *strings.Reader
		converter *Converter
		results   []result
		errors    []error
	}{
		{
			name: "Wrong package",
			converter: &Converter{
				Parser:      addressBookProtoParser,
				Filename:    addressBookFilename,
				Package:     "tutorial2",
				MessageType: "AddressBook",
			},
			input: func() *strings.Reader {
				return strings.NewReader("")
			},
			errors: []error{
				errors.New("can't find AddressBook in tutorial2 package"),
			},
		},
		{
			name: "Wrong type",
			converter: &Converter{
				Parser:      addressBookProtoParser,
				Filename:    addressBookFilename,
				Package:     "tutorial",
				MessageType: "AddressBook2",
			},
			input: func() *strings.Reader {
				return strings.NewReader("")
			},
			errors: []error{
				errors.New("can't find AddressBook2 in tutorial package"),
			},
		},
		{
			name: "No package provided, defaults to package of proto file",
			converter: &Converter{
				Parser:      addressBookProtoParser,
				Filename:    addressBookFilename,
				MessageType: "AddressBook",
			},
			input: func() *strings.Reader {
				return strings.NewReader(string(protoBytes))
			},
			results: []result{{
				isJSON: true,
				val:    addressBookAsJSONBytes,
			}},
		},
		{
			name: "No message type provided, defaults to first message type",
			converter: &Converter{
				Parser:   addressBookProtoParser,
				Filename: addressBookFilename,
				Package:  "tutorial",
			},
			input: func() *strings.Reader {
				return strings.NewReader(string(protoBytes))
			},
			results: []result{{
				isJSON: true,
				val:    addressBookAsJSONBytes,
			}},
		},
		{
			name: "three addressbook messages",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				return strings.NewReader(b.String())
			},
			results: []result{{
				isJSON: true,
				val:    addressBookAsJSONBytes,
			}, {
				isJSON: true,
				val:    addressBookAsJSONBytes,
			}, {
				isJSON: true,
				val:    addressBookAsJSONBytes,
			}},
		},
		{
			name: "three addressbook messages with timestamp and other data",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write([]byte("Hello world"))

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				b.Write([]byte("Timestamp: 1234567890"))

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				// nothing in-between messages

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				b.Write([]byte("Bye world"))

				return strings.NewReader(b.String())
			},
			results: []result{
				{val: []byte("Hello world")},
				{isJSON: true, val: addressBookAsJSONBytes},
				{val: []byte("Timestamp: 1234567890")},
				{isJSON: true, val: addressBookAsJSONBytes},
				{isJSON: true, val: addressBookAsJSONBytes},
				{val: []byte("Bye world")},
			},
		},
		{
			// kcat ... -f '{"key": "%k", "timestamp": %T, "value":--START--%s--END--}'
			name: "consumes json-formatted input",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write([]byte(`{"key": "gr_12345", "timestamp": 1644882297612, "value":`))
				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)
				b.Write([]byte("}"))

				return strings.NewReader(b.String())
			},
			results: []result{
				{val: []byte(`{"key": "gr_12345", "timestamp": 1644882297612, "value":`)},
				{isJSON: true, val: addressBookAsJSONBytes},
				{val: []byte("}")},
			},
		},
		{
			name: "single addressbook message",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write(protoBytes)
				return strings.NewReader(b.String())
			},
			results: []result{
				{isJSON: true, val: addressBookAsJSONBytes},
			},
		},
		{
			name: "single addressbook message with indent",
			converter: &Converter{
				Parser:   addressBookProtoParser,
				Filename: addressBookFilename,
				Indent:   true,
			},
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write(protoBytes)
				return strings.NewReader(b.String())
			},
			results: []result{
				{isJSON: true, val: addressBookAsIndentedJSONBytes},
			},
		},
		{
			name: "invalid first message doesn't stop processing",
			input: func() *strings.Reader {
				var b bytes.Buffer
				b.Write(startMarker)
				b.Write([]byte("qwe"))
				b.Write(endMarker)

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				b.Write(startMarker)
				b.Write(protoBytes)
				b.Write(endMarker)

				return strings.NewReader(b.String())
			},
			results: []result{
				{val: []byte("qwe")},
				{isJSON: true, val: addressBookAsJSONBytes},
				{isJSON: true, val: addressBookAsJSONBytes},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			c := defaultConverter
			if test.converter != nil {
				c = test.converter
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
			for i, expected := range test.results {
				if expected.isJSON {
					assert.JSONEq(t, string(expected.val), results[i])
				} else {
					assert.Equal(t, string(expected.val), results[i])
				}
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
	assert.Empty(t, res)

}

func TestSplitting(t *testing.T) {
	tests := map[string]struct {
		input        []byte
		startMarker  []byte
		endMarker    []byte
		expectedMsgs [][]byte
	}{
		"empty": {
			input:        []byte{},
			endMarker:    []byte{},
			expectedMsgs: [][]byte{},
		},
		"without markers -> return original input": {
			input:       appendSlices([]byte{1, 2, '\n', 3, 4, '\r'}, []byte{4, 5, '\n', 6, 7, 'p'}),
			startMarker: []byte("#startMarker#"),
			endMarker:   []byte("#endMarker#"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r', 4, 5, '\n', 6, 7, 'p'},
			},
		},
		"with markers -> split": {
			input: appendSlices(
				[]byte("#startMarker#"),
				[]byte{1, 2, '\n', 3, 4, '\r'},
				[]byte("#endMarker#"),

				[]byte("#startMarker#"),
				[]byte{4, 5, '\n', 6, 7, 'p'},
				[]byte("#endMarker#")),
			startMarker: []byte("#startMarker#"),
			endMarker:   []byte("#endMarker#"),
			expectedMsgs: [][]byte{
				{1, 2, '\n', 3, 4, '\r'},
				{4, 5, '\n', 6, 7, 'p'},
			},
		},
		"three msgs with markers and random data": {
			input: appendSlices(
				[]byte("Hello world"),

				[]byte("--START--"),
				[]byte{1, 2, '\n', 3, 4, '\r'}, //msg1
				[]byte("--END--"),

				[]byte("Timestamp: 1234567890"),

				[]byte("--START--"),
				[]byte{4, 5, '\n', 7, 'p'}, //msg2
				[]byte("--END--"),

				// no extra data

				[]byte("--START--"),
				[]byte{4, 5, '\n', 6, 7, 8, 9, 10}, //msg3
				[]byte("--END--"),

				[]byte("Bye world"),
			),
			startMarker: []byte("--START--"),
			endMarker:   []byte("--END--"),
			expectedMsgs: [][]byte{
				[]byte("Hello world"),
				{1, 2, '\n', 3, 4, '\r'},
				[]byte("Timestamp: 1234567890"),
				{4, 5, '\n', 7, 'p'},
				{4, 5, '\n', 6, 7, 8, 9, 10},
				[]byte("Bye world"),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(tt.input))
			s := bufio.NewScanner(r)
			s.Split(splitMessagesOnMarkers(tt.startMarker, tt.endMarker))
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
	var res []byte
	for _, s := range ss {
		res = append(res, s...)
	}
	return res
}
