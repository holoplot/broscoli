# brOSColi - simple OSC command executor

[![CircleCI](https://circleci.com/gh/holoplot/broscoli/tree/main.svg?style=svg)](https://circleci.com/gh/holoplot/broscoli/tree/main)

Broscoli is a simple executor of local commands, triggered by OSC messages.
It can be used to run local scripts or other executables when a configured OSC messages is received.

## Install

```
$ go install github.com/holoplot/broscoli/cmd/broscoli
```

## Compile from sources

```sh
$ go build -o broscoli ./cmd/broscoli/...
```

## Usage

```sh
$ ./broscoli --help
Usage of ./broscoli:
  -config string
        Config file to parse (default "config.yaml")
```

## OSC messages

Handlers are installed for all actions listed in the config file, prefixed by `/action`.
Hence if there is an action `/foo` in the config file, clients can trigger it by sending an OSC messgage `/action/foo`.

## Config

Configuration is done through a YAML file:

```yaml
port: 8765
address: 0.0.0.0 # Set to 127.0.0.1 if you only have local senders
actions:
  /test1:
    command: /bin/touch "/tmp/file with spaces"
    wait: true

  /test2:
    command: /bin/rm "/tmp/file with spaces"
    wait: true
```

The `wait` flag specifies whether the command is executed synchronously or asynchronously.
When set to `true`, the server will block execution until the given command exits.

## License

MIT