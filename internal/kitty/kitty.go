package kitty

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// Detect checks if the terminal supports the Kitty graphics protocol.
func Detect() bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Fprint(os.Stdout, "\033_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\033\\")
	os.Stdout.Sync()

	type readResult struct {
		data []byte
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := os.Stdin.Read(buf)
		ch <- readResult{data: buf[:n], err: err}
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			return false
		}
		return strings.Contains(string(result.data), "_G")
	case <-time.After(200 * time.Millisecond):
		return false
	}
}

// Unicode placeholder character for Kitty graphics protocol.
// Terminals that support this replace U+10EEEE with image pixels.
const placeholder = '\U0010EEEE'

// Row/column diacritics for Kitty Unicode placeholders.
// These combining characters encode the row/column index of each image cell.
var diacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
	0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
}

// RenderImage returns a string that, when printed to a Kitty-compatible terminal,
// transmits image data and displays it using Unicode placeholders.
//
// The returned string contains:
//  1. Escape sequences to transmit the PNG data (invisible, processed by terminal)
//  2. Unicode placeholder characters (U+10EEEE) that the terminal replaces with
//     the image. These characters have measurable width, so lipgloss layout works.
//
// cols and rows control the display size in terminal cells.
func RenderImage(id uint32, pngData []byte, cols, rows int) string {
	var sb strings.Builder

	// Part 1: Transmit image data with virtual placement (invisible to layout)
	b64 := base64.StdEncoding.EncodeToString(pngData)
	chunks := chunkString(b64, 4096)
	for i, chunk := range chunks {
		if i == 0 && len(chunks) == 1 {
			// Single chunk: transmit + virtual placement in one command
			fmt.Fprintf(&sb, "\033_Gi=%d,f=100,t=d,a=T,U=1,c=%d,r=%d,q=2;%s\033\\", id, cols, rows, chunk)
		} else if i == 0 {
			fmt.Fprintf(&sb, "\033_Gi=%d,f=100,t=d,a=T,U=1,c=%d,r=%d,q=2,m=1;%s\033\\", id, cols, rows, chunk)
		} else if i == len(chunks)-1 {
			fmt.Fprintf(&sb, "\033_Gm=0;%s\033\\", chunk)
		} else {
			fmt.Fprintf(&sb, "\033_Gm=1;%s\033\\", chunk)
		}
	}

	// Part 2: Unicode placeholder grid
	// Encode image ID as RGB foreground color
	r := (id >> 16) & 0xFF
	g := (id >> 8) & 0xFF
	b := id & 0xFF
	fgStart := fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	fgEnd := "\033[39m"

	for row := 0; row < rows; row++ {
		sb.WriteString(fgStart)
		for col := 0; col < cols; col++ {
			sb.WriteRune(placeholder)
			if row < len(diacritics) {
				sb.WriteRune(diacritics[row])
			}
			if col < len(diacritics) {
				sb.WriteRune(diacritics[col])
			}
		}
		sb.WriteString(fgEnd)
		if row < rows-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// RenderInline is the legacy direct-placement renderer (no Unicode placeholders).
// Kept for reference but RenderImage should be preferred for TUI apps.
func RenderInline(pngData []byte, cols, rows int) string {
	b64 := base64.StdEncoding.EncodeToString(pngData)
	chunks := chunkString(b64, 4096)
	var sb strings.Builder
	for i, chunk := range chunks {
		if i == 0 && len(chunks) == 1 {
			fmt.Fprintf(&sb, "\033_Gf=100,a=T,t=d,c=%d,r=%d;%s\033\\", cols, rows, chunk)
		} else if i == 0 {
			fmt.Fprintf(&sb, "\033_Gf=100,a=T,t=d,c=%d,r=%d,m=1;%s\033\\", cols, rows, chunk)
		} else if i == len(chunks)-1 {
			fmt.Fprintf(&sb, "\033_Gm=0;%s\033\\", chunk)
		} else {
			fmt.Fprintf(&sb, "\033_Gm=1;%s\033\\", chunk)
		}
	}
	return sb.String()
}

func chunkString(s string, size int) []string {
	var chunks []string
	for len(s) > 0 {
		if len(s) < size {
			size = len(s)
		}
		chunks = append(chunks, s[:size])
		s = s[size:]
	}
	return chunks
}
