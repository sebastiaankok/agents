package github

import (
	"testing"
)

func TestParseBlockedBy(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []int
	}{
		{
			name: "single blocker",
			body: "Some description\n\nBlocked by #5",
			want: []int{5},
		},
		{
			name: "multiple blockers",
			body: "Blocked by #3\nBlocked by #7\nBlocked by #42",
			want: []int{3, 7, 42},
		},
		{
			name: "no blocker",
			body: "Just a description with no blockers",
			want: nil,
		},
		{
			name: "malformed line",
			body: "Blocked by #\nBlocked by abc\nBlocked by #5",
			want: []int{5},
		},
		{
			name: "mixed case",
			body: "BLOCKED BY #10\nblocked by #20\nBlocked By #30",
			want: []int{10, 20, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBlockedBy(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseBlockedBy() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseBlockedBy()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
