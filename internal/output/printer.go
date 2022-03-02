package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Printer is the interface that knows how to print different results of a kafka consumer.
type Printer interface {
	Print(Msg)
	PrintErr(error)
}

// Msg is the successfully consumed Kafka message with some metadata for it.
type Msg struct {
	Key, Value string
	Topic      string
	Time       time.Time
}

// FormattedPrinter is a printer that knows how to parse the Kafkacat's format spec.
type FormattedPrinter struct {
	format      string
	out, errOut io.Writer
}

// NewFormatterPrinter returns a new instance of a printer that supports formatting similar to kafkacat's.
// Format tokens:
// 	%s		Message payload
//	%k		Message key
//	%t		Topic
//	%T		Timestamp in milliseconds
//	%Tf		Timestamp formatted as RFC3339
//  \n \r 	Newlines
// 	\t		Tab
func NewFormatterPrinter(format string, out, errOut io.Writer) *FormattedPrinter {
	return &FormattedPrinter{
		format: format,
		out:    out,
		errOut: errOut,
	}
}

// Print applies a specific format to a consumed Kafka message.
func (f *FormattedPrinter) Print(msg Msg) {
	val := f.format

	val = strings.ReplaceAll(val, "\\t", "\t")
	val = strings.ReplaceAll(val, "\\n", "\n")
	val = strings.ReplaceAll(val, "\\r", "\r")
	val = strings.ReplaceAll(val, "%s", msg.Value)
	val = strings.ReplaceAll(val, "%k", msg.Key)
	val = strings.ReplaceAll(val, "%t", msg.Topic)
	val = strings.ReplaceAll(val, "%Tf", msg.Time.Format(time.RFC3339))
	val = strings.ReplaceAll(val, "%T", strconv.Itoa(int(msg.Time.UnixMilli())))

	_, _ = fmt.Fprintln(f.out, val)
}

// PrintErr knows how to print an error.
func (f *FormattedPrinter) PrintErr(err error) {
	_, _ = fmt.Fprintln(f.errOut, err)
}
