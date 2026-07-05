package aprs

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"

	"github.com/szporwolik/cqops/internal/applog"
)

// KISSClient reads AX.25 frames from a KISS TNC connected via serial port.
// It satisfies the same interface as Client for use in app.App.
type KISSClient struct {
	mu      sync.Mutex
	port    io.ReadWriteCloser
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}

	// Configuration
	portName string
	baud     int
	dataBits int
	parity   serial.Parity
	stopBits serial.StopBits
	dtr      bool
	rts      bool

	// Callbacks
	OnPacket func(raw string)
	OnStatus func(connected bool, err error)
}

// NewKISSClient creates a KISS TNC client. Call Start() to connect.
func NewKISSClient(portName string, baud, dataBits int, parity serial.Parity, stopBits serial.StopBits, dtr, rts bool) *KISSClient {
	return &KISSClient{
		portName: portName,
		baud:     baud,
		dataBits: dataBits,
		parity:   parity,
		stopBits: stopBits,
		dtr:      dtr,
		rts:      rts,
	}
}

// Start opens the serial port and begins reading KISS frames.
// Non-blocking — connection runs in a goroutine. Use OnStatus for feedback.
func (k *KISSClient) Start() {
	k.mu.Lock()
	if k.running {
		k.mu.Unlock()
		return
	}
	k.running = true
	k.stopCh = make(chan struct{})
	k.doneCh = make(chan struct{})
	k.mu.Unlock()

	go k.runLoop()
}

// Stop disconnects and shuts down the read loop.
func (k *KISSClient) Stop() {
	k.mu.Lock()
	if !k.running {
		k.mu.Unlock()
		return
	}
	k.running = false
	close(k.stopCh)
	if k.port != nil {
		k.port.Close()
	}
	k.mu.Unlock()

	select {
	case <-k.doneCh:
	case <-time.After(3 * time.Second):
	}
	applog.Info("KISS: client stopped")
}

// IsRunning returns true when the client is active.
func (k *KISSClient) IsRunning() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.running
}

// IsConnected returns true when we have an active serial connection.
func (k *KISSClient) IsConnected() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.port != nil
}

// KISSServerClient connects to a KISS TNC over TCP (e.g. Dire Wolf, aprs_tnc).
// It implements the Client interface. KISS framing is identical to the serial
// KISSClient — only the transport differs.
type KISSServerClient struct {
	mu      sync.Mutex
	conn    net.Conn
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}

	addr string // "host:port"

	// Callbacks
	OnPacket func(raw string)
	OnStatus func(connected bool, err error)
}

// NewKISSServerClient creates a TCP KISS client. Call Start() to connect.
func NewKISSServerClient(addr string) *KISSServerClient {
	return &KISSServerClient{addr: addr}
}

// Start connects to the KISS TCP server and begins reading frames.
func (c *KISSServerClient) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	go c.runLoop()
}

// Stop disconnects and shuts down the read loop.
func (c *KISSServerClient) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stopCh)
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	select {
	case <-c.doneCh:
	case <-time.After(3 * time.Second):
	}
	applog.Info("KISS: server client stopped")
}

// IsRunning returns true when the client is active.
func (c *KISSServerClient) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// IsConnected returns true when we have an active TCP connection.
func (c *KISSServerClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// SendPosition transmits a position report over the TCP KISS connection.
func (c *KISSServerClient) SendPosition(callsign string, lat, lon float64, symbol, comment string) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("KISS server send: not connected")
	}

	body := formatUncompressedPosition(lat, lon, symbol, comment)
	tnc2 := fmt.Sprintf("%s>APRS,WIDE1-1,WIDE2-1:%s", strings.ToUpper(callsign), body)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("KISS server send: connection lost")
	}
	if err := sendKISSFrame(c.conn, tnc2); err != nil {
		applog.Error("KISS server: send position failed", "error", err)
		return fmt.Errorf("KISS server send: %w", err)
	}
	applog.Info("KISS server: position sent", "callsign", callsign, "lat", lat, "lon", lon)
	return nil
}

