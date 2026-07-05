package gps

import (
	"bufio"
	"fmt"

	"github.com/szporwolik/cqops/internal/applog"
)

// SerialConfig holds the serial port parameters for a GPS receiver.
type SerialConfig struct {
	Port     string // e.g. "COM6" (Windows) or "/dev/ttyUSB0" (Linux)
	BaudRate int    // e.g. 115200
	DTR      bool   // enable Data Terminal Ready
	RTS      bool   // enable Request To Send
}

// SerialReader reads NMEA lines from a serial port.
type SerialReader struct {
	cfg     SerialConfig
	port    serialPort
	scanner *bufio.Scanner
}

// serialPort abstracts the platform-specific serial port implementation.
type serialPort interface {
	Close() error
	Read(p []byte) (int, error)
}

// NewSerialReader creates a SerialReader for the given configuration.
// The port is not opened until ReadLine is called (lazy open + auto-reconnect).
func NewSerialReader(cfg SerialConfig) *SerialReader {
	return &SerialReader{cfg: cfg}
}

// TryOpen attempts to open the serial port synchronously.
// Returns nil on success, or an error describing why the port is unavailable.
// Idempotent — if the port is already open, returns nil immediately.
func (r *SerialReader) TryOpen() error {
	if r.port != nil {
		return nil
	}
	return r.open()
}

// ReadLine returns the next NMEA line, or an error if the port is
// unavailable. Automatically opens the port on first call and
// reconnects on read errors.
func (r *SerialReader) ReadLine() (string, error) {
	if r.scanner == nil {
		if err := r.open(); err != nil {
			return "", err
		}
	}
	if !r.scanner.Scan() {
		err := r.scanner.Err()
		if err != nil {
			applog.Warn("GPS: read error — reconnecting", "port", r.cfg.Port, "error", err.Error())
		}
		r.close()
		return "", fmt.Errorf("GPS: port closed")
	}
	return r.scanner.Text(), nil
}

// openSerialPort is the function used to open a physical serial port.
// It is a package-level variable so tests can inject a fake port.
// Defaults to openSerial (in serial_port.go).
var openSerialPort = openSerial

func (r *SerialReader) open() error {
	p, err := openSerialPort(r.cfg.Port, r.cfg.BaudRate, r.cfg.DTR, r.cfg.RTS)
	if err != nil {
		return err
	}
	r.port = p
	r.scanner = bufio.NewScanner(p)
	r.scanner.Split(bufio.ScanLines)
	applog.Info("GPS: serial port opened",
		"port", r.cfg.Port,
		"baud", fmt.Sprintf("%d", r.cfg.BaudRate),
		"dtr", fmt.Sprintf("%v", r.cfg.DTR),
		"rts", fmt.Sprintf("%v", r.cfg.RTS),
	)
	return nil
}

func (r *SerialReader) close() {
	if r.port != nil {
		r.port.Close()
		r.port = nil
	}
	r.scanner = nil
	applog.Debug("GPS: serial port closed", "port", r.cfg.Port)
}

// Close closes the serial port.
func (r *SerialReader) Close() error {
	r.close()
	return nil
}
