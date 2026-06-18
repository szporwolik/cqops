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

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig"
)

type Client struct {
	url     string
	timeout time.Duration
	client  *http.Client
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
		freqHz int64
		mode   string
		pwr    float64
		wg     sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		v, err := f.getFrequency(ctx)
		if err != nil {
			return
		}
		freqHz = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getMode(ctx)
		if err != nil {
			return
		}
		mode = v
	}()

	go func() {
		defer wg.Done()
		v, err := f.getPower(ctx)
		if err != nil {
			return
		}
		pwr = v
	}()

	wg.Wait()

	if freqHz == 0 {
		rs.Connected = false
		return rs, nil
	}

	rs.Connected = true
	rs.FrequencyHz = freqHz
	rs.FrequencyMHz = float64(freqHz) / 1_000_000.0

	if mode != "" {
		rs.RawMode = mode
		rs.Mode = qso.MapFlrigMode(mode)
	}

	rs.Power = pwr

	if rs.FrequencyMHz > 0 {
		rs.Band = qso.DeriveBand(rs.FrequencyMHz)
	}

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

// SetFrequency tunes the rig VFO to the given frequency in Hz via flrig XML-RPC.
// SetFrequency sets the VFO frequency in Hz.
func (f *Client) SetFrequency(ctx context.Context, freqHz int64) error {
	_, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.set_vfo", fmt.Sprintf("%d", freqHz))
	return err
}

// GetModes returns the list of available mode names from flrig.
// The returned slice is ordered by flrig's mode table index.
func (f *Client) GetModes(ctx context.Context) ([]string, error) {
	v, err := f.xmlrpcCall(ctx, xmlrpcDouble, "rig.get_modes")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(v) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(v), "\n"), nil
}

// SetMode sets the rig operating mode by index into flrig's mode table.
// Use GetModes to obtain the mode table first.
func (f *Client) SetMode(ctx context.Context, modeIdx int) error {
	_, err := f.xmlrpcCall(ctx, xmlrpcI4, "rig.set_mode", fmt.Sprintf("%d", modeIdx))
	if err != nil {
		return err
	}
	return nil
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
	Double string `xml:"double,omitempty"`
	I4     string `xml:"i4,omitempty"`
	Int    string `xml:"int,omitempty"`
}

// xmlrpcValueType selects the XML-RPC value element.
type xmlrpcValueType int

const (
	xmlrpcDouble xmlrpcValueType = iota
	xmlrpcI4
)

// xmlrpcCall builds a properly marshaled XML-RPC request and returns the
// response body as a string. All parameters are encoded with the given
// value type (Double for frequencies, I4 for mode indices).
func (f *Client) xmlrpcCall(ctx context.Context, vt xmlrpcValueType, method string, params ...string) (string, error) {
	call := xmlrpcMethodCall{MethodName: method}
	if len(params) > 0 {
		call.Params = &xmlrpcParams{}
		for _, p := range params {
			v := xmlrpcValue{}
			switch vt {
			case xmlrpcI4:
				v.I4 = p
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
