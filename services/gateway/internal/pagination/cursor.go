// Package pagination provides cursor-based pagination helpers for the API gateway.
//
// Cursors are opaque base64url-encoded strings. The maximum page size is 100.
// Callers encode arbitrary internal tokens (e.g. a row ID or composite key)
// and pass them to clients. Clients return the cursor verbatim on subsequent
// requests; the gateway decodes it and passes the value to the backend.
package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

const (
	// MaxPageSize is the maximum number of items per page.
	MaxPageSize = 100
	// DefaultPageSize is the default number of items per page.
	DefaultPageSize = 20
)

// ErrInvalidCursor is returned when a cursor cannot be decoded.
var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// ErrPageSizeTooLarge is returned when the requested limit exceeds MaxPageSize.
var ErrPageSizeTooLarge = fmt.Errorf("pagination: page size exceeds maximum of %d", MaxPageSize)

// ErrPageSizeInvalid is returned when the limit is not a positive integer.
var ErrPageSizeInvalid = errors.New("pagination: limit must be a positive integer")

// EncodeCursor encodes an internal token as an opaque base64url cursor string.
func EncodeCursor(token string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(token))
}

// DecodeCursor decodes an opaque cursor back to an internal token.
func DecodeCursor(cursor string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", ErrInvalidCursor
	}
	if len(b) == 0 {
		return "", ErrInvalidCursor
	}
	return string(b), nil
}

// ParseLimit parses a limit query-string value and enforces the MaxPageSize cap.
// Returns def if the value is empty.
func ParseLimit(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, ErrPageSizeInvalid
	}
	if n > MaxPageSize {
		return 0, ErrPageSizeTooLarge
	}
	return n, nil
}

// Page represents a page of results with optional next/prev cursors.
type Page struct {
	Items      []any   `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	PrevCursor string  `json:"prev_cursor,omitempty"`
	Total      int     `json:"total,omitempty"`
}
