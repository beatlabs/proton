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

## Usage

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

## Examples

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

### Usage with Kafka consumers

Because Proto bytes can contain newlines (`\n`) and often do,
we need to use a different marker to delimit the start and the end of a message byte-stream.

Proton expects a start and an end of message markers, or will read to the end of the stream if they're not provided.

Because there might be some data from each message both after and before the markers, 
Proton can't figure out at the moment where is the end of a message. You need to manage new lines in your `kcat` format.

You can add markers at the end of each message with tools like [kafkacat](https://github.com/edenhill/kcat), like so:

```shell script
kcat -b my-broker:9092 -t my-topic -f '--START--%s--END--\n'
```

You can consume messages and parse them with Proton by doing the following:

```shell script
kcat -b my-broker:9092 -t my-topic -f '--START--%s--END--\n' -o beginning | proton json -f ./my-schema.proto -s '--START--' -m '--END--'
```

This allows you to format any other data (e.g. a timestamp) and specify which part of the data should be parsed as proto binary

```shell
kcat -b my-broker:9092 -t my-topic -f '{"key": "%k", "timestamp": %T, "value": --START--%s--END--}\n' | proton json -f ./my-schema.proto -s '--START--' -m '--END--'
```

**Don't see messages?**

If you execute the above commands, but you don't see messages until you stop the consumer, you might have to adjust your buffer settings:
You can do this with the `stdbuf` command.

```shell script
stdbuf -o0 kcat -b my-broker:9092 -t my-topic -f '%s--END--' -o beginning | proton json -f ./my-schema.proto -m '--END--'
```

If you don't have `stdbuf`, you can install it via `brew install coreutils` (MacOS).

