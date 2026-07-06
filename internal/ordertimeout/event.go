package ordertimeout

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

type Event struct {
	EventID   string    `json:"event_id"`
	OrderID   int64     `json:"order_id"`
	UserID    int64     `json:"user_id"`
	TimeoutAt time.Time `json:"timeout_at"`
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("order timeout event ID is required")
	}
	if e.OrderID <= 0 {
		return errors.New("order timeout order ID must be positive")
	}
	if e.UserID <= 0 {
		return errors.New("order timeout user ID must be positive")
	}
	if e.TimeoutAt.IsZero() {
		return errors.New("order timeout deadline is required")
	}
	return nil
}

func encodeEvent(event Event) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(event)
}

func decodeEvent(body []byte) (Event, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var event Event
	if err := decoder.Decode(&event); err != nil {
		return Event{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Event{}, errors.New("order timeout event contains multiple JSON values")
		}
		return Event{}, err
	}
	if err := event.Validate(); err != nil {
		return Event{}, err
	}
	return event, nil
}
