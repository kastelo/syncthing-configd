package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-resty/resty/v2"
	stconfig "github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/thejerf/suture/v4"
)

type API struct {
	suture.Service
	log             *slog.Logger
	address         string
	client          *resty.Client
	serialisedFuncs chan func()
	configChangers  chan func(cfg *stconfig.Configuration)
	events          chan events.Event
}

func NewAPI(l *slog.Logger, address string, apiKey string) *API {
	url := fmt.Sprintf("http://%s/rest/", address)

	c := resty.New()
	c.SetBaseURL(url)
	c.SetAuthScheme("Bearer")
	c.SetAuthToken(apiKey)

	svc := suture.NewSimple("API")

	api := &API{
		Service:         svc,
		log:             l.With("address", address),
		address:         address,
		client:          c,
		serialisedFuncs: make(chan func(), 1),
		configChangers:  make(chan func(cfg *stconfig.Configuration), 1),
		events:          make(chan events.Event, 64),
	}

	svc.Add(configService{API: api})

	return api
}

func (s *API) Address() string {
	return s.address
}

func (s *API) String() string {
	return fmt.Sprintf("api(%s)@%p", s.address, s)
}

type configService struct {
	*API
}

func (s configService) Serve(ctx context.Context) error {
	s.log.Debug("Starting config service")
	defer s.log.Debug("Stopping config service")

	const afterConfigWait = time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case fn := <-s.serialisedFuncs:
			fn()
		case fn := <-s.configChangers:
			cfg, err := s.GetConfig()
			if err != nil {
				s.log.Error("Failed to get config", "error", err)
				continue
			}
			fn(cfg)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(afterConfigWait):
			}
		}
	}
}

func (s *API) Events(types []events.EventType) *EventSource {
	return &EventSource{
		api:        s,
		eventTypes: types,
	}
}

func (s *API) GetConfig() (*stconfig.Configuration, error) {
	var cfg stconfig.Configuration
	r := s.client.R()
	r.SetResult(&cfg)
	_, err := r.Get("config")
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

type SystemStatus struct {
	MyID protocol.DeviceID `json:"myID"`
}

type maybe[T any] struct {
	value T
	err   error
}

func maybeFunc[T any](f func() (T, error)) maybe[T] {
	v, err := f()
	return maybe[T]{v, err}
}

func (s *API) GetSystemStatus() (*SystemStatus, error) {
	resC := make(chan maybe[*SystemStatus], 1)
	s.serialisedFuncs <- func() {
		resC <- maybeFunc(func() (*SystemStatus, error) {
			var status SystemStatus
			r := s.client.R()
			r.SetResult(&status)
			resp, err := r.Get("system/status")
			if err != nil {
				return nil, err
			}
			if resp.IsError() {
				return nil, errors.New(resp.Status())
			}
			return &status, nil
		})
	}
	res := <-resC
	return res.value, res.err
}

type SystemVersion struct {
	Version string
	OS      string
	Arch    string
}

func (s *API) GetSystemVersion() (*SystemVersion, error) {
	resC := make(chan maybe[*SystemVersion], 1)
	s.serialisedFuncs <- func() {
		resC <- maybeFunc(func() (*SystemVersion, error) {
			var status SystemVersion
			r := s.client.R()
			r.SetResult(&status)
			resp, err := r.Get("system/version")
			if err != nil {
				return nil, err
			}
			if resp.IsError() {
				return nil, errors.New(resp.Status())
			}
			return &status, nil
		})
	}
	res := <-resC
	return res.value, res.err
}

func (s *API) SetDevice(cfg *stconfig.DeviceConfiguration) error {
	errC := make(chan error, 1)
	s.configChangers <- func(cur *stconfig.Configuration) {
		if cur == nil {
			errC <- errors.New("getting config failed")
			return
		}
		for _, d := range cur.Devices {
			if d.DeviceID == cfg.DeviceID {
				errC <- nil
				return
			}
		}

		r := s.client.R()
		r.SetBody(cfg)
		resp, err := r.Put("config/devices/" + cfg.DeviceID.String())
		if err != nil {
			errC <- err
			return
		}
		if resp.IsError() {
			errC <- errors.New(resp.Status())
			return
		}
		errC <- nil
	}
	return <-errC
}

func (s *API) SetFolder(cfg *stconfig.FolderConfiguration) error {
	errC := make(chan error, 1)
	s.configChangers <- func(cur *stconfig.Configuration) {
		if cur == nil {
			errC <- errors.New("getting config failed")
			return
		}

		var resp *resty.Response
		var err error
		exists := false
		r := s.client.R()
		for _, f := range cur.Folders {
			if f.ID == cfg.ID {
				// Folder exists, just add device
				exists = true
				f.Devices = append(f.Devices, cfg.Devices...)

				r.SetBody(f)
				resp, err = r.Patch("config/folders/" + f.ID)
				break
			}
		}

		if !exists {
			// Folder does not exist, create it
			r.SetBody(cfg)
			resp, err = r.Post("config/folders")
		}

		if err != nil {
			errC <- err
			return
		}
		if resp.IsError() {
			errC <- errors.New(resp.Status())
			return
		}
		errC <- nil
	}
	return <-errC
}
