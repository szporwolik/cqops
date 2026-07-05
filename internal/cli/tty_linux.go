//go:build linux
// +build linux

package cli

import (
	"bytes"
	"io"
	"os"
)

// linuxConsoleReader wraps stdin and translates Linux console function-key
// escape sequences into xterm-compatible sequences that Bubble Tea v2
// recognises. On the Linux console (TERM=linux):
//
//	F1 → \e[[A    F2 → \e[[B    F3 → \e[[C    F4 → \e[[D    F5 → \e[[E
//
// These are misinterpreted by the input parser (the extra '[' looks like
// a CSI parameter, so F1 becomes "cursor up" and the final letter leaks
// through as a plain key). This reader rewrites them:
//
//	F1 → \eOP     F2 → \eOQ     F3 → \eOR     F4 → \eOS     F5 → \e[15~
//
// All other bytes are passed through unchanged.
type linuxConsoleReader struct {
	inner io.Reader
	buf   bytes.Buffer // leftover translated bytes
}

// Fd exposes the underlying file descriptor so Bubble Tea v2 can recognise
// the input as a TTY and put it in raw mode.
func (r *linuxConsoleReader) Fd() uintptr { return os.Stdin.Fd() }

// linuxFKey maps Linux-console F1–F5 sequences (3 bytes after ESC) to their
// xterm replacements (full replacement including ESC).
var linuxFKey = map[[3]byte][]byte{
	{0x5b, 0x5b, 0x41}: {0x1b, 0x4f, 0x50},             // \e[[A → \eOP  (F1)
	{0x5b, 0x5b, 0x42}: {0x1b, 0x4f, 0x51},             // \e[[B → \eOQ  (F2)
	{0x5b, 0x5b, 0x43}: {0x1b, 0x4f, 0x52},             // \e[[C → \eOR  (F3)
	{0x5b, 0x5b, 0x44}: {0x1b, 0x4f, 0x53},             // \e[[D → \eOS  (F4)
	{0x5b, 0x5b, 0x45}: {0x1b, 0x5b, 0x31, 0x35, 0x7e}, // \e[[E → \e[15~ (F5)
}

func (r *linuxConsoleReader) Read(p []byte) (int, error) {
	// Return buffered translated bytes first.
	if r.buf.Len() > 0 {
		return r.buf.Read(p)
	}

	n, err := r.inner.Read(p)
	if n <= 0 {
		return n, err
	}

	// Scan for \e[[X patterns (X ∈ A–E) and translate them.
	translated := translateLinuxFKeys(p[:n])
	if len(translated) > n {
		// Translated version is longer (e.g. F5 → 5 bytes from 4).
		// Return n bytes and buffer the remainder.
		written := copy(p, translated[:n])
		r.buf.Write(translated[n:])
		return written, err
	}
	// In-place replacement (same or shorter length).
	copy(p, translated)
	// If shorter, zero-fill the tail (we can't shrink p, but the caller
	// uses n, not len(p)).
	return n, err
}

// translateLinuxFKeys finds Linux-console F-key sequences in data and
// replaces them with xterm equivalents. Returns a new slice (possibly
// longer) with replacements applied.
func translateLinuxFKeys(data []byte) []byte {
	if len(data) < 4 {
		return data // too short to contain \e[[X
	}

	// Walk through looking for ESC (0x1b).
	var out bytes.Buffer
	out.Grow(len(data) + 8) // slight over-allocation for F5 expansion

	i := 0
	for i < len(data) {
		if data[i] == 0x1b && i+3 < len(data) {
			// Peek at the next 3 bytes.
			var seq [3]byte
			copy(seq[:], data[i+1:i+4])
			if repl, ok := linuxFKey[seq]; ok {
				out.Write(repl)
				i += 4
				continue
			}
		}
		out.WriteByte(data[i])
		i++
	}
	return out.Bytes()
}
