package pshttp

import (
	"time"

	"github.com/peterbourgon/ps"
)

const (
	// EventTypeData is the EventSource type for data events.
	EventTypeData = "data/v1"

	// EventTypeHeartbeat is the EventSource type for heartbeat events. The
	// event data is the JSON encoding of a [HeartbeatEvent] value.
	EventTypeHeartbeat = "heartbeat/v1"
)

// HeartbeatEvent is sent under the [EventTypeHeartbeat] type.
type HeartbeatEvent struct {
	Timestamp time.Time `json:"ts"`
	Stats     ps.Stats  `json:"stats,omitempty"`
	Error     string    `json:"error,omitempty"`
}
