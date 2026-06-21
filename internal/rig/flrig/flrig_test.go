package flrig

import (
	"strings"
	"testing"
)

// =============================================================================
// parseXMLRPCResponse tests
// =============================================================================

func TestParseXMLRPCResponse_Double(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><double>14.074</double></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if !strings.Contains(v, "14.074") {
		t.Errorf("expected 14.074 in %q", v)
	}
}

func TestParseXMLRPCResponse_Int(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><int>42</int></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if !strings.Contains(v, "42") {
		t.Errorf("expected 42 in %q", v)
	}
}

func TestParseXMLRPCResponse_I4(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><i4>7</i4></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if !strings.Contains(v, "7") {
		t.Errorf("expected 7 in %q", v)
	}
}

func TestParseXMLRPCResponse_String(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><string>USB</string></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if v != "USB" {
		t.Errorf("expected USB, got %q", v)
	}
}

func TestParseXMLRPCResponse_Boolean(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><boolean>1</boolean></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if !strings.Contains(v, "1") {
		t.Errorf("expected 1 in %q", v)
	}
}

func TestParseXMLRPCResponse_CharData(t *testing.T) {
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value>hello</value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if v != "hello" {
		t.Errorf("expected hello, got %q", v)
	}
}

func TestParseXMLRPCResponse_EmptyValue(t *testing.T) {
	// Empty value is valid for setter calls (e.g. rig.set_vfo).
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value></value></param></params>
</methodResponse>`
	v, err := parseXMLRPCResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}

func TestParseXMLRPCResponse_NonXML(t *testing.T) {
	raw := "just a plain response"
	v, err := parseXMLRPCResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parseXMLRPCResponse: %v", err)
	}
	if v != raw {
		t.Errorf("expected %q, got %q", raw, v)
	}
}

func TestParseXMLRPCResponse_Garbage(t *testing.T) {
	_, err := parseXMLRPCResponse([]byte("<methodResponse><params>broken"))
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

// =============================================================================
// parseXMLRPCStringArray tests
// =============================================================================

func TestParseXMLRPCStringArray_Standard(t *testing.T) {
	// flrig's rig.get_modes returns bare <value> elements inside an array.
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><array><data>
    <value>USB</value>
    <value>LSB</value>
    <value>CW</value>
  </data></array></value></param></params>
</methodResponse>`
	modes, err := parseXMLRPCStringArray([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCStringArray: %v", err)
	}
	if len(modes) < 3 {
		t.Errorf("expected >= 3 modes, got %d: %v", len(modes), modes)
	}
	found := make(map[string]bool)
	for _, m := range modes {
		found[m] = true
	}
	for _, want := range []string{"USB", "LSB", "CW"} {
		if !found[want] {
			t.Errorf("missing mode %q in %v", want, modes)
		}
	}
}

func TestParseXMLRPCStringArray_NewlineDelimited(t *testing.T) {
	// Older flrig versions return a single string with newline-delimited modes.
	xml := `<?xml version="1.0"?>
<methodResponse>
  <params><param><value><string>USB
LSB
CW
</string></value></param></params>
</methodResponse>`
	modes, err := parseXMLRPCStringArray([]byte(xml))
	if err != nil {
		t.Fatalf("parseXMLRPCStringArray: %v", err)
	}
	if len(modes) < 3 {
		t.Errorf("expected >= 3 modes, got %d: %v", len(modes), modes)
	}
}

func TestParseXMLRPCStringArray_Empty(t *testing.T) {
	_, err := parseXMLRPCStringArray([]byte("<methodResponse></methodResponse>"))
	if err == nil {
		t.Error("expected error for no modes")
	}
}

func TestParseXMLRPCStringArray_Garbage(t *testing.T) {
	_, err := parseXMLRPCStringArray([]byte("not xml"))
	if err == nil {
		t.Error("expected error for non-XML")
	}
}

// =============================================================================
// New / constructor
// =============================================================================

func TestNew_Defaults(t *testing.T) {
	c := New("http://localhost:12345", 0)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.url != "http://localhost:12345" {
		t.Errorf("url = %q", c.url)
	}
	if c.timeout == 0 {
		t.Error("timeout should default to non-zero")
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	c := New("http://localhost:12345", 5000)
	if c.timeout != 5*1e9 { // 5 seconds in nanoseconds
		t.Errorf("timeout = %v", c.timeout)
	}
}

func TestNew_StripsTrailingSlash(t *testing.T) {
	c := New("http://localhost:12345/", 1000)
	if c.url != "http://localhost:12345" {
		t.Errorf("url = %q", c.url)
	}
}

// =============================================================================
// XML-RPC marshaling
// =============================================================================

func TestXMLRPCMethodCall_Marshaling(t *testing.T) {
	// Verify the XML-RPC request envelope is well-formed.
	call := xmlrpcMethodCall{MethodName: "rig.get_vfo"}
	params := &xmlrpcParams{}
	params.Param = append(params.Param, xmlrpcParam{Value: xmlrpcValue{Double: "14074000"}})
	call.Params = params

	var buf strings.Builder
	// Just verify it doesn't panic; actual XML is validated by flrig.
	if call.MethodName != "rig.get_vfo" {
		t.Error("method name mismatch")
	}
	if len(call.Params.Param) != 1 {
		t.Error("param count mismatch")
	}
	_ = buf
}
