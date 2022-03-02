package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/proton/v2/internal/consumer"
	"github.com/beatlabs/proton/v2/internal/json"
	"github.com/beatlabs/proton/v2/internal/output"
	"github.com/beatlabs/proton/v2/internal/protoparser"
	"github.com/spf13/cobra"
)

// consumeCmd represents the consume command
var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "consume from given topics",
	Run:   Run,
}

var topic string
var broker string
var model string
var format string
var startTime, endTime int64
var verbose bool

func init() {
	rootCmd.AddCommand(consumeCmd)

	consumeCmd.Flags().StringVarP(&broker, "broker", "b", "", "Broker URL to consume from")
	if consumeCmd.MarkFlagRequired("broker") != nil {
		log.Fatal("you must specify a a broker URL using the `-b <url>` option")
	}

	consumeCmd.Flags().StringVarP(&topic, "topic", "t", "", "A topic to consume from")
	if consumeCmd.MarkFlagRequired("topic") != nil {
		log.Fatal("you must specify a topic to consume using the `-t <topic>` option")
	}

	consumeCmd.Flags().StringVarP(&model, "model", "m", "", "A path to a proto file an URL to it")
	if consumeCmd.MarkFlagRequired("model") != nil {
		log.Fatal("you must specify a proto file using the `-m <path>` option")
	}

	consumeCmd.Flags().StringVarP(&format, "format", "f", "%T: %s", `
A Kcat-like format string. Defaults to "%T: %s".
Format string tokens:
	%s                 Message payload
	%k                 Message key
	%t                 Topic
	%T                 Message timestamp (milliseconds since epoch UTC)
	%Tf                Message time formatted as RFC3339 # this is not supported by kcat
	\n \r \t           Newlines, tab
	
	// [not yet supported] \xXX \xNNN         Any ASCII character
	// [not yet supported] %S                 Message payload length (or -1 for NULL)
	// [not yet supported] %R                 Message payload length (or -1 for NULL) serialized as a binary big endian 32-bit signed integer
	// [not yet supported] %K                 Message key length (or -1 for NULL)
	// [not yet supported] %h                 Message headers (n=v CSV)	
	// [not yet supported] %p                 Partition
	// [not yet supported] %o                 Message offset	
Example:
	-f 'Key: %k, Time: %Tf \nValue: %s'`)

	consumeCmd.Flags().Int64VarP(&startTime, "start", "s", sarama.OffsetOldest, "Start offset timestamp milliseconds")
	consumeCmd.Flags().Int64VarP(&endTime, "end", "e", sarama.OffsetNewest, "End offset timestamp milliseconds")

	consumeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Whether to print out proton's debug messages")
}

// Run runs this whole thing.
func Run(cmd *cobra.Command, _ []string) {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	protoParser, fileName, err := protoparser.New(ctx, model)
	if err != nil {
		log.Fatal(err)
	}

	kafka, err := consumer.NewKafka(ctx, consumer.Cfg{
		URL:     broker,
		Topic:   topic,
		Start:   startTime,
		End:     endTime,
		Verbose: verbose,
	}, &protoDecoder{json.Converter{
		Parser:   protoParser,
		Filename: fileName,
	}}, output.NewFormatterPrinter(format, os.Stdout, os.Stderr))

	if err != nil {
		log.Fatal(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	errCh := kafka.Run()

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal(err)
		}
	case _ = <-signals:
		break
	}
}

type protoDecoder struct {
	json.Converter
}

// Decode uses the existing json decoder and adapts it to this consumer.
func (p *protoDecoder) Decode(rawData []byte) (string, error) {
	stream, errCh := p.ConvertStream(bytes.NewReader(rawData))
	select {
	case msg := <-stream:
		return string(msg), nil
	case err := <-errCh:
		return "", err
	}
}
