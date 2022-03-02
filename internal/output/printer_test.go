package output

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrint(t *testing.T) {
	currentTime := time.Unix(42, 123)

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
			expected: "Topic: my-topic, Key: my-key, \n\rMsg: my-val, \tTimestamp: 42000, Time: 1970-01-01T01:00:42+01:00\n",
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
