package api

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/syncthing/syncthing/lib/events"
)

type EventSource struct {
	api        *API
	eventTypes []events.EventType
	lastSeen   int
	start      time.Time
}

func (s *EventSource) Events(ctx context.Context) ([]events.Event, error) {
	var eventTypes string
	if len(s.eventTypes) > 0 {
		var typeStrs []string
		for _, t := range s.eventTypes {
			typeStrs = append(typeStrs, t.String())
		}
		eventTypes = strings.Join(typeStrs, ",")
	}

	if s.start.IsZero() {
		s.start = time.Now()
	}

	r := s.api.client.R()
	r.SetQueryParam("since", strconv.Itoa(s.lastSeen))
	if s.lastSeen == 0 {
		r.SetQueryParam("limit", "1")
	}
	if eventTypes != "" {
		r.SetQueryParam("events", eventTypes)
	}

	r.SetResult([]events.Event{})
	resp, err := r.Get("events")
	if err != nil || resp.IsError() {
		return nil, err
	}
	res := *resp.Result().(*[]events.Event)
	filtered := res[:0]
	for _, e := range res {
		if e.Time.After(s.start) {
			filtered = append(filtered, e)
		}
		s.lastSeen = e.SubscriptionID
	}
	return filtered, nil
}
