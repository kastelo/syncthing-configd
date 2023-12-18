package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/syncthing/syncthing/lib/events"
	"kastelo.dev/syncthing-configd/internal/api"
	"kastelo.dev/syncthing-configd/internal/config"
)

type EventListener struct {
	log        *slog.Logger
	api        *api.API
	patterns   *config.Configuration
	eventTypes []events.EventType
}

func NewEventListener(log *slog.Logger, api *api.API, patterns *config.Configuration, eventTypes []events.EventType) *EventListener {
	return &EventListener{
		log:        log.With("address", api.Address()),
		api:        api,
		patterns:   patterns,
		eventTypes: eventTypes,
	}
}

func (s *EventListener) Serve(ctx context.Context) error {
	defer time.Sleep(time.Second) // slow down retry rate

	ver, err := s.api.GetSystemVersion()
	if err != nil {
		s.log.Error("Failed to get Syncthing version", "error", err)
		return err
	}
	stat, err := s.api.GetSystemStatus()
	if err != nil {
		s.log.Error("Failed to get Syncthing status", "error", err)
		return err
	}

	s.log.Info("Connected to Syncthing", "version", ver.Version, "os", ver.OS, "arch", ver.Arch, "id", stat.MyID)

	es := s.api.Events(s.eventTypes)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.log.Debug("Listening for events...")
		evs, err := es.Events(ctx)
		if err != nil {
			s.log.Error("Failed to get events", "error", err)
			return err
		}

		for _, ev := range evs {
			s.log.Debug("Received event", "type", ev.Type, "id", ev.GlobalID)
			switch ev.Type {
			case events.DeviceRejected:
				data, err := getDeviceRejectedData(ev)
				if err != nil {
					s.log.Error("Failed to process DeviceRejected event", "error", err)
					continue
				}
				if err := s.handleDeviceRejected(data, s.patterns); err != nil {
					s.log.Error("Failed to process device", "error", err)
				}
			case events.FolderRejected:
				s.log.Debug("FolderRejected event ignored", "event", ev)
			}
		}
	}
}

func (s *EventListener) String() string {
	return fmt.Sprintf("eventListener(%s)@%p", s.api.Address(), s)
}

func (s *EventListener) handleDeviceRejected(data *deviceRejectedData, cfg *config.Configuration) error {
	l := slog.With("device", data.device, "name", data.name, "address", data.address)

	addDevice, addFolders, error := getDeviceRejectedConfigs(data, cfg)
	if error != nil {
		if errors.Is(error, errNoMatchingPattern) {
			l.Info("No matching pattern found")
			return nil
		}
		return error
	}

	l.Info("Accepting device")
	if err := s.api.SetDevice(addDevice); err != nil {
		l.Error("Failed to add device", "error", err)
	}
	for _, fld := range addFolders {
		l := l.With("folder", fld.ID)
		l.Info("Accepting folder")
		if err := s.api.SetFolder(fld); err != nil {
			l.Error("Failed to add folder", "error", err)
		}
	}

	return nil
}
