package protoparser

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	fp "path/filepath"

	"github.com/jhump/protoreflect/desc/protoparse"
)

// NewFile initializes a proto parser from a local proto file.
func NewFile(filePath string) (protoparse.Parser, string, error) {
	abs, err := fp.Abs(fp.Clean(filePath))
	if err != nil {
		return protoparse.Parser{}, "", err
	}

	dir, fileName := fp.Split(abs)
	parser := protoparse.Parser{ImportPaths: []string{dir}}

	return parser, fileName, nil
}

// NewHTTP initializes a proto parser from a remote proto file.
func NewHTTP(ctx context.Context, fileURL *url.URL) (protoparse.Parser, string, error) {
	req, err := http.NewRequest("GET", fileURL.String(), nil)
	if err != nil {
		return protoparse.Parser{}, "", err
	}

	req = req.WithContext(ctx)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return protoparse.Parser{}, "", err
	}

	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return protoparse.Parser{}, "", fmt.Errorf("status code is %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return protoparse.Parser{}, "", err
	}

	_, fileName := fp.Split(fileURL.Path)

	accessor := protoparse.FileContentsFromMap(map[string]string{fileName: string(bodyBytes)})
	parser := protoparse.Parser{Accessor: accessor}

	return parser, fileName, nil
}
