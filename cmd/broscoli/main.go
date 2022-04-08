package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/kballard/go-shellquote"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type Action struct {
	Command    string `yaml:"command"`
	Wait       bool   `yaml:"wait"`
	execParams []string
}

type Config struct {
	Port    uint16             `yaml:"port"`
	Address string             `yaml:"address"`
	Actions map[string]*Action `yaml:"actions"`
}

const (
	oscPrefix = "/action"
)

func main() {
	configFileFlag := flag.String("config", "config.yaml", "Config file to parse")
	flag.Parse()

	consoleWriter := zerolog.ConsoleWriter{
		Out: os.Stdout,
	}

	if isatty.IsTerminal(os.Stdout.Fd()) {
		consoleWriter.TimeFormat = time.RFC3339
	}

	log.Logger = log.Output(consoleWriter)

	c := Config{}

	b, err := ioutil.ReadFile(*configFileFlag)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("filename", *configFileFlag).
			Msg("Unable to open config file")
	}

	err = yaml.Unmarshal(b, &c)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("filename", *configFileFlag).
			Msg("Unable to parse config file")
	}

	addr := fmt.Sprintf("%s:%d", c.Address, c.Port)

	dispatchMessage := func(msg *osc.Message) {
		if !strings.HasPrefix(msg.Address, oscPrefix) {
			return
		}

		k := msg.Address[len(oscPrefix):]
		a, ok := c.Actions[k]
		if !ok {
			return
		}

		log.Info().
			Str("trigger", msg.Address).
			Str("command", a.Command).
			Bool("wait", a.Wait).
			Msg("Executing command")

		run := func() {
			cmd := exec.Command(a.execParams[0], a.execParams[1:]...)
			if output, err := cmd.CombinedOutput(); err != nil {
				log.Error().
					Err(err).
					Str("command", a.Command).
					Str("output", string(output)).
					Msg("Error running command")
			}
		}

		if a.Wait {
			run()
		} else {
			go run()
		}
	}

	d := osc.NewStandardDispatcher()

	for trigger, a := range c.Actions {
		a.execParams, err = shellquote.Split(a.Command)
		if err != nil {
			log.Error().
				Err(err).
				Str("trigger", oscPrefix+trigger).
				Str("command", a.Command).
				Msg("Cannot parse command for action")
			continue
		}

		log.Debug().
			Str("trigger", oscPrefix+trigger).
			Str("command", a.Command).
			Msg("Installing handler")
		d.AddMsgHandler(oscPrefix+trigger, dispatchMessage)
	}

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: d,
	}

	log.Info().
		Str("address", addr).
		Int("num_actions", len(c.Actions)).
		Msgf("brOSColi is ready to serve")

	server.ListenAndServe()
}
