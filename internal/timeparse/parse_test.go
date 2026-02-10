package timeparse

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"2h", 7200},
		{"30m", 1800},
		{"2h 30m", 9000},
		{"1.5h", 5400},
		{"2ч 30м", 9000},
		{"1ч", 3600},
		{"45м", 2700},
		{"0h", 0},
		{"", 0},
		{"  2h  30m  ", 9000},
		{"1H 15M", 4500},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Parse(tt.input)
			if got != tt.want {
				t.Errorf("Parse(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
