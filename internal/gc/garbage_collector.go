package gc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	stconfig "github.com/syncthing/syncthing/lib/config"
	"kastelo.dev/syncthing-configd/internal/api"
	"kastelo.dev/syncthing-configd/internal/config"
)

type GarbageCollector struct {
	log *slog.Logger
	api *api.API
	cfg *config.GarbageCollection
}

func NewGarbageCollector(log *slog.Logger, api *api.API, cfg *config.GarbageCollection) *GarbageCollector {
	return &GarbageCollector{
		log: log.With("address", api.Address()),
		api: api,
		cfg: cfg,
	}
}

func (s *GarbageCollector) Serve(ctx context.Context) error {
	interval := time.Duration(s.cfg.RunEveryS) * time.Second
	next := time.Now().Truncate(interval).Add(interval)

	stat, err := s.api.GetSystemStatus()
	if err != nil {
		s.log.Error("Failed to get Syncthing status", "error", err)
		return err
	}

	for {
		s.log.Debug("Waiting for next run", "next", next)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(next)):
			next = next.Add(interval)
		}

		cfg, err := s.api.GetConfig()
		if err != nil {
			s.log.Error("Failed to get Syncthing configuration", "error", err)
			return err
		}

		if s.cfg.UnseenDevicesDays > 0 {
			s.gcUnseenDevices(cfg, stat)
		}

		if s.cfg.UnsharedFolders {
			s.gcUnsharedFolders(cfg)
		}
	}
}

func (s *GarbageCollector) String() string {
	return fmt.Sprintf("garbageCollector(%s)@%p", s.api.Address(), s)
}

func (s *GarbageCollector) gcUnseenDevices(cfg *stconfig.Configuration, stat *api.SystemStatus) {
	// Gather device statics; remove devices with a last seen time older
	// than the threshold.

	stats, err := s.api.GetDeviceStats()
	if err != nil {
		s.log.Error("Failed to get Syncthing device stats", "error", err)
	}

	for _, dev := range cfg.Devices {
		if dev.DeviceID == stat.MyID {
			continue
		}

		lastSeen := stats[dev.DeviceID].LastSeen
		daysSince := int(time.Since(lastSeen) / time.Hour / 24)

		s.log.Debug("Checking device", "device", dev.DeviceID, "name", dev.Name, "lastSeen", lastSeen, "daysSince", daysSince)

		if lastSeen.IsZero() || lastSeen.Unix() == 0 {
			// Not seen at all; skip for now.
			s.log.Debug("Skipping device", "device", dev.DeviceID, "name", dev.Name, "reason", "never seen")
			continue
		}

		if daysSince > int(s.cfg.UnseenDevicesDays) {
			s.log.Info("Removing device", "device", dev.DeviceID, "name", dev.Name)
			if err := s.api.RemoveDevice(dev.DeviceID); err != nil {
				s.log.Error("Failed to remove device", "device", dev.DeviceID, "name", dev.Name, "error", err)
			}
		}
	}
}

func (s *GarbageCollector) gcUnsharedFolders(cfg *stconfig.Configuration) {
	// Remove folders that are not shared with anyone.
	for _, fld := range cfg.Folders {
		s.log.Debug("Checking folder", "folder", fld.ID, "label", fld.Label, "numDevices", len(fld.Devices))
		if len(fld.Devices) == 1 { // only self
			s.log.Info("Removing folder", "folder", fld.ID, "label", fld.Label)
			if err := s.api.RemoveFolder(fld.ID); err != nil {
				s.log.Error("Failed to remove folder", "folder", fld.ID, "label", fld.Label, "error", err)
			}
		}
	}
}
