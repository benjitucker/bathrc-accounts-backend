package jotform

import (
	"encoding/json"
	"fmt"
	"time"
)

type RawRequestPayload interface {
	FormKind() string
}

// UnixMillis parses millisecond timestamps stored as strings
type UnixMillis time.Time

func (t *UnixMillis) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	var ms int64
	if _, err := fmt.Sscan(s, &ms); err != nil {
		return err
	}

	*t = UnixMillis(time.UnixMilli(ms))
	return nil
}

func (t UnixMillis) Time() time.Time {
	return time.Time(t)
}
