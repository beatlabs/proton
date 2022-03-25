# proton

#### CLI protobuf to json converter.

## Installation

Execute:

```bash
$ go install github.com/beatlabs/proton/v2@latest
```
Or download from [Releases](https://github.com/beatlabs/proton/releases)

Or using [Homebrew 🍺](https://brew.sh)

```bash
brew tap beatlabs/proton https://github.com/beatlabs/proton
brew install proton
```

## Usage as a converter Protobuf to JSON

```shell script
Usage:
  proton json [flags]

Flags:
  -m, --end-of-message-marker string   Marker for end of message used when piping data
  -f, --file string                    Proto file path or url
  -h, --help                           help for json
      --indent                         Indent output json
  -p, --package string                 Proto package
                                       Defaults to the package found in the Proton file if not specified
  -t, --type string                    Proto message type
                                       Defaults to the first message type in the Proton file if not specified
```

### Examples

Proto file from URL with input message as argument
```shell script
proton json -f https://raw.githubusercontent.com/protocolbuffers/protobuf/master/examples/addressbook.proto testdata/out.bin
```

Proto file from local with input message as argument
```shell script
proton json -f ./testdata/addressbook.proto testdata/out.bin
```

Proto file from URL with input message piped
```shell script
cat testdata/out.bin | proton json -f https://raw.githubusercontent.com/protocolbuffers/protobuf/master/examples/addressbook.proto
```

Proto file from local with input message piped
```shell script
cat testdata/out.bin | proton json -f ./testdata/addressbook.proto
```

Multiple proto files from a producer with input messages piped
```shell script
./testdata/producer.sh '--END--' | proton json -f ./testdata/addressbook.proto -m '--END--'
```

### Piping data from Kafkacat

Because Proto bytes can contain newlines (`\n`) and often do,
we need to use a different marker to delimit the end of a message byte-stream and the beginning of the next.
Proton expects an end of message marker, or will read to the end of the stream if not provided.

You can add markers at the end of each messae with tools like [kafkacat](https://github.com/edenhill/kcat), like so:

```shell script
kcat -b my-broker:9092 -t my-topic -f '%s--END--'
```

You can consume messages and parse them with Proton by doing the following:

```shell script
kcat -b my-broker:9092 -t my-topic -f '%s--END--' -o beginning | proton json -f ./my-schema.proto -m '--END--'
```

**Don't see messages?**

If you execute the above command, but you don't see messages until you stop the consumer, you might have to adjust your buffer settings:
You can do this with the `stdbuf` command.

```shell script
stdbuf -o0 kcat -b my-broker:9092 -t my-topic -f '%s--END--' -o beginning | proton json -f ./my-schema.proto -m '--END--'
```

If you don't have `stdbuf`, you can install it via `brew install coreutils`.

## Using proton as a standalone Kafka consumer

Proton can consume from Kafka directly. The syntax of all the parameters is kept as close as possible to the same from Kafkacat.

```shell
$ proton consume --help
consume from given topics

Usage:
  proton consume [flags]

Flags:
  -b, --broker string     Broker URL to consume from
  -f, --format string
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
                          	-f 'Key: %k, Time: %Tf \nValue: %s' (default "%Tf: %s")
  -h, --help              help for consume
      --key string        Grep RegExp for a key value (default ".*")
  -o, --offsets strings
                          Offset to start consuming from
                          	 s@<value> (timestamp in ms to start at)
                          	 e@<value> (timestamp in ms to stop at (not included))

      --proto string      A path to a proto file an URL to it
  -t, --topic string      A topic to consume from
  -v, --verbose           Whether to print out proton's debug messages
```

The minimal configuration to run Proton as a standalone consumer is
```shell
proton consume -b my-broker -t my-topic --proto ./my-schema.proto
```
This would consume all the messages from the topic since its start and use default formatting.

You can specify the start and/or the end offset timestamp in milliseconds. Both are optional.
```shell
proton consume -b my-broker -t my-topic --proto ./my-schema.proto -o s@1646218065015 -o e@1646218099197
```
If the end offset is set, proton will stop consuming once it's reached. Otherwise, it will keep consuming.

You can specify the format of the output.
```shell
$ proton consume -b my-broker -t my-topic --proto ./my-schema.proto -f "Time: %T \t %k\t%s"
# ...
Time: 1646218065015 	 key  {"field1":"value1","field2":"value2"}
Time: 1646218099197 	 key  {"field1":"value1","field2":"value2"}
# ... 
```
Run `proton consume -h` to see all the available formatting options.

To filter out keys, you can use `--key <regexp>` option like in this example:
```shell
proton consume -b my-broker -t my-topic --proto ./my-schema.proto --key "my-key"
proton consume -b my-broker -t my-topic --proto ./my-schema.proto --key "my-k.*"
```


