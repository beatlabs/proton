package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/beatlabs/proton/internal/json"
	"github.com/beatlabs/proton/internal/protoparser"
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
			Parser:      protoParser,
			Filename:    fileName,
			Package:     pkg,
			MessageType: messageType,
			Indent:      indent,
		}

		var r io.Reader

		if isInputFromPipe() {
			r = os.Stdin
		} else {
			if len(args) != 1 {
				return errors.New("input file path is empty")
			}

			file, err := getFile(args[0])
			if err != nil {
				return err
			}

			defer file.Close()

			r = file
		}

		convert, err := c.Convert(r)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(os.Stdout, string(convert))

		return err
	},
}

var indent bool
var file string
var pkg string
var messageType string

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
