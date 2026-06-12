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
	"time"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig"
)

type Flrig struct {
	url       string
	timeout   time.Duration
}

func New(url string, timeoutMS int) *Flrig {
	d := time.Duration(timeoutMS) * time.Millisecond
	if d <= 0 {
		d = 2 * time.Second
	}
	return &Flrig{
		url:     strings.TrimSuffix(url, "/"),
		timeout: d,
	}
}

func (f *Flrig) Status(ctx context.Context) (rig.RigStatus, error) {
	rs := rig.RigStatus{
		Provider: "flrig",
	}

	freqHz, err := f.getFrequency(ctx)
	if err != nil {
		rs.Connected = false
		return rs, nil
	}
	rs.Connected = true
	rs.FrequencyHz = freqHz
	rs.FrequencyMHz = float64(freqHz) / 1_000_000.0

	mode, err := f.getMode(ctx)
	if err == nil {
		rs.RawMode = mode
		rs.Mode = qso.MapFlrigMode(mode)
	}

	pwr, err := f.getPower(ctx)
	if err == nil {
		rs.Power = pwr
	}

	if rs.FrequencyMHz > 0 {
		rs.Band = qso.DeriveBand(rs.FrequencyMHz)
	}

	return rs, nil
}

func (f *Flrig) getFrequency(ctx context.Context) (int64, error) {
	v, err := f.xmlrpcCall(ctx, "rig.get_vfo")
	if err != nil {
		return 0, err
	}
	var freq float64
	if _, scanErr := fmt.Sscanf(v, "%f", &freq); scanErr != nil {
		return 0, fmt.Errorf("parse frequency %q: %w", v, scanErr)
	}
	return int64(math.Round(freq)), nil
}

func (f *Flrig) getMode(ctx context.Context) (string, error) {
	v, err := f.xmlrpcCall(ctx, "rig.get_mode")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(v), nil
}

func (f *Flrig) getPower(ctx context.Context) (float64, error) {
	v, err := f.xmlrpcCall(ctx, "rig.get_power")
	if err != nil {
		return 0, err
	}
	var pwr float64
	if _, scanErr := fmt.Sscanf(v, "%f", &pwr); scanErr != nil {
		return 0, fmt.Errorf("parse power %q: %w", v, scanErr)
	}
	return pwr, nil
}

func (f *Flrig) xmlrpcCall(ctx context.Context, method string) (string, error) {
	body := fmt.Sprintf(
		`<?xml version="1.0"?><methodCall><methodName>%s</methodName></methodCall>`,
		method,
	)
	req, err := http.NewRequestWithContext(ctx, "POST", f.url+"/RPC2", strings.NewReader(body))
	if err != nil { return "", err }
	req.Header.Set("Content-Type", "text/xml")
	client := &http.Client{Timeout: f.timeout}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil { return "", err }
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
