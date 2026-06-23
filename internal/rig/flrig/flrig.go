package flrig

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig"
)

type Client struct {
	url     string
	timeout time.Duration
	client  *http.Client
}

// Close drains idle HTTP connections so the transport can be garbage
// collected.  Satisfies the same interface as hamlib.Client.Close() so
// refreshRigClient can clean up either backend uniformly.
func (f *Client) Close() error {
	f.client.CloseIdleConnections()
	return nil
}

// New creates a new flrig HTTP client. url is the base URL of the flrig
// XML-RPC endpoint (e.g. "http://localhost:12345"). timeoutMS is the
// request timeout in milliseconds; values <= 0 default to 2 seconds.
func New(url string, timeoutMS int) *Client {
	d := time.Duration(timeoutMS) * time.Millisecond
	if d <= 0 {
		d = 2 * time.Second
	}
	return &Client{
		url:     strings.TrimSuffix(url, "/"),
		timeout: d,
		client:  &http.Client{Timeout: d},
	}
}

func (f *Client) Status(ctx context.Context) (rig.RigStatus, error) {
	rs := rig.RigStatus{
		Provider: "flrig",
	}

	var (
		freqHz   int64
		freqRxHz int64
		split    bool
		mode     string
		pwr      float64
		wg       sync.WaitGroup
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		v, err := f.getFrequency(ctx)
		if err != nil {
			applog.Debug("flrig: get_vfo failed", "error", err)
			return
		}
		freqHz = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getFrequencyB(ctx)
		if err != nil {
			applog.Debug("flrig: get_vfoB failed", "error", err)
			return
		}
		freqRxHz = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getSplit(ctx)
		if err != nil {
			applog.Debug("flrig: get_split failed", "error", err)
			return
		}
		split = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getMode(ctx)
		if err != nil {
			applog.Debug("flrig: get_mode failed", "error", err)
			return
		}
		mode = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getPower(ctx)
		if err != nil {
			applog.Debug("flrig: get_power failed", "error", err)
			return
		}
		pwr = v
	}()

	wg.Wait()

	if freqHz == 0 {
		applog.Debug("flrig: status — no VFO A freq, treating as disconnected")
		rs.Connected = false
		return rs, nil
	}

	rs.Connected = true
	rs.FrequencyHz = freqHz
	rs.FrequencyMHz = float64(freqHz) / 1_000_000.0
	rs.Split = split
	// Always report VFO B when split is active — the two VFOs can
	// briefly land on the same frequency during A/B swaps or when
	// split is first engaged.  Dropping it breaks split tracking.
	if freqRxHz > 0 {
		rs.FrequencyRxHz = freqRxHz
		rs.FrequencyRxMHz = float64(freqRxHz) / 1_000_000.0
	}

	if mode != "" {
		rs.RawMode = mode
		rs.Mode = qso.NormalizeRigMode(mode)
	}

	rs.Power = pwr

	if rs.FrequencyMHz > 0 {
		rs.Band = qso.DeriveBand(rs.FrequencyMHz)
	}

	applog.Debug("flrig: status",
		"freq_mhz", fmt.Sprintf("%.6f", rs.FrequencyMHz),
		"freq_rx_mhz", fmt.Sprintf("%.6f", rs.FrequencyRxMHz),
		"vfoB_hz", freqRxHz,
		"split", split,
		"mode", mode,
		"power", fmt.Sprintf("%.0f", pwr),
	)

	return rs, nil
}

func (f *Client) getFrequency(ctx context.Context) (int64, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.get_vfo")
	if err != nil {
		return 0, err
	}
	var freq float64
	if _, scanErr := fmt.Sscanf(v, "%f", &freq); scanErr != nil {
		return 0, fmt.Errorf("parse frequency %q: %w", v, scanErr)
	}
	return int64(math.Round(freq)), nil
}

// getSplit queries flrig for the split state via rig.get_split.
func (f *Client) getSplit(ctx context.Context) (bool, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcI4, "rig.get_split")
	if err != nil {
		return false, err
	}
	var val int
	if _, scanErr := fmt.Sscanf(v, "%d", &val); scanErr != nil {
		return false, fmt.Errorf("parse split %q: %w", v, scanErr)
	}
	return val != 0, nil
}

