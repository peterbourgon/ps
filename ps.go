package ps

import (
	"errors"
	"fmt"
)

var (
	// ErrAlreadySubscribed signals that a given subscription already exists.
	ErrAlreadySubscribed = errors.New("already subscribed")

	// ErrNotSubscribed indicates that a given subscription doesn't exist.
	ErrNotSubscribed = errors.New("not subscribed")
)

// Stats represents the outcome of one or more published values.
type Stats struct {
	// Skips are values that were not sent due to filtering rules.
	Skips uint64

	// Sends are values that were sent successfully.
	Sends uint64

	// Drops are values that failed to send because the subscriber blocked.
	Drops uint64
}

// Total number of values represented by the stats.
func (s Stats) Total() uint64 {
	return s.Skips + s.Sends + s.Drops
}

// String representation of the stats.
func (s Stats) String() string {
	return fmt.Sprintf("Skips=%d Sends=%d Drops=%d Total=%d", s.Skips, s.Sends, s.Drops, s.Total())
}
