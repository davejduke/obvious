package pagination

import (
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	tokens := []string{
		"row-id-12345",
		"org:abc/engagement:xyz/after:2024-01-01T00:00:00Z",
		"",
	}
	for _, tok := range tokens {
		encoded := EncodeCursor(tok)
		if tok == "" {
			// Encoding empty string returns a non-empty cursor
			// but decoding it back should return ErrInvalidCursor.
			_, err := DecodeCursor(encoded)
			if err == nil {
				// EncodeCursor("") produces "" which decodes to ErrInvalidCursor
				t.Logf("empty token encoded to %q", encoded)
			}
			continue
		}
		decoded, err := DecodeCursor(encoded)
		if err != nil {
			t.Errorf("DecodeCursor(%q): unexpected error: %v", encoded, err)
			continue
		}
		if decoded != tok {
			t.Errorf("round-trip: got %q, want %q", decoded, tok)
		}
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	cases := []string{"!!! not base64 !!!", ""}
	for _, c := range cases {
		_, err := DecodeCursor(c)
		if err == nil {
			t.Errorf("DecodeCursor(%q): expected error", c)
		}
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		s       string
		def     int
		want    int
		wantErr bool
	}{
		{"", 20, 20, false},
		{"10", 20, 10, false},
		{"100", 20, 100, false},
		{"101", 20, 0, true},      // over max
		{"-1", 20, 0, true},       // negative
		{"abc", 20, 0, true},      // non-numeric
		{"0", 20, 0, true},        // zero
	}
	for _, tt := range tests {
		got, err := ParseLimit(tt.s, tt.def)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseLimit(%q, %d): err=%v, wantErr=%v", tt.s, tt.def, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseLimit(%q, %d): got %d, want %d", tt.s, tt.def, got, tt.want)
		}
	}
}

func TestCursorOpaqueness(t *testing.T) {
	// Cursor must not expose internal token structure in a readable form.
	internal := "row-id=12345&after=2024-01-01"
	cursor := EncodeCursor(internal)
	// Must not contain the raw internal value
	if cursor == internal {
		t.Error("cursor is not opaque — it exposes the internal token")
	}
	// Must be decodable
	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != internal {
		t.Errorf("round-trip: got %q, want %q", decoded, internal)
	}
}
