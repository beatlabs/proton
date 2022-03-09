package consumer

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/proton/v2/internal/output"
	"github.com/beatlabs/proton/v2/internal/protoparser"
)

const defaultPort = "9092"

// Cfg is the configuration of this consumer.
type Cfg struct {
	URL        string
	Topic      string
	Start, End int64
	Verbose    bool
	KeyGrep    string
}

// Kafka is the consumer itself.
type Kafka struct {
	ctx context.Context

	topic   string
	offsets []offsets

	keyGrep *regexp.Regexp
	verbose bool

	client sarama.Client

	decoder protoparser.Decoder
	printer output.Printer
}

type offsets struct {
	partition  int32
	start, end int64
}

// NewKafka returns a new instance of this consumer or an error if something isn't right.
func NewKafka(ctx context.Context, cfg Cfg, decoder protoparser.Decoder, printer output.Printer) (*Kafka, error) {
	config := sarama.NewConfig()
	config.ClientID = "proton-consumer"
	config.Consumer.Return.Errors = true
	config.Version = sarama.V0_11_0_0
	config.Consumer.IsolationLevel = sarama.ReadCommitted

	if cfg.Verbose {
		fmt.Println("Spinning the wheel... Connecting, gathering partitions data and stuff...")
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	broker := parsed.String()
	if parsed.Port() == "" {
		broker = fmt.Sprintf("%s:%s", broker, defaultPort)
	}

	client, err := sarama.NewClient([]string{broker}, config)
	if err != nil {
		return nil, err
	}

	var oo []offsets
	topic := cfg.Topic
	partitions, err := client.Partitions(topic)
	if err != nil {
		return nil, err
	}

	for _, p := range partitions {
		start := sarama.OffsetOldest
		if cfg.Start != sarama.OffsetOldest {
			start, err = client.GetOffset(topic, p, cfg.Start)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
		}

		end := sarama.OffsetNewest
		if cfg.End != sarama.OffsetNewest {
			end, err = client.GetOffset(topic, p, cfg.End)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
		}

		oo = append(oo, offsets{partition: p, start: start, end: end})
	}

	keyGrep, err := regexp.Compile(cfg.KeyGrep)
	if err != nil {
		return nil, err
	}

	return &Kafka{
		ctx:     ctx,
		topic:   topic,
		offsets: oo,
		keyGrep: keyGrep,
		verbose: cfg.Verbose,
		client:  client,
		decoder: decoder,
		printer: printer,
	}, nil
}

// Run runs the consumer and consumes everything according to its configuration.
// If any [infra] error happens before we even started, it gets written to the output error channel.
// If any [parsing] error happens during the consumption, it's given to a printer.
// When consumer reaches the configured end offset, it stops. Otherwise, it keeps waiting for new messages.
// All consumers will stop if the consumer context is cancelled.
func (k *Kafka) Run() <-chan error {
	errCh := make(chan error)

	go func() {
		wg := sync.WaitGroup{}

		consumer, err := sarama.NewConsumerFromClient(k.client)
		if err != nil {
			errCh <- err
			return
		}

		defer func() {
			if err := consumer.Close(); err != nil {
				errCh <- err
			}
		}()

		for _, o := range k.offsets {
			wg.Add(1)
			go func(topic string, o offsets) {
				defer wg.Done()

				k.log(fmt.Sprintf("# Going to consume from %s until %s", offsetMsg(topic, o.partition, o.start), offsetMsg(topic, o.partition, o.end)))

				c, err := consumer.ConsumePartition(topic, o.partition, o.start)
				if err != nil {
					errCh <- err
					return
				}

				for {
					select {
					case <-k.ctx.Done():
						return
					case message := <-c.Messages():
						if !k.keyGrep.Match(message.Key) {
							continue
						}

						msg, err := k.decoder.Decode(message.Value)
						if err == nil {
							k.printer.Print(output.Msg{
								Key:   string(message.Key),
								Value: msg,
								Topic: topic,
								Time:  message.Timestamp,
							})
						} else {
							k.printer.PrintErr(err)
						}

						if o.end == message.Offset {
							k.log(fmt.Sprintf("# Reached stop timestamp for topic %s: exiting", offsetMsg(topic, o.partition, o.end)))
							return
						}

						if message.Offset+1 == c.HighWaterMarkOffset() {
							k.log(fmt.Sprintf("# Reached stop timestamp for topic %s", offsetMsg(topic, o.partition, o.end)))
						}
					}
				}
			}(k.topic, o)
		}

		wg.Wait()
		close(errCh)
	}()

	return errCh
}

func offsetMsg(topic string, partition int32, offset int64) string {
	om := fmt.Sprintf("%d", offset)
	if offset == sarama.OffsetNewest {
		om = "<end>"
	}
	if offset == sarama.OffsetOldest {
		om = "<start>"
	}
	return fmt.Sprintf("%s [%d] at offset %s", topic, partition, om)
}

func (k *Kafka) log(msg string) {
	if k.verbose {
		fmt.Println(msg)
	}
}
