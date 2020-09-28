package protoparser

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func Test_FileParser(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		assert func(protoparse.Parser, string, error)
	}{
		{
			name: "File doesn't exist",
			path: "../../testdata/abcd.proto",
			assert: func(parser protoparse.Parser, fileName string, err error) {
				assert.NotEqual(t, protoparse.Parser{}, parser)
				assert.Equal(t, "abcd.proto", fileName)
				assert.NoError(t, err)
				_, parseErr := parser.ParseFiles(fileName)
				assert.Error(t, parseErr)
				assert.Contains(t, parseErr.Error(), "no such file or directory")
			},
		},
		{
			name: "File exist",
			path: "../../testdata/addressbook.proto",
			assert: func(parser protoparse.Parser, fileName string, err error) {
				assert.NotEqual(t, protoparse.Parser{}, parser)
				assert.Equal(t, "addressbook.proto", fileName)
				assert.NoError(t, err)
				_, parseErr := parser.ParseFiles(fileName)
				assert.NoError(t, parseErr)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser, filename, err := NewFile(test.path)
			test.assert(parser, filename, err)
		})
	}
}

func Test_HTTPParser(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		prepare func()
		assert  func(protoparse.Parser, string, error)
	}{
		{
			name: "Not found",
			url:  "http://protoregistry.com/testdata/abcd.proto",
			prepare: func() {
				gock.New("http://protoregistry.com").
					Get("/testdata/abcd.proto").
					Reply(http.StatusNotFound)
			},
			assert: func(parser protoparse.Parser, fileName string, err error) {
				assert.Equal(t, protoparse.Parser{}, parser)
				assert.Empty(t, fileName)
				assert.Error(t, err)
				assert.EqualError(t, err, "status code is 404")
			},
		},
		{
			name: "Internal error",
			url:  "http://protoregistry.com/testdata/abcde.proto",
			prepare: func() {
				gock.New("http://protoregistry.com").
					Get("/testdata/abcde.proto").
					Reply(http.StatusInternalServerError)
			},
			assert: func(parser protoparse.Parser, fileName string, err error) {
				assert.Equal(t, protoparse.Parser{}, parser)
				assert.Empty(t, fileName)
				assert.Error(t, err)
				assert.EqualError(t, err, "status code is 500")

			},
		},
		{
			name: "File found",
			url:  "http://protoregistry.com/testdata/addressbook.proto",
			prepare: func() {
				abs, _ := filepath.Abs("../../testdata/addressbook.proto")

				gock.New("http://protoregistry.com").
					Get("/testdata/addressbook.proto").
					Reply(http.StatusOK).
					File(abs)
			},
			assert: func(parser protoparse.Parser, fileName string, err error) {
				assert.NotEqual(t, protoparse.Parser{}, parser)
				assert.Equal(t, "addressbook.proto", fileName)
				assert.NoError(t, err)
				_, parseErr := parser.ParseFiles(fileName)
				assert.NoError(t, parseErr)
			},
		},
	}

	defer gock.Off()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.prepare()
			parse, err := url.Parse(test.url)
			assert.NoError(t, err)
			parser, filename, err := NewHTTP(context.TODO(), parse)
			test.assert(parser, filename, err)
		})
	}

	assert.True(t, gock.IsDone())
}