// getFrequencyB queries flrig for the VFO B (RX) frequency via rig.get_vfoB.
func (f *Client) getFrequencyB(ctx context.Context) (int64, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.get_vfoB")
	if err != nil {
		return 0, err
	}
	var freq float64
	if _, scanErr := fmt.Sscanf(v, "%f", &freq); scanErr != nil {
		return 0, fmt.Errorf("parse vfoB frequency %q: %w", v, scanErr)
	}
	return int64(math.Round(freq)), nil
}

// SetFrequency tunes the rig VFO to the given frequency in Hz via flrig XML-RPC.
func (f *Client) SetFrequency(ctx context.Context, freqHz int64) error {
	_, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.set_vfo", fmt.Sprintf("%d", freqHz))
	return err
}

// GetModes returns the list of available mode names from flrig.
// The returned slice is ordered by flrig's mode table index.
func (f *Client) GetModes(ctx context.Context) ([]string, error) {
	// Use the dedicated array parser — flrig returns rig.get_modes as an
	// XML-RPC array of strings, which the single-value parseXMLRPCResponse
	// cannot handle.
	return f.getModesArray(ctx)
}

// getModesArray calls rig.get_modes and parses the XML-RPC array response
// directly, bypassing the single-value parseXMLRPCResponse.
func (f *Client) getModesArray(ctx context.Context) ([]string, error) {
	call := xmlrpcMethodCall{MethodName: "rig.get_modes"}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(call); err != nil {
		return nil, fmt.Errorf("encode xmlrpc: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", f.url+"/RPC2", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseXMLRPCStringArray(data)
}

// parseXMLRPCStringArray extracts a flat list of strings from an XML-RPC
// array response. Handles both <array><data><value>X</value>... (flrig's
// actual format) and <value><string>X</string>...</value> patterns.
func parseXMLRPCStringArray(data []byte) ([]string, error) {
	raw := strings.TrimSpace(string(data))

	// ── Single newline-delimited string (older flrig versions) ──
	if idx := strings.Index(raw, "<string>"); idx >= 0 {
		content := raw[idx+8:]
		if endIdx := strings.Index(content, "</string>"); endIdx >= 0 {
			val := strings.TrimSpace(content[:endIdx])
			if val != "" && strings.Contains(val, "\n") {
				return strings.Split(val, "\n"), nil
			}
		}
	}

	// ── XML-RPC array of bare <value> elements (flrig's actual format) ──
	dec := xml.NewDecoder(bytes.NewReader(data))
	inValue := false
	var result []string
	var cur strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "value", "string":
				inValue = true
				cur.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "value", "string":
				inValue = false
				s := strings.TrimSpace(cur.String())
				if s != "" {
					result = append(result, s)
				}
			}
		case xml.CharData:
			if inValue {
				cur.Write(t)
			}
		}
	}

	if len(result) > 0 {
		return result, nil
	}
	return nil, fmt.Errorf("no modes found in response")
}

// SetMode sets the rig operating mode by name (e.g. "LSB", "USB", "CW-L").
// flrig's rig.set_mode expects the mode name string, not a numeric index.
func (f *Client) SetMode(ctx context.Context, mode string) error {
	_, err := f.xmlrpcCall(ctx, xmlrpcBare, "rig.set_mode", mode)
	return err
}

func (f *Client) getMode(ctx context.Context) (string, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.get_mode")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(v), nil
}

func (f *Client) getPower(ctx context.Context) (float64, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.get_power")
	if err != nil {
		return 0, err
	}
	var pwr float64
	if _, scanErr := fmt.Sscanf(v, "%f", &pwr); scanErr != nil {
		return 0, fmt.Errorf("parse power %q: %w", v, scanErr)
	}
	return pwr, nil
}

// Power returns the current RF power setting (watts). Satisfies the rig.Rig interface.
func (f *Client) Power(ctx context.Context) (float64, error) {
	return f.getPower(ctx)
}

// GetName returns the rig model name from flrig (e.g. "FT-DX10", "FTDX10").
func (f *Client) GetName(ctx context.Context) (string, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcBare, "rig.get_xcvr")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(v), nil
}

// xmlrpcMethodCall is the XML-RPC request envelope.
type xmlrpcMethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     *xmlrpcParams `xml:"params,omitempty"`
}

type xmlrpcParams struct {
	Param []xmlrpcParam `xml:"param"`
}

type xmlrpcParam struct {
	Value xmlrpcValue `xml:"value"`
}

type xmlrpcValue struct {
	CharData string `xml:",chardata"`
	Double   string `xml:"double,omitempty"`
	I4       string `xml:"i4,omitempty"`
	Int      string `xml:"int,omitempty"`
}

