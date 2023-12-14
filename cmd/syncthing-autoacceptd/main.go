package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/thejerf/suture/v4"
	"google.golang.org/protobuf/encoding/prototext"
	"kastelo.dev/syncthing-autoacceptd/internal/api"
	"kastelo.dev/syncthing-autoacceptd/internal/build"
	"kastelo.dev/syncthing-autoacceptd/internal/config"
	"kastelo.dev/syncthing-autoacceptd/internal/processor"
)

type CLI struct {
	Address  string `short:"a" help:"Address of Syncthing API" default:"127.0.0.1:8384" env:"SYNCTHING_AUTOACCEPTD_ADDRESS"`
	APIKey   string `short:"k" help:"Key for the Syncthing API" required:"true" env:"SYNCTHING_AUTOACCEPTD_APIKEY"`
	Patterns string `short:"p" type:"existingfile" help:"Path to patterns.conf" required:"true" env:"SYNCTHING_AUTOACCEPTD_PATTERNS_FILE"`
	Debug    bool   `short:"d" help:"Enable debug logging" env:"SYNCTHING_AUTOACCEPTD_DEBUG"`
}

func main() {
	// Parse CLI options
	var cli CLI
	kong.Parse(&cli)

	// Created a logger with console coloring and optional debug logging
	w := os.Stdout
	level := slog.LevelInfo
	if cli.Debug {
		level = slog.LevelDebug
	}
	l := slog.New(tint.NewHandler(w, &tint.Options{
		Level:     level,
		AddSource: cli.Debug,
		NoColor:   !isatty.IsTerminal(w.Fd()),
	}))
	slog.SetDefault(l)

	l.Info("Starting Syncthing Auto Accept Daemon", "version", build.GitVersion, "os", runtime.GOOS, "arch", runtime.GOARCH)

	config, err := loadConfig(cli.Patterns)
	if err != nil {
		l.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	api := api.NewAPI(l, cli.Address, cli.APIKey)
	types := []events.EventType{events.ConfigSaved, events.DeviceRejected, events.FolderRejected}
	proc := processor.NewEventListener(l, api, config, types)

	main := suture.New("main", suture.Spec{
		FailureThreshold: 2,
		EventHook: func(e suture.Event) {
			if e.Type() == suture.EventTypeResume {
				l.Info("Service resuming", "event", e.String())
			} else if v, ok := e.Map()["restarting"].(bool); ok && v {
				l.Warn("Service restarting", "event", e.String())
			} else {
				l.Error("Service error", "event", e.String())
			}
		},
	})

	main.Add(api)
	main.Add(proc)

	if err := main.Serve(context.Background()); err != nil {
		l.Error("Failed to run service", "error", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.Configuration, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	uo := prototext.UnmarshalOptions{}
	var patterns config.Configuration
	if err := uo.Unmarshal(bs, &patterns); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := patterns.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &patterns, nil
}
