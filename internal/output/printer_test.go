package output

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrint(t *testing.T) {
	currentTime := time.Unix(42, 123)
	timeFormatted := currentTime.Format(time.RFC3339)

	tests := []struct {
		name     string
		format   string
		msg      Msg
		expected string
	}{
		{
			name:   "use all the possible formatting tokens",
			format: "Topic: %t, Key: %k, \\n\\rMsg: %s, \\tTimestamp: %T, Time: %Tf",
			msg: Msg{
				Key:   "my-key",
				Value: "my-val",
				Topic: "my-topic",
				Time:  currentTime,
			},
			expected: fmt.Sprintf("Topic: my-topic, Key: my-key, \n\rMsg: my-val, \tTimestamp: 42000, Time: %s\n", timeFormatted),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			buffer := bytes.NewBufferString("")
			bufferErr := bytes.NewBufferString("")
			printer := NewFormatterPrinter(test.format, buffer, bufferErr)

			// when
			printer.Print(test.msg)

			// then
			assert.Equal(t, test.expected, buffer.String())
			assert.Empty(t, bufferErr.String())
		})
	}
}

func TestPrintErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "prints err",
			err:      errors.New("b00m"),
			expected: "b00m\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			buffer := bytes.NewBufferString("")
			bufferErr := bytes.NewBufferString("")
			printer := NewFormatterPrinter("irrelevant", buffer, bufferErr)

			// when
			printer.PrintErr(test.err)

			// then
			assert.Equal(t, test.expected, bufferErr.String())
			assert.Empty(t, buffer.String())
		})
	}
}
