package json

import (
	"bytes"
	"io"
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
	loc, _ := time.LoadLocation("UTC")
	d := time.Date(2013, 1, 2, 9, 22, 0, 0, loc)
	d2 := d.Add(27 * 24 * time.Hour).Add(23 * time.Minute)

	addressBook := &another_tutorial.AddressBook{
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
