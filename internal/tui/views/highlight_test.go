package views

import (
	"testing"
)

func TestCharDiff(t *testing.T) {
	tests := []struct {
		name           string
		oldLine        string
		newLine        string
		wantOldCount   int
		wantNewCount   int
		wantOldChanged []bool // changed flag for each region
		wantNewChanged []bool
	}{
		{
			name:           "insertion",
			oldLine:        "hello world",
			newLine:        "hello Go world",
			wantOldCount:   2, // "hello " (unchanged), "world" (unchanged)
			wantNewCount:   3, // "hello " (unchanged), "Go " (changed), "world" (unchanged)
			wantOldChanged: []bool{false, false},
			wantNewChanged: []bool{false, true, false},
		},
		{
			name:         "identical lines",
			oldLine:      "no change",
			newLine:      "no change",
			wantOldCount: 0,
			wantNewCount: 0,
		},
		{
			name:           "completely different",
			oldLine:        "aaa",
			newLine:        "bbb",
			wantOldCount:   1,
			wantNewCount:   1,
			wantOldChanged: []bool{true},
			wantNewChanged: []bool{true},
		},
		{
			name:           "prefix change",
			oldLine:        "foo bar baz",
			newLine:        "qux bar baz",
			wantOldCount:   2,
			wantNewCount:   2,
			wantOldChanged: []bool{true, false},
			wantNewChanged: []bool{true, false},
		},
		{
			name:           "suffix change",
			oldLine:        "foo bar baz",
			newLine:        "foo bar qux",
			wantOldCount:   2,
			wantNewCount:   2,
			wantOldChanged: []bool{false, true},
			wantNewChanged: []bool{false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldR, newR := charDiff(tt.oldLine, tt.newLine)
			if len(oldR) != tt.wantOldCount {
				t.Errorf("old regions: got %d, want %d: %+v", len(oldR), tt.wantOldCount, oldR)
			}
			if len(newR) != tt.wantNewCount {
				t.Errorf("new regions: got %d, want %d: %+v", len(newR), tt.wantNewCount, newR)
			}
			for i, r := range oldR {
				if i < len(tt.wantOldChanged) && r.changed != tt.wantOldChanged[i] {
					t.Errorf("old region %d changed = %v, want %v", i, r.changed, tt.wantOldChanged[i])
				}
			}
			for i, r := range newR {
				if i < len(tt.wantNewChanged) && r.changed != tt.wantNewChanged[i] {
					t.Errorf("new region %d changed = %v, want %v", i, r.changed, tt.wantNewChanged[i])
				}
			}
		})
	}
}

func TestPairChangedLines(t *testing.T) {
	tests := []struct {
		name  string
		parts []diffParts
		want  [][2]int
	}{
		{
			name: "simple delete-insert pair",
			parts: []diffParts{
				{marker: '-', code: "hello world"},
				{marker: '+', code: "hello Go world"},
			},
			want: [][2]int{{0, 1}},
		},
		{
			name: "similarity-based pairing skips inserted lines",
			parts: []diffParts{
				{marker: '-', code: "ColorDiffAddFg    color.Color"},
				{marker: '-', code: "ColorDiffRemoveFg color.Color"},
				{marker: '+', code: "ColorDiffAddFg       color.Color"},
				{marker: '+', code: "ColorDiffAddInlineBg color.Color"},
				{marker: '+', code: "ColorDiffRemoveFg       color.Color"},
			},
			want: [][2]int{{0, 2}, {1, 4}},
		},
		{
			name: "context breaks runs",
			parts: []diffParts{
				{marker: '-', code: "old line"},
				{marker: '+', code: "new line"},
				{marker: ' ', code: "context"},
				{marker: '-', code: "old two"},
				{marker: '+', code: "new two"},
			},
			want: [][2]int{{0, 1}, {3, 4}},
		},
		{
			name: "only deletes — no pairs",
			parts: []diffParts{
				{marker: '-', code: "a"},
				{marker: '-', code: "b"},
			},
			want: nil,
		},
		{
			name: "completely dissimilar — no pairs",
			parts: []diffParts{
				{marker: '-', code: "aaaaaa"},
				{marker: '+', code: "zzzzzz"},
			},
			want: nil,
		},
		{
			name: "all context — no pairs",
			parts: []diffParts{
				{marker: ' ', code: "ctx"},
				{marker: ' ', code: "ctx"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pairChangedLines(tt.parts)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d pairs, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("pair %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDiffPrefix(t *testing.T) {
	tests := []struct {
		line       string
		wantMarker byte
		wantCode   string
	}{
		{" 29 - old code", '-', "old code"},
		{" 30 + new code", '+', "new code"},
		{" 28   context", ' ', "context"},
		{"  - removed", '-', "removed"},
		{"  + added", '+', "added"},
		{"    context", ' ', "context"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			marker, _, code := diffPrefix(tt.line)
			if marker != tt.wantMarker {
				t.Errorf("marker = %q, want %q", marker, tt.wantMarker)
			}
			if code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}
