package kitty

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// Detect checks if the terminal supports the Kitty graphics protocol
// by sending a query and checking for a response.
func Detect() bool {
	// Don't detect in non-TTY environments
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}

	// Send a minimal kitty graphics query
	fmt.Fprint(os.Stdout, "\033_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\033\\")
	os.Stdout.Sync()

	// Switch to raw mode to read response
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read response with timeout
	deadline := time.Now().Add(100 * time.Millisecond)
	buf := make([]byte, 256)
	for time.Now().Before(deadline) {
		os.Stdin.SetReadDeadline(deadline)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}
		if strings.Contains(string(buf[:n]), "_G") {
			return true
		}
	}
	return false
}

// RenderInline sends a PNG image to the terminal via Kitty graphics protocol.
// cols and rows control the display size in terminal cells.
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
