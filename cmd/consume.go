package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

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
	consumerCfg consumer.Cfg
	offsets     []string
	model       string
	format      string
}

var consumeCfg = &ConsumeCfg{
	consumerCfg: consumer.Cfg{},
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

	consumeCmd.Flags().StringSliceVarP(&consumeCfg.offsets, "offsets", "o", []string{}, `
Offset to start consuming from
	 s@<value> (timestamp in ms to start at)
	 e@<value> (timestamp in ms to stop at (not included))
`)

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

	consumeCfg.consumerCfg.Start, consumeCfg.consumerCfg.End = parseOffsets(consumeCfg.offsets)

	kafka, err := consumer.NewKafka(ctx, consumeCfg.consumerCfg,
		&protoDecoder{json.Converter{
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

func parseOffsets(offsets []string) (int64, int64) {
	return parseOffset("s@", offsets, sarama.OffsetOldest), parseOffset("e@", offsets, sarama.OffsetNewest)
}

func parseOffset(prefix string, offsets []string, defaultVal int64) int64 {
	for _, offset := range offsets {
		if strings.HasPrefix(offset, prefix) {
			v, err := strconv.Atoi(offset[len(prefix):])
			if err == nil {
				return int64(v)
			}
		}
	}
	return defaultVal
}
