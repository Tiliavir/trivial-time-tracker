package cmd

import "testing"

func TestCsvEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain", "plain"},
		{"with space", "with space"},
		{"with,comma", `"with,comma"`},
		{`with"quote`, `"with""quote"`},
		{"with\nnewline", "\"with\nnewline\""},
		{"with\rreturn", "\"with\rreturn\""},
		{"", ""},
	}
	for _, tt := range tests {
		got := csvEscape(tt.input)
		if got != tt.want {
			t.Errorf("csvEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