// xmlrpcValueType selects the XML-RPC value element.
type xmlrpcValueType int

const (
	xmlrpcDouble xmlrpcValueType = iota
	xmlrpcI4
	xmlrpcBare // bare chardata, no type tag — flrig expects this for set_mode
)

// xmlrpcCall builds a properly marshaled XML-RPC request and returns the
// response body as a string.  Value type selects the XML-RPC element:
// Double for frequencies/power, I4 for split state, Bare for mode names.
func (f *Client) xmlrpcCall(ctx context.Context, vt xmlrpcValueType, method string, params ...string) (string, error) {
	t0 := time.Now()
	call := xmlrpcMethodCall{MethodName: method}
	if len(params) > 0 {
		call.Params = &xmlrpcParams{}
		for _, p := range params {
			v := xmlrpcValue{}
			switch vt {
			case xmlrpcI4:
				v.I4 = p
			case xmlrpcBare:
				v.CharData = p
			default:
				v.Double = p
			}
			call.Params.Param = append(call.Params.Param, xmlrpcParam{Value: v})
		}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(call); err != nil {
		return "", fmt.Errorf("encode xmlrpc: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", f.url+"/RPC2", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml")
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	result, err := parseXMLRPCResponse(data)
	elapsed := time.Since(t0)
	if err != nil {
		applog.Debug("flrig: rpc error", "method", method, "elapsed", elapsed, "error", err)
	} else {
		applog.Debug("flrig: rpc ok", "method", method, "elapsed", elapsed)
	}
	return result, err
}

type rpcResponse struct {
	XMLName xml.Name `xml:"methodResponse"`
	Params  struct {
		Param struct {
			Value struct {
				CharData string  `xml:",chardata"`
				String   string  `xml:"string"`
				Double   float64 `xml:"double"`
				Int      int64   `xml:"int"`
				I4       int64   `xml:"i4"`
				Boolean  int64   `xml:"boolean"`
			} `xml:"value"`
		} `xml:"param"`
	} `xml:"params"`
}

func parseXMLRPCResponse(data []byte) (string, error) {
	raw := strings.TrimSpace(string(data))

	if !strings.HasPrefix(raw, "<?xml") && !strings.HasPrefix(raw, "<methodResponse") {
		return raw, nil
	}

	var r rpcResponse
	decoder := xml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&r); err != nil {
		return "", fmt.Errorf("xml parse: %w", err)
	}

	v := r.Params.Param.Value

	// Empty <value></value> is a valid success response for setter calls
	// (e.g. rig.set_vfo, rig.set_mode). All fields are zero/empty.
	if v.CharData == "" && v.String == "" && v.Double == 0 && v.Int == 0 && v.I4 == 0 && v.Boolean == 0 &&
		!strings.Contains(string(data), "<string>") &&
		!strings.Contains(string(data), "<double>") &&
		!strings.Contains(string(data), "<int>") &&
		!strings.Contains(string(data), "<i4>") &&
		!strings.Contains(string(data), "<boolean>") {
		return "", nil
	}

	if v.CharData != "" {
		return strings.TrimSpace(v.CharData), nil
	}

	if v.String != "" {
		return strings.TrimSpace(v.String), nil
	}

	if v.Double != 0 || strings.Contains(string(data), "<double>") {
		if v.Double == 0 {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "<double>") {
					start := strings.Index(line, "<double>") + 8
					end := strings.Index(line, "</double>")
					if start > 0 && end > start {
						var f float64
						if _, err := fmt.Sscanf(line[start:end], "%f", &f); err == nil {
							return fmt.Sprintf("%f", f), nil
						}
					}
				}
			}
		}
		return fmt.Sprintf("%f", v.Double), nil
	}
	if v.Int != 0 || v.I4 != 0 || strings.Contains(string(data), "<int>") || strings.Contains(string(data), "<i4>") {
		val := v.Int
		if v.I4 != 0 {
			val = v.I4
		}
		return fmt.Sprintf("%d", val), nil
	}
	if v.Boolean != 0 || strings.Contains(string(data), "<boolean>") {
		return fmt.Sprintf("%d", v.Boolean), nil
	}

	raw = string(data)
	if strings.Contains(raw, "<fault>") {
		return "", fmt.Errorf("flrig fault: %s", raw)
	}

	return "", fmt.Errorf("unexpected response: %s", raw)
}
