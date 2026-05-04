package kitty

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestRenderInline_SmallImage(t *testing.T) {
	// Small image data that encodes to < 4096 base64 chars fits in one chunk
	pngData := make([]byte, 100)
	for i := range pngData {
		pngData[i] = byte(i % 256)
	}

	result := RenderInline(pngData, 10, 5)

	if !strings.HasPrefix(result, "\033_G") {
		t.Errorf("expected output to start with APC escape, got: %q", result[:min(len(result), 10)])
	}
	if !strings.Contains(result, "f=100") {
		t.Errorf("expected output to contain f=100, got: %q", result)
	}
	if !strings.Contains(result, "a=T") {
		t.Errorf("expected output to contain a=T, got: %q", result)
	}
	if !strings.Contains(result, "c=10") {
		t.Errorf("expected output to contain c=10 (cols), got: %q", result)
	}
	if !strings.Contains(result, "r=5") {
		t.Errorf("expected output to contain r=5 (rows), got: %q", result)
	}
	// Single chunk should not have m=1 or m=0
	if strings.Contains(result, "m=1") {
		t.Errorf("single-chunk output should not contain m=1, got: %q", result)
	}
	if strings.Contains(result, "m=0") {
		t.Errorf("single-chunk output should not contain m=0, got: %q", result)
	}
}

func TestRenderInline_LargeImage(t *testing.T) {
	// Large enough that base64 encoding exceeds 4096 chars per chunk
	// base64 encodes 3 bytes as 4 chars, so we need > 4096*3/4 = 3072 bytes
	pngData := make([]byte, 10000)
	for i := range pngData {
		pngData[i] = byte(i % 256)
	}

	result := RenderInline(pngData, 20, 10)

	// Verify the b64 content is indeed multi-chunk
	b64 := base64.StdEncoding.EncodeToString(pngData)
	if len(b64) <= 4096 {
		t.Fatalf("test data did not produce multiple chunks (b64 len=%d)", len(b64))
	}

	// First chunk should have m=1 (more data follows)
	firstChunkEnd := strings.Index(result, "\033\\")
	if firstChunkEnd < 0 {
		t.Fatal("could not find APC terminator in output")
	}
	firstChunk := result[:firstChunkEnd]
	if !strings.Contains(firstChunk, "m=1") {
		t.Errorf("first chunk should contain m=1, got: %q", firstChunk)
	}

	// Last chunk should have m=0 (no more data)
	lastChunkStart := strings.LastIndex(result, "\033_G")
	if lastChunkStart < 0 {
		t.Fatal("could not find last APC sequence in output")
	}
	lastChunk := result[lastChunkStart:]
	if !strings.Contains(lastChunk, "m=0") {
		t.Errorf("last chunk should contain m=0, got: %q", lastChunk)
	}
}

func TestChunkString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		size     int
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			size:     4,
			expected: nil,
		},
		{
			name:     "fits in one chunk",
			input:    "abc",
			size:     4,
			expected: []string{"abc"},
		},
		{
			name:     "exact chunk size",
			input:    "abcd",
			size:     4,
			expected: []string{"abcd"},
		},
		{
			name:     "multiple chunks",
			input:    "abcdefgh",
			size:     3,
			expected: []string{"abc", "def", "gh"},
		},
		{
			name:     "many chunks",
			input:    "1234567890",
			size:     4,
			expected: []string{"1234", "5678", "90"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkString(tt.input, tt.size)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %d chunks, got %d: %v", len(tt.expected), len(got), got)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("chunk[%d]: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
			// Verify reassembly matches original
			if tt.input != "" {
				joined := strings.Join(got, "")
				if joined != tt.input {
					t.Errorf("reassembled chunks %q != original %q", joined, tt.input)
				}
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
