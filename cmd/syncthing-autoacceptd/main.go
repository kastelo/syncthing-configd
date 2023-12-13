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
	stconfig "github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	stfs "github.com/syncthing/syncthing/lib/fs"
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
	addFolders := make([]stconfig.FolderConfiguration, 0)
	var addDevice *stconfig.DeviceConfiguration
	for _, pat := range cfg.Pattern {
		if pat.MatchesAddress(data.address) {
			addDevice = &stconfig.DeviceConfiguration{
				DeviceID:          data.device,
				Name:              data.name,
				Addresses:         pat.Settings.Addresses,
				AllowedNetworks:   pat.Settings.AllowedNetworks,
				AutoAcceptFolders: pat.Settings.AutoAcceptFolders,
				MaxSendKbps:       int(pat.Settings.MaxSendKbps),
				MaxRecvKbps:       int(pat.Settings.MaxRecvKbps),
				MaxRequestKiB:     int(pat.Settings.MaxRequestKib),
				RawNumConnections: int(pat.Settings.NumConnections),
			}

			for _, fld := range pat.Folder {
				settings := fld.Settings
				if settings == nil {
					settings = &config.FolderConfiguration{}
				}

				folderCfg := stconfig.FolderConfiguration{
					ID:               replaceVariables(fld.Id, data),
					Path:             replaceVariables(settings.Path, data),
					Type:             stconfig.FolderType(settings.Type),
					RescanIntervalS:  int(settings.RescanIntervalS),
					FSWatcherEnabled: !settings.FsWatcherDisabled,
					FSWatcherDelayS:  settings.FsWatcherDelayS,
					IgnorePerms:      settings.IgnorePermissions,
					AutoNormalize:    !settings.NoAutoNormalize,
					MinDiskFree: stconfig.Size{
						Value: settings.GetMinDiskFree().GetValue(),
						Unit:  settings.GetMinDiskFree().GetUnit(),
					},
					Copiers:                 int(settings.Copiers),
					PullerMaxPendingKiB:     int(settings.PullerMaxPendingKib),
					Hashers:                 int(settings.Hashers),
					Order:                   stconfig.PullOrder(settings.Order),
					IgnoreDelete:            settings.IgnoreDelete,
					ScanProgressIntervalS:   int(settings.ScanProgressIntervalS),
					PullerPauseS:            int(settings.PullerPauseS),
					MaxConflicts:            int(settings.MaxConflicts),
					DisableSparseFiles:      settings.DisableSparseFiles,
					DisableTempIndexes:      settings.DisableTempIndexes,
					WeakHashThresholdPct:    int(settings.WeakHashThresholdPct),
					MarkerName:              settings.MarkerName,
					CopyOwnershipFromParent: settings.CopyOwnershipFromParent,
					RawModTimeWindowS:       int(settings.ModTimeWindowS),
					MaxConcurrentWrites:     int(settings.MaxConcurrentWrites),
					DisableFsync:            settings.DisableFsync,
					BlockPullOrder:          stconfig.BlockPullOrder(settings.BlockPullOrder),
					CopyRangeMethod:         stfs.CopyRangeMethod(settings.CopyRangeMethod),
					CaseSensitiveFS:         settings.CaseSensitiveFs,
					JunctionsAsDirs:         settings.FollowJunctions,
					SyncOwnership:           settings.SyncOwnership,
					SendOwnership:           settings.SendOwnership,
					SyncXattrs:              settings.SyncXattrs,
					SendXattrs:              settings.SendXattrs,
					Devices: []stconfig.FolderDeviceConfiguration{
						{DeviceID: data.device},
					},
				}
				addFolders = append(addFolders, folderCfg)
			}
			break
		}
	}
	if addDevice == nil {
		l.Info("Device does not match any pattern, ignored")
		return nil
	}

	l.Info("Accepting device")
	if err := st.SetDevice(addDevice); err != nil {
		l.Error("Failed to add device", "error", err)
	}
	for _, fld := range addFolders {
		l := l.With("folder", fld.ID)
		l.Info("Accepting folder")
		if err := st.SetFolder(&fld); err != nil {
			l.Error("Failed to add folder", "error", err)
		}
	}

	return nil
}

func replaceVariables(s string, d *deviceRejectedData) string {
	return os.Expand(s, func(key string) string {
		switch key {
		case "device":
			return d.device.String()
		case "name":
			return d.name
		case "address":
			return d.address.String()
		}
		return ""
	})
}
