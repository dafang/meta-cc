package sequences

import (
	"testing"
)

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "unicode arrow",
			pattern: "Read → Edit → Bash",
			want:    []string{"Read", "Edit", "Bash"},
		},
		{
			name:    "ascii arrow",
			pattern: "Read -> Edit -> Bash",
			want:    []string{"Read", "Edit", "Bash"},
		},
		{
			name:    "with extra spaces",
			pattern: "Read  ->  Edit  ->  Bash",
			want:    []string{"Read", "Edit", "Bash"},
		},
		{
			name:    "single tool",
			pattern: "Read",
			want:    []string{"Read"},
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePattern(tt.pattern)
			if len(got) != len(tt.want) {
				t.Errorf("parsePattern() returned %d tools, want %d", len(got), len(tt.want))
				return
			}
			for i, tool := range got {
				if tool != tt.want[i] {
					t.Errorf("parsePattern()[%d] = %q, want %q", i, tool, tt.want[i])
				}
			}
		})
	}
}

func TestMatchesSequence(t *testing.T) {
	toolCalls := []toolCallWithTurn{
		{toolName: "Read", turn: 0},
		{toolName: "Edit", turn: 1},
		{toolName: "Bash", turn: 2},
		{toolName: "Grep", turn: 3},
	}

	tests := []struct {
		name  string
		start int
		tools []string
		want  bool
	}{
		{
			name:  "matches at start",
			start: 0,
			tools: []string{"Read", "Edit", "Bash"},
			want:  true,
		},
		{
			name:  "matches at middle",
			start: 1,
			tools: []string{"Edit", "Bash"},
			want:  true,
		},
		{
			name:  "no match",
			start: 0,
			tools: []string{"Read", "Bash"},
			want:  false,
		},
		{
			name:  "out of bounds",
			start: 3,
			tools: []string{"Grep", "Read"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSequence(toolCalls, tt.start, tt.tools)
			if got != tt.want {
				t.Errorf("matchesSequence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindMinMaxTimestamps(t *testing.T) {
	tests := []struct {
		name       string
		timestamps []int64
		wantMin    int64
		wantMax    int64
	}{
		{
			name:       "empty slice",
			timestamps: []int64{},
			wantMin:    0,
			wantMax:    0,
		},
		{
			name:       "single timestamp",
			timestamps: []int64{100},
			wantMin:    100,
			wantMax:    100,
		},
		{
			name:       "multiple timestamps",
			timestamps: []int64{300, 100, 500, 200},
			wantMin:    100,
			wantMax:    500,
		},
		{
			name:       "already sorted",
			timestamps: []int64{100, 200, 300},
			wantMin:    100,
			wantMax:    300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax := findMinMaxTimestamps(tt.timestamps)
			if gotMin != tt.wantMin || gotMax != tt.wantMax {
				t.Errorf("findMinMaxTimestamps() = (%d, %d), want (%d, %d)", gotMin, gotMax, tt.wantMin, tt.wantMax)
			}
		})
	}
}
