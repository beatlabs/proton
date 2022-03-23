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

// ConsumeCfg is the config for everything this tool needs.
type ConsumeCfg struct {
	consumerCfg *consumer.Cfg
	model       string
	format      string
}

var consumeCfg = &ConsumeCfg{
	consumerCfg: &consumer.Cfg{},
}

func init() {
	rootCmd.AddCommand(consumeCmd)

	consumeCmd.Flags().StringVarP(&consumeCfg.consumerCfg.URL, "broker", "b", "", "Broker URL to consume from")
	if consumeCmd.MarkFlagRequired("broker") != nil {
		log.Fatal("you must specify a a broker URL using the `-b <url>` option")
	}

	consumeCmd.Flags().StringVarP(&consumeCfg.consumerCfg.Topic, "topic", "t", "", "A topic to consume from")
	if consumeCmd.MarkFlagRequired("topic") != nil {
		log.Fatal("you must specify a topic to consume using the `-t <topic>` option")
	}

	consumeCmd.Flags().StringVarP(&consumeCfg.model, "proto", "", "", "A path to a proto file an URL to it")
	if consumeCmd.MarkFlagRequired("proto") != nil {
		log.Fatal("you must specify a proto file using the `-m <path>` option")
	}

	consumeCmd.Flags().StringVarP(&consumeCfg.format, "format", "f", "%Tf: %s", `
A Kcat-like format string. Defaults to "%T: %s".
Format string tokens:
	%s                 Message payload
	%k                 Message key
	%t                 Topic
	%p                 Partition
	%o                 Offset
	%T                 Message timestamp (milliseconds since epoch UTC)
	%Tf                Message time formatted as RFC3339
	\n \r \t           Newlines, tab	
Example:
	-f 'Key: %k, Time: %Tf \nValue: %s'`)

	// FIXME: kafkacat's syntax allows specifying offsets using an array of `-o` flags.
	// Specifying `-o 123 -o 234` doesn't work with Cobra for an unknown reason but it actually should.
	// So before it's fixed, using the non-conventional `-s 123456789` and `-e 234567890`. It should be `-o s@123456789 -o e@234567890` instead.
	consumeCmd.Flags().Int64VarP(&consumeCfg.consumerCfg.Start, "start", "s", sarama.OffsetOldest, "Start timestamp offset")
	consumeCmd.Flags().Int64VarP(&consumeCfg.consumerCfg.End, "end", "e", sarama.OffsetNewest, "End timestamp offset")

	consumeCmd.Flags().StringVarP(&consumeCfg.consumerCfg.KeyGrep, "key", "", ".*", "Grep RegExp for a key value")

	consumeCmd.Flags().BoolVarP(&consumeCfg.consumerCfg.Verbose, "verbose", "v", false, "Whether to print out proton's debug messages")
}

// Run runs this whole thing.
func Run(cmd *cobra.Command, _ []string) {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	protoParser, fileName, err := protoparser.New(ctx, consumeCfg.model)
	if err != nil {
		log.Fatal(err)
	}

	kafka, err := consumer.NewKafka(ctx, consumer.Cfg{
		URL:     consumeCfg.consumerCfg.URL,
		Topic:   consumeCfg.consumerCfg.Topic,
		Start:   consumeCfg.consumerCfg.Start,
		End:     consumeCfg.consumerCfg.End,
		Verbose: consumeCfg.consumerCfg.Verbose,
		KeyGrep: consumeCfg.consumerCfg.KeyGrep,
	}, &protoDecoder{json.Converter{
		Parser:   protoParser,
		Filename: fileName,
	}}, output.NewFormatterPrinter(consumeCfg.format, os.Stdout, os.Stderr))

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
