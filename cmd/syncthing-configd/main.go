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
	stevents "github.com/syncthing/syncthing/lib/events"
	"github.com/thejerf/suture/v4"
	"google.golang.org/protobuf/encoding/prototext"
	"kastelo.dev/syncthing-configd/internal/api"
	"kastelo.dev/syncthing-configd/internal/build"
	"kastelo.dev/syncthing-configd/internal/config"
	"kastelo.dev/syncthing-configd/internal/events"
	"kastelo.dev/syncthing-configd/internal/gc"
)

type CLI struct {
	Config string `short:"c" type:"existingfile" help:"Path to configd.conf" env:"CONFIG_FILE" default:"/etc/syncthing-configd/configd.conf"`
	Debug  bool   `short:"d" help:"Enable debug logging" env:"DEBUG"`
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

	l.Info("Starting Syncthing Configuration Daemon", "version", build.GitVersion, "os", runtime.GOOS, "arch", runtime.GOARCH)

	config, err := loadConfig(cli.Config)
	if err != nil {
		l.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	main := suture.New("main", suture.Spec{
		FailureThreshold: 1,
		EventHook: func(e suture.Event) {
			switch e.Type() {
			case suture.EventTypeServiceTerminate, suture.EventTypeBackoff, suture.EventTypeResume:
				// Business as usual, handled by logging in the services themselves
			default:
				l.Error("Service error", "event", e.String())
			}
		},
	})

	types := []stevents.EventType{stevents.ConfigSaved, stevents.DeviceRejected, stevents.FolderRejected}
	for _, s := range config.Syncthing {
		api := api.NewAPI(l, s.Address, s.ApiKey)
		main.Add(api)

		el := events.NewEventListener(l, api, config, types)
		main.Add(el)

		if config.GarbageCollect.RunEveryS > 0 {
			gc := gc.NewGarbageCollector(l, api, config.GarbageCollect)
			main.Add(gc)
		}
	}

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
