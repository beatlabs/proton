# proton

#### cli protobuf to json converter.

## Installation

Execute:

```bash
$ go get github.com/beatlabs/proton
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
                                       Defaults to "--END--" if not provided
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
./testdata/producer.sh | proton json -f ./testdata/addressbook.proto
```

### Usage with Kafka consumers

Because Proto bytes can contain newlines (`\n`) and often do, we need to use a different marker to delimit the end of a message byte-stream and the beginning of the next.
Proton expects this separator to be on a new line, and expects `--END--` by default.

You can add separators with tool like Kafkacat, like so:

```shell script
kcat -b my-broker:9092 -t my-topic -f '%s\n--END--\n'
```

You can consume messages and parse them with Proton by doing the following:

```shell script
kcat -b my-broker:9092 -t my-topic -f '%s\n--END--\n' -o beginning | proton json -f ./my-schema.proto
```

**Don't see messages?**

If you execute the above command, but you don't see messages until you stop the consumer, you might have to adjust your buffer settings:
You can do this with the `stdbuf` command.

```shell script
stdbuf -o0 kcat -b my-broker:9092 -t my-topic -f '%s\n--END--\n' -o beginning | proton json -f ./my-schema.proto
```

If you don't have `stdbuf`, you can install it via `brew install coreutils`.

