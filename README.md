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
  -f, --file string      Proto file path or url
  -h, --help             help for json
      --indent           Indent output json
  -p, --package string   Proto package
  -t, --type string      Proto message type

```

## Examples

Proto file from URL with input message as argument
```shell script
proton json -f https://raw.githubusercontent.com/protocolbuffers/protobuf/master/examples/addressbook.proto -p tutorial -t AddressBook testdata/out.bin
```

Proto file from local with input message as argument
```shell script
proton json -f ./testdata/addressbook.proto -p tutorial -t AddressBook testdata/out.bin
```

Proto file from URL with input message piped
```shell script
cat testdata/out.bin | proton json -f https://raw.githubusercontent.com/protocolbuffers/protobuf/master/examples/addressbook.proto -p tutorial -t AddressBook
```

Proto file from local with input message piped
```shell script
cat testdata/out.bin | proton json -f ./testdata/addressbook.proto -p tutorial -t AddressBook
```


