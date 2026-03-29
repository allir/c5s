package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeCwd(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"absolute path", "/Users/allir/allir/c5s", "-Users-allir-allir-c5s"},
		{"path with hyphens", "/Users/allir/my-project", "-Users-allir-my-project"},
		{"empty", "", ""},
		{"simple", "foo", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeCwd(tt.path)
			if got != tt.want {
				t.Errorf("encodeCwd(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsProcessAlive(t *testing.T) {
	// Our own process should definitely be alive
	if !isProcessAlive(os.Getpid()) {
		t.Error("isProcessAlive(os.Getpid()) = false, want true")
	}

	// A ludicrously high PID should not be alive
	if isProcessAlive(999999999) {
		t.Error("isProcessAlive(999999999) = true, want false")
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name    string
		content any
		want    string
	}{
		{
			name:    "plain string",
			content: "hello world",
			want:    "hello world",
		},
		{
			name:    "long string not truncated under limit",
			content: "this is a very long string that definitely exceeds sixty characters and should be truncated",
			want:    "this is a very long string that definitely exceeds sixty characters and should be truncated",
		},
		{
			name: "content block array with text",
			content: []any{
				map[string]any{"type": "text", "text": "extracted text"},
			},
			want: "extracted text",
		},
		{
			name: "content block array with no text type",
			content: []any{
				map[string]any{"type": "image", "url": "https://example.com/img.png"},
			},
			want: "",
		},
		{
			name:    "nil content",
			content: nil,
			want:    "",
		},
		{
			name:    "unexpected type",
			content: 42,
			want:    "",
		},
		{
			name:    "empty array",
			content: []any{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextContent(tt.content)
			if got != tt.want {
				t.Errorf("extractTextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string no truncation",
			input:  "hello",
			maxLen: 60,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "abcdef",
			maxLen: 6,
			want:   "abcdef",
		},
		{
			name:   "long string truncated",
			input:  "abcdefghij",
			maxLen: 8,
			want:   "abcde...",
		},
		{
			name:   "leading XML tag stripped",
			input:  "<context>actual content</context>",
			maxLen: 60,
			want:   "",
		},
		{
			name:   "leading XML tag with content after",
			input:  "<context>inner</context>actual content here",
			maxLen: 60,
			want:   "actual content here",
		},
		{
			name:   "nested XML tags stripped",
			input:  "<outer>stuff</outer><inner>more</inner>the real content",
			maxLen: 60,
			want:   "the real content",
		},
		{
			name:   "newlines collapsed to spaces",
			input:  "line one\nline two\nline three",
			maxLen: 60,
			want:   "line one line two line three",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 60,
			want:   "",
		},
		{
			name:   "tag with attributes not stripped",
			input:  "<div class='x'>content</div>",
			maxLen: 60,
			want:   "<div class='x'>content</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestScanReal(t *testing.T) {
	configDir := DefaultConfigDir()
	sessionsDir := filepath.Join(configDir, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Skip("skipping: no Claude sessions directory found")
	}

	sessions, _, err := Scan(configDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	t.Logf("Found %d live sessions", len(sessions))
	for i, s := range sessions {
		if i >= 5 {
			break
		}
		if s.PID <= 0 {
			t.Errorf("session %d has PID=%d, want > 0", i, s.PID)
		}
		t.Logf("  [%d] PID=%d Project=%s Branch=%s Status=%s Modified=%s Summary=%.40s",
			i, s.PID, s.Project, s.GitBranch, s.Status, s.LastModified.Format("15:04"), s.Summary)
	}
}
