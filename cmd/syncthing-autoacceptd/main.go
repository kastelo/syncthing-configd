package main

import (
	"context"
	"errors"
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
)

var errMalformedEvent = errors.New("malformed event")

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

	main := suture.NewSimple("main")
	main.ServeBackground(context.Background())

	// Read and validate config file
	var patterns config.Configuration
	bs, err := os.ReadFile(cli.Patterns)
	if err != nil {
		l.Error("Failed to read config", "error", err)
		os.Exit(2)
	}
	if err := patterns.Validate(); err != nil {
		l.Error("Failed to validate config", "error", err)
		os.Exit(2)
	}

	uo := prototext.UnmarshalOptions{}
	if err := uo.Unmarshal(bs, &patterns); err != nil {
		l.Error("Failed to unmarshal config", "error", err)
		os.Exit(2)
	}

	types := []events.EventType{events.ConfigSaved, events.DeviceRejected, events.FolderRejected}
	st := api.NewAPI(l, cli.Address, cli.APIKey, types)
	main.Add(st)

	ver, err := st.GetSystemVersion()
	if err != nil {
		l.Error("Failed to get Syncthing version", "error", err)
		os.Exit(1)
	}
	stat, err := st.GetSystemStatus()
	if err != nil {
		l.Error("Failed to get Syncthing status", "error", err)
		os.Exit(1)
	}
	l.Info("Connected to Syncthing", "version", ver.Version, "os", ver.OS, "arch", ver.Arch, "id", stat.MyID)

	l.Debug("Listening for events...", "types", types)

	for ev := range st.Events() {
		l.Debug("Received event", "type", ev.Type, "id", ev.GlobalID)
		switch ev.Type {
		case events.DeviceRejected:
			data, err := getDeviceRejectedData(ev)
			if err != nil {
				l.Error("Failed to process DeviceRejected event", "error", err)
				continue
			}
			log := slog.With("device", data.device, "name", data.name, "address", data.address)
			if err := handleDeviceRejected(log, st, data, &patterns); err != nil {
				l.Error("Failed to process device", "error", err)
			}
		case events.FolderRejected:
			l.Debug("FolderRejected event ignored", "event", ev)
		}
	}
}

func handleDeviceRejected(l *slog.Logger, st *api.API, data *deviceRejectedData, cfg *config.Configuration) error {
	addDevice, addFolders, error := getDeviceRejectedConfigs(data, cfg)
	if error != nil {
		if errors.Is(error, errNoMatchingPattern) {
			l.Info("No matching pattern found")
			return nil
		}
		return error
	}

	l.Info("Accepting device")
	if err := st.SetDevice(addDevice); err != nil {
		l.Error("Failed to add device", "error", err)
	}
	for _, fld := range addFolders {
		l := l.With("folder", fld.ID)
		l.Info("Accepting folder")
		if err := st.SetFolder(fld); err != nil {
			l.Error("Failed to add folder", "error", err)
		}
	}

	return nil
}