func (c *KISSServerClient) runLoop() {
	defer close(c.doneCh)

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", c.addr, 10*time.Second)
		if err != nil {
			applog.Debug("KISS server: cannot connect (retrying)", "addr", c.addr, "error", err)
			select {
			case <-c.stopCh:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		applog.Info("KISS server: connected", "addr", c.addr)
		if c.OnStatus != nil {
			c.OnStatus(true, nil)
		}

		// Read KISS frames from TCP connection.
		c.readFrames(conn)

		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		conn.Close()

		applog.Warn("KISS server: disconnected", "addr", c.addr)
		if c.OnStatus != nil {
			c.OnStatus(false, nil)
		}

		select {
		case <-c.stopCh:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *KISSServerClient) readFrames(conn net.Conn) {
	buf := make([]byte, 4096)
	frame := make([]byte, 0, 512)
	inFrame := false
	escape := false

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		for i := 0; i < n; i++ {
			b := buf[i]
			if b == 0xC0 {
				if inFrame && len(frame) > 0 {
					payload := frame
					if len(payload) > 1 && payload[0] == 0x00 {
						payload = payload[1:]
					}
					tnc2 := extractTNC2(payload)
					if tnc2 != "" {
						applog.Debug("KISS server: frame received", "len", fmt.Sprintf("%d", len(payload)), "tnc2Len", fmt.Sprintf("%d", len(tnc2)))
						if c.OnPacket != nil {
							c.OnPacket(tnc2)
						}
					}
				}
				frame = frame[:0]
				inFrame = true
				escape = false
				continue
			}
			if !inFrame {
				continue
			}
			if escape {
				escape = false
				switch b {
				case 0xDC:
					b = 0xC0
				case 0xDD:
					b = 0xDB
				}
				frame = append(frame, b)
				continue
			}
			if b == 0xDB {
				escape = true
				continue
			}
			frame = append(frame, b)
		}
	}
}

// SendPosition transmits an uncompressed APRS position report over the
// KISS TNC to RF. The packet uses a standard WIDE1-1,WIDE2-1 digipeater
// path. Returns an error if the TNC is not connected or the write fails.
func (k *KISSClient) SendPosition(callsign string, lat, lon float64, symbol, comment string) error {
	k.mu.Lock()
	port := k.port
	k.mu.Unlock()

	if port == nil {
		return fmt.Errorf("KISS send: not connected")
	}

	body := formatUncompressedPosition(lat, lon, symbol, comment)
	// RF path uses WIDE1-1,WIDE2-1 instead of TCPIP*
	tnc2 := fmt.Sprintf("%s>APRS,WIDE1-1,WIDE2-1:%s", strings.ToUpper(callsign), body)

	k.mu.Lock()
	defer k.mu.Unlock()
	if k.port == nil {
		return fmt.Errorf("KISS send: connection lost")
	}
	if err := sendKISSFrame(k.port, tnc2); err != nil {
		applog.Error("KISS: send position failed", "error", err)
		return fmt.Errorf("KISS send: %w", err)
	}
	applog.Info("KISS: position sent", "callsign", callsign, "lat", lat, "lon", lon)
	return nil
}

// sendKISSFrame builds an AX.25 UI frame from a TNC2 string, wraps it in
// KISS framing, and writes it to the serial port. The TNC adds FCS and
// handles HDLC bit-stuffing.
//
// TNC2 format:  SRCCALL>DEST,DIGI1,DIGI2:BODY
// AX.25 format: [dest 7B][src 7B][digi 7B...][ctrl 0x03][pid 0xF0][info]
//
// Each address is 6 bytes of callsign (padded, shifted left 1) + 1 byte
// with SSID<<1 | end-bit. The last address has bit 0 set to 1.
func sendKISSFrame(w io.Writer, tnc2 string) error {
	const fend byte = 0xC0
	const fesc byte = 0xDB
	const tfend byte = 0xDC
	const tfesc byte = 0xDD

	// Parse TNC2: "SRC>DST,DIGI1,DIGI2:BODY"
	headerEnd := strings.IndexByte(tnc2, ':')
	if headerEnd < 0 {
		return fmt.Errorf("KISS send: invalid TNC2, missing ':'")
	}
	header := tnc2[:headerEnd]
	info := tnc2[headerEnd+1:]

	// Split "SRC>DST,DIGI1,DIGI2"
	gt := strings.IndexByte(header, '>')
	if gt < 0 {
		return fmt.Errorf("KISS send: invalid TNC2, missing '>'")
	}
	src := header[:gt]
	rest := header[gt+1:]

	parts := strings.Split(rest, ",")
	if len(parts) == 0 {
		return fmt.Errorf("KISS send: invalid TNC2, no destination")
	}
	dst := parts[0]
	digis := parts[1:]

	// Build address list: [destination, source, digi1, digi2, ...]
	// Last address in the list gets bit 0 of the SSID byte set.
	addrs := make([]string, 0, 2+len(digis))
	addrs = append(addrs, dst)
	addrs = append(addrs, src)
	addrs = append(addrs, digis...)

	// Calculate frame size: 7 bytes per address + 2 (ctrl+pid) + info
	frameLen := len(addrs)*7 + 2 + len(info)
	frame := make([]byte, 0, frameLen)

	for idx, addr := range addrs {
		b := encodeAX25Addr(addr, idx == len(addrs)-1)
		frame = append(frame, b...)
	}
	frame = append(frame, 0x03) // UI frame
	frame = append(frame, 0xF0) // No layer 3
	frame = append(frame, []byte(info)...)

	// Wrap in KISS framing with escape handling.
	kissLen := 2 + len(frame) + 2 // FEND CMD ... FEND (worst case no escapes)
	kiss := make([]byte, 0, kissLen)
	kiss = append(kiss, fend, 0x00) // FEND + data command (port 0)

	for _, b := range frame {
		switch b {
		case fend:
			kiss = append(kiss, fesc, tfend)
		case fesc:
			kiss = append(kiss, fesc, tfesc)
		default:
			kiss = append(kiss, b)
		}
	}
	kiss = append(kiss, fend)

	_, err := w.Write(kiss)
	return err
}

// encodeAX25Addr encodes a callsign-SSID (e.g. "SP9MOA-9" or "WIDE1-1")
// into a 7-byte AX.25 address field. If last is true, bit 0 of byte 6 is
// set to mark the end of the address chain.
func encodeAX25Addr(addr string, last bool) []byte {
	b := make([]byte, 7)

	// Split callsign and SSID.
	cs := addr
	ssid := 0
	if dash := strings.LastIndexByte(addr, '-'); dash >= 0 {
		cs = addr[:dash]
		if n, err := fmt.Sscanf(addr[dash+1:], "%d", &ssid); err != nil || n != 1 {
			ssid = 0
		}
	}

	// Encode 6 bytes of callsign (shift left 1, upper case, space padded).
	for i := 0; i < 6; i++ {
		if i < len(cs) {
			c := cs[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			b[i] = c << 1
		} else {
			b[i] = ' ' << 1 // space fill
		}
	}

	// SSID byte: SSID in bits 1-4, bit 0 = end marker.
	b[6] = byte(ssid&0x0F) << 1
	if last {
		b[6] |= 1
	}
	// Also set bits 5-6 (always 1 in standard AX.25).
	b[6] |= 0x60

	return b
}

func (k *KISSClient) runLoop() {
	defer close(k.doneCh)

	mode := &serial.Mode{
		BaudRate: k.baud,
		DataBits: k.dataBits,
		Parity:   k.parity,
		StopBits: k.stopBits,
	}

	// Connect loop — retry on failure.
	isFirst := true
	suppressToast := true  // first failure: DEBUG only
	suppressToast2 := true // second failure: WARN log only, no toast
	for {
		select {
		case <-k.stopCh:
			return
		default:
		}

		// Brief pause on reconnect to let OS release a previous handle.
		if !isFirst {
			select {
			case <-k.stopCh:
				return
			case <-time.After(200 * time.Millisecond):
			}
		}

		p, err := serial.Open(k.portName, mode)
		isFirst = false
		if err != nil {
			// Suppress toast for the first two retry cycles (save-triggered
			// restart needs time for Windows to release the COM port).
			if suppressToast {
				applog.Debug("KISS: cannot open port (retrying)", "port", k.portName, "error", err)
				suppressToast = false // next failure will log Warn but no toast
			} else if suppressToast2 {
				applog.Warn("KISS: cannot open port", "port", k.portName, "error", err)
				suppressToast2 = false // follow-up failure will show toast
			} else {
				applog.Warn("KISS: cannot open port", "port", k.portName, "error", err)
				if k.OnStatus != nil {
					k.OnStatus(false, fmt.Errorf("cannot open %s: %v", k.portName, err))
				}
			}
			// Wait before retry.
			select {
			case <-k.stopCh:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		if k.dtr {
			_ = p.SetDTR(true)
		}
		if k.rts {
			_ = p.SetRTS(true)
		}

		k.mu.Lock()
		k.port = p
		k.mu.Unlock()

		applog.Info("KISS: connected", "port", k.portName, "baud", fmt.Sprintf("%d", k.baud))
		if k.OnStatus != nil {
			k.OnStatus(true, nil)
		}

		// Read KISS frames.
		k.readFrames(p)

		k.mu.Lock()
		if k.port == p {
			k.port = nil
		}
		k.mu.Unlock()
		p.Close()

		applog.Warn("KISS: disconnected", "port", k.portName)
		if k.OnStatus != nil {
			k.OnStatus(false, nil)
		}

		// Wait before reconnect.
		select {
		case <-k.stopCh:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// readFrames reads KISS frames from the serial port and feeds them to OnPacket.
// KISS frames are delimited by FEND (0xC0). The command byte (0x00 for data)
// is stripped, then the AX.25 header is parsed to extract the TNC2 payload.
func (k *KISSClient) readFrames(p io.ReadWriteCloser) {
	buf := make([]byte, 4096)
	frame := make([]byte, 0, 512)
	inFrame := false
	escape := false

	for {
		select {
		case <-k.stopCh:
			return
		default:
		}

		n, err := p.Read(buf)
		if err != nil {
			return // port closed or error — reconnect
		}

		for i := 0; i < n; i++ {
			b := buf[i]
			if b == 0xC0 { // FEND
				if inFrame && len(frame) > 0 {
					// Strip KISS command byte (first byte) if present.
					payload := frame
					if len(payload) > 1 && payload[0] == 0x00 {
						payload = payload[1:]
					}
					// Extract TNC2 payload from AX.25 frame.
					// AX.25 UI frame: dest(7) src(7) ctrl(1) pid(1) info...
					// The info field is the APRS TNC2 text (CALLSIGN>DEST,PATH:BODY).
					tnc2 := extractTNC2(payload)
					if tnc2 != "" {
						applog.Debug("KISS: frame received", "len", fmt.Sprintf("%d", len(payload)), "tnc2Len", fmt.Sprintf("%d", len(tnc2)))
						if k.OnPacket != nil {
							k.OnPacket(tnc2)
						}
					}
				}
				frame = frame[:0]
				inFrame = true
				escape = false
				continue
			}

			if !inFrame {
				continue
			}

			if escape {
				escape = false
				switch b {
				case 0xDC:
					b = 0xC0 // TFEND → FEND
				case 0xDD:
					b = 0xDB // TFESC → FESC
				}
				frame = append(frame, b)
				continue
			}

			if b == 0xDB { // FESC
				escape = true
				continue
			}

			frame = append(frame, b)
		}
	}
}

// extractTNC2 extracts the APRS TNC2-format payload from an AX.25 UI frame.
// Returns a TNC2-formatted string suitable for ParsePositionPacket:
//
//	SRCCALL>DEST,PATH:BODY
//
// If the frame is already in plain TNC2 format, it is returned as-is.
func extractTNC2(raw []byte) string {
	if len(raw) < 17 {
		return string(raw)
	}

	// Check if this looks like plain TNC2 (starts with a letter/digit, has '>').
	if (raw[0] >= 'A' && raw[0] <= 'Z') || (raw[0] >= '0' && raw[0] <= '9') {
		for i := 0; i < len(raw) && i < 20; i++ {
			if raw[i] == '>' {
				return string(raw)
			}
		}
	}

	// Parse AX.25 addresses to extract source callsign, destination,
	// and digipeater path.
	var addrs []string
	i := 0
	for i+7 <= len(raw) {
		addr := decodeAX25Addr(raw[i : i+7])
		if addr == "" {
			break
		}
		addrs = append(addrs, addr)
		lastByte := raw[i+6]
		i += 7
		if (lastByte & 1) == 1 {
			break // end of address chain
		}
	}

	if len(addrs) < 2 {
		// Can't decode addresses — return raw.
		return string(raw)
	}

	// Build TNC2 header: SRCCALL>DEST,digi1,digi2,...
	src := addrs[1] // source is second address
	dst := addrs[0] // destination is first
	path := strings.Join(addrs[2:], ",")

	// Find the info field after ctrl+pid.
	infoStart := i + 2 // skip ctrl+pid (0x03 + 0xF0 typically)
	if infoStart < len(raw) && raw[i] == 0x03 && raw[i+1] == 0xF0 {
		// standard UI frame
	} else if infoStart-1 < len(raw) && raw[i] == 0x03 {
		infoStart = i + 1 // no PID byte
	} else {
		infoStart = i // no ctrl/pid (unlikely)
	}

	body := string(raw[infoStart:])
	if body == "" {
		return string(raw)
	}

	// Reconstruct TNC2 line.
	tnc2 := src + ">" + dst
	if path != "" {
		tnc2 += "," + path
	}
	tnc2 += ":" + body

	applog.Debug("KISS: extracted info", "headerLen", fmt.Sprintf("%d", infoStart), "tnc2", tnc2[:min(len(tnc2), 80)])
	return tnc2
}

// decodeAX25Addr decodes a 7-byte AX.25 address field (6 chars + SSID).
// Returns the callsign-SSID string, e.g. "SP9MOA-9". Returns empty string
// if the field is invalid.
func decodeAX25Addr(b []byte) string {
	if len(b) < 7 {
		return ""
	}
	var cs [6]byte
	for i := 0; i < 6; i++ {
		cs[i] = b[i] >> 1 // shift right to get ASCII
		if cs[i] < ' ' || cs[i] > '~' {
			return ""
		}
	}
	callsign := strings.TrimRight(string(cs[:]), " ")
	if callsign == "" {
		return ""
	}
	ssid := b[6] >> 1 & 0x0F // lower 4 bits after shift
	if ssid > 0 {
		callsign += fmt.Sprintf("-%d", ssid)
	}
	return callsign
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
