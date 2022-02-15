package cmd

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/beatlabs/proton/v2/internal/json"
	"github.com/beatlabs/proton/v2/internal/protoparser"
	"github.com/spf13/cobra"
)

// jsonCmd represents the json command
var jsonCmd = &cobra.Command{
	Use:   "json",
	Short: "pass protobuf message or pipe in binary format",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, err := url.Parse(file)
		if err != nil {
			return err
		}

		var protoParser json.ProtoParser
		fileName := ""
		if url.Scheme == "" {
			protoParser, fileName, err = protoparser.NewFile(url.String())
			if err != nil {
				return err
			}
		} else {
			protoParser, fileName, err = protoparser.NewHTTP(cmd.Context(), url)
			if err != nil {
				return err
			}
		}

		c := json.Converter{
			Parser:               protoParser,
			Filename:             fileName,
			Package:              pkg,
			MessageType:          messageType,
			Indent:               indent,
			StartOfMessageMarker: []byte(startOfMessageMarker),
			EndOfMessageMarker:   []byte(endOfMessageMarker),
		}

		r := os.Stdin
		if !isInputFromPipe() {
			if len(args) != 1 {
				return errors.New("input file path is empty")
			}

			r, err = getFile(args[0])
			if err != nil {
				return err
			}

			defer r.Close()
		}

		resultCh, errorCh := c.ConvertStream(r)
		var lastError error
		for {
			done := false
			select {
			case m, ok := <-resultCh:
				if !ok {
					done = true
					break
				}
				_, _ = fmt.Fprint(os.Stdout, m)
			case e, ok := <-errorCh:
				if !ok {
					done = true
					break
				}
				lastError = e
				_, _ = fmt.Fprintln(os.Stderr, e)
			}
			if done {
				break
			}
		}
		return lastError
	},
}

var indent bool
var file string
var pkg string
var messageType string
var startOfMessageMarker string
var endOfMessageMarker string

func init() {
	rootCmd.AddCommand(jsonCmd)

	jsonCmd.Flags().BoolVar(&indent, "indent", false, "Indent output json")
	jsonCmd.Flags().StringVarP(&file, "file", "f", "", "Proto file path or url")
	err := jsonCmd.MarkFlagRequired("file")
	if err != nil {
		log.Fatalf("Failed setting the 'file' flag to required")
	}
	jsonCmd.Flags().StringVarP(&pkg, "package", "p", "", "Proto package"+
		"\nDefaults to the package found in the Proton file if not specified")
	jsonCmd.Flags().StringVarP(&messageType, "type", "t", "", "Proto message type"+
		"\nDefaults to the first message type in the Proton file if not specified")
	jsonCmd.Flags().StringVarP(&startOfMessageMarker, "start-of-message-marker", "s", "",
		"\nMarker for the start of a message used when piping data, ignored if end marker is not specified")
	jsonCmd.Flags().StringVarP(&endOfMessageMarker, "end-of-message-marker", "m", "",
		"\nMarker for the end of a message used when piping data, ignored if start marker is not specified")
}

func isInputFromPipe() bool {
	fileInfo, _ := os.Stdin.Stat()

	return fileInfo.Mode()&os.ModeCharDevice == 0
}

func getFile(path string) (*os.File, error) {
	if path == "" {
		return nil, errors.New("input file path is empty")
	}

	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	if !fileExists(path) {
		return nil, errors.New("input file does not exist")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
