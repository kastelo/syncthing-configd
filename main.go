package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/go-resty/resty/v2"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

var (
	errMalformedEvent = errors.New("malformed event")
)

func main() {
	stAddr := kingpin.Flag("syncthing-addr", "Address of Syncthing API").Default("127.0.0.1:8384").String()
	stApiKey := kingpin.Flag("syncthing-api-key", "Key for the Syncthing API").Required().String()
	kingpin.Parse()

	st := NewSyncthing(*stAddr, *stApiKey)
	evs := st.Events()
	for ev := range evs {
		devID, err := deviceInEvent(ev)
		if err != nil {
			log.Println("Processing event:", err)
			continue
		}
		if err := handleDeviceRejected(st, devID); err != nil {
			log.Println("Processing device:", err)
		}
		log.Println("Added device", devID)
	}
}

func handleDeviceRejected(st *syncthing, devID protocol.DeviceID) error {
	cfg, err := st.GetConfig()
	if err != nil {
		return err
	}

	cfg.Devices = append(cfg.Devices, config.DeviceConfiguration{DeviceID: devID})
	for i, fld := range cfg.Folders {
		cfg.Folders[i].Devices = append(fld.Devices, config.FolderDeviceConfiguration{DeviceID: devID})
	}

	return st.SetConfig(cfg)
}

func deviceInEvent(ev events.Event) (protocol.DeviceID, error) {
	dataMap, ok := ev.Data.(map[string]interface{})
	if !ok {
		return protocol.EmptyDeviceID, errMalformedEvent
	}
	devStr, ok := dataMap["device"].(string)
	if !ok {
		return protocol.EmptyDeviceID, errMalformedEvent
	}
	return protocol.DeviceIDFromString(devStr)
}

type syncthing struct {
	apiURL string
	client *resty.Client
}

func NewSyncthing(hostPort string, apiKey string) *syncthing {
	url := fmt.Sprintf("http://%s/rest/", hostPort)
	c := resty.New()
	c.SetHostURL(url)
	c.SetRetryWaitTime(1 * time.Second)
	c.SetRetryCount(5)
	c.SetHeader("X-API-Key", apiKey)
	return &syncthing{url, c}
}

func (st *syncthing) Events() <-chan events.Event {
	eventCh := make(chan events.Event)
	go func() {
		// skip old events, client got current state from filesystem
		start := time.Now()
		lastSeen := 0
		for {
			r := st.client.R()
			r.SetQueryParam("since", strconv.Itoa(lastSeen))
			r.SetQueryParam("events", "DeviceRejected")
			r.SetResult([]events.Event{})
			resp, err := r.Get("events")
			if err != nil {
				log.Printf("API Get events: %s", err)
				continue
			}
			if resp.IsError() {
				log.Printf("API Get events status: %s (body: %s)", resp.Status(), resp.String())
				time.Sleep(2 * time.Second)
				continue
			}
			for _, e := range *resp.Result().(*[]events.Event) {
				if e.Time.After(start) {
					eventCh <- e
				}
				lastSeen = e.SubscriptionID
			}
		}
	}()
	return eventCh
}

func (st *syncthing) GetConfig() (config.Configuration, error) {
	var cfg config.Configuration
	r := st.client.R()
	r.SetResult(&cfg)
	_, err := r.Get("system/config")
	if err != nil {
		return config.Configuration{}, err
	}
	return cfg, nil
}

func (st *syncthing) SetConfig(cfg config.Configuration) error {
	r := st.client.R()
	r.SetBody(cfg)
	_, err := r.Post("system/config")
	return err
}
