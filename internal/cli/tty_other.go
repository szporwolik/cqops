//go:build !linux
// +build !linux

package cli

import "io"

// linuxConsoleReader is a no-op on non-Linux platforms.
type linuxConsoleReader struct{ inner io.Reader }

func (r *linuxConsoleReader) Read(p []byte) (int, error) {
	return r.inner.Read(p)
}
