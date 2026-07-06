package ordertimeout

import (
	"strings"
	"testing"
	"time"
)

func TestOrderTimeoutEventRoundTrip(t *testing.T) {
	want := Event{EventID: "event-1", OrderID: 10, UserID: 20, TimeoutAt: time.Now().UTC().Truncate(time.Millisecond)}
	body, err := encodeEvent(want)
	if err != nil {
		t.Fatalf("encodeEvent: %v", err)
	}
	got, err := decodeEvent(body)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if got.EventID != want.EventID || got.OrderID != want.OrderID || got.UserID != want.UserID || !got.TimeoutAt.Equal(want.TimeoutAt) {
		t.Fatalf("event=%+v want=%+v", got, want)
	}
}

func TestDecodeOrderTimeoutEventRejectsInvalidPayload(t *testing.T) {
	for _, body := range []string{
		`{"event_id":"","order_id":1,"user_id":1,"timeout_at":"2026-01-01T00:00:00Z"}`,
		`{"event_id":"event","order_id":0,"user_id":1,"timeout_at":"2026-01-01T00:00:00Z"}`,
		`{"event_id":"event","order_id":1,"user_id":1,"timeout_at":"2026-01-01T00:00:00Z","extra":true}`,
		`{} {}`,
	} {
		if _, err := decodeEvent([]byte(body)); err == nil {
			t.Fatalf("decodeEvent(%s): want error", body)
		}
	}
}

func TestExpirationMilliseconds(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := expirationMilliseconds(now, now.Add(30*time.Minute)); got != "1800000" {
		t.Fatalf("expiration=%s", got)
	}
	if got := expirationMilliseconds(now, now.Add(-time.Minute)); got != "1" {
		t.Fatalf("overdue expiration=%s", got)
	}
	if strings.TrimSpace(expirationMilliseconds(now, now)) == "" {
		t.Fatal("expiration must not be empty")
	}
}
