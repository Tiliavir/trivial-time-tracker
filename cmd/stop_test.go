package cmd

import "testing"

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		seconds int64
		want    string
	}{
		{0, "0s"},
		{30, "30s"},
		{59, "59s"},
		{60, "1m 0s"},
		{90, "1m 30s"},
		{3600, "1h 0m 0s"},
		{3661, "1h 1m 1s"},
		{7322, "2h 2m 2s"},
	}
	for _, tt := range tests {
		got := formatElapsed(tt.seconds)
		if got != tt.want {
			t.Errorf("formatElapsed(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}
