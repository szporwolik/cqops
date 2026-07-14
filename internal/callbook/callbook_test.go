package callbook

import (
	"errors"
	"testing"
)

// --- mock provider ---

type mockProvider struct {
	name     string
	priority int
	data     *Result
	err      error
}

func (m *mockProvider) Lookup(callsign string) (*Result, error) { return m.data, m.err }
func (m *mockProvider) TestConnection() error                   { return nil }
func (m *mockProvider) Name() string                            { return m.name }
func (m *mockProvider) Priority() int                           { return m.priority }

// --- Registry tests ---

func TestRegistryEmpty(t *testing.T) {
	r := NewRegistry(nil)
	data, err := r.Lookup("SP9ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Error("expected nil result for empty registry")
	}
}

func TestRegistrySingleProvider(t *testing.T) {
	p := &mockProvider{
		name: "test", priority: 10,
		data: &Result{Callsign: "SP9ABC", Name: "Jan", Grid: "JO90", Provider: "test"},
	}
	r := NewRegistry([]Provider{p})
	data, err := r.Lookup("SP9ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data == nil || data.Name != "Jan" {
		t.Error("expected data from single provider")
	}
}

func TestRegistryProviderError(t *testing.T) {
	p := &mockProvider{
		name: "test", priority: 10,
		err: errors.New("timeout"),
	}
	r := NewRegistry([]Provider{p})
	_, err := r.Lookup("SP9ABC")
	if err == nil {
		t.Error("expected error from failing provider")
	}
}

func TestRegistryEmptyCall(t *testing.T) {
	p := &mockProvider{name: "test", priority: 10, data: &Result{Callsign: "SP9"}}
	r := NewRegistry([]Provider{p})
	data, err := r.Lookup("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Error("expected nil for empty call")
	}
}

func TestRegistryPriorityOrder_Cascade(t *testing.T) {
	// High-priority provider returns partial data.
	primary := &mockProvider{
		name: "primary", priority: 90,
		data: &Result{Callsign: "SP9ABC", Name: "Jan", Provider: "primary"},
	}
	// Low-priority provider fills in missing fields.
	secondary := &mockProvider{
		name: "secondary", priority: 50,
		data: &Result{Callsign: "SP9ABC", Grid: "JO90", Country: "Poland", Provider: "secondary"},
	}
	r := NewRegistry([]Provider{secondary, primary}) // inserted out of order — registry sorts
	data, err := r.Lookup("SP9ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data == nil {
		t.Fatal("expected data")
	}
	if data.Name != "Jan" {
		t.Errorf("name = %q, want Jan (from primary)", data.Name)
	}
	if data.Grid != "JO90" {
		t.Errorf("grid = %q, want JO90 (from secondary)", data.Grid)
	}
	if data.Country != "Poland" {
		t.Errorf("country = %q, want Poland (from secondary)", data.Country)
	}
	if data.Provider != "primary" {
		t.Errorf("provider = %q, want primary", data.Provider)
	}
}

func TestRegistryPriorityOrder_NoOverwrite(t *testing.T) {
	// Primary already has all data; secondary should not overwrite.
	primary := &mockProvider{
		name: "primary", priority: 90,
		data: &Result{Callsign: "SP9ABC", Name: "Jan", Grid: "JO90", Country: "Poland", Provider: "primary"},
	}
	secondary := &mockProvider{
		name: "secondary", priority: 50,
		data: &Result{Callsign: "SP9ABC", Name: "OVERWRITE", Grid: "XX00", Country: "Wrong", Provider: "secondary"},
	}
	r := NewRegistry([]Provider{primary, secondary})
	data, _ := r.Lookup("SP9ABC")
	if data.Name != "Jan" {
		t.Error("secondary should not overwrite primary's name")
	}
	if data.Grid != "JO90" {
		t.Error("secondary should not overwrite primary's grid")
	}
}

func TestRegistrySkipNilResult(t *testing.T) {
	// First provider returns nil (not found), second succeeds.
	p1 := &mockProvider{name: "p1", priority: 90, data: nil}
	p2 := &mockProvider{name: "p2", priority: 50, data: &Result{Callsign: "SP9ABC", Name: "Jan"}}
	r := NewRegistry([]Provider{p1, p2})
	data, _ := r.Lookup("SP9ABC")
	if data == nil || data.Name != "Jan" {
		t.Error("expected data from second provider when first returns nil")
	}
}

func TestRegistryLen(t *testing.T) {
	if got := NewRegistry(nil).Len(); got != 0 {
		t.Errorf("nil registry len = %d, want 0", got)
	}
	r := NewRegistry([]Provider{
		&mockProvider{name: "a", priority: 10},
		&mockProvider{name: "b", priority: 20},
	})
	if got := r.Len(); got != 2 {
		t.Errorf("registry len = %d, want 2", got)
	}
}

// --- mergeInto tests ---

func TestMergeInto_FillsBlanks(t *testing.T) {
	dst := &Result{Callsign: "SP9ABC", Name: "Jan"}
	src := &Result{Callsign: "SP9ABC", Name: "Ignored", Grid: "JO90", Country: "Poland"}
	mergeInto(dst, src)
	if dst.Grid != "JO90" {
		t.Errorf("grid not filled, got %q", dst.Grid)
	}
	if dst.Country != "Poland" {
		t.Errorf("country not filled, got %q", dst.Country)
	}
	if dst.Name != "Jan" {
		t.Error("existing name should not be overwritten")
	}
}

func TestMergeInto_AllFields(t *testing.T) {
	dst := &Result{Callsign: "SP9ABC"}
	src := &Result{
		Name: "Jan", Grid: "JO90", Country: "Poland", QTH: "Krakow",
		State: "MA", Zip: "30-001", County: "Malopolskie", Class: "A",
		Email: "a@b.com", URL: "https://x.com", Lat: "50", Lon: "19",
		DXCC: "269", CQZone: "15", ITUZone: "28", ImageURL: "https://img",
	}
	mergeInto(dst, src)
	if dst.Name != "Jan" || dst.Grid != "JO90" || dst.Country != "Poland" {
		t.Error("basic fields not filled")
	}
	if dst.QTH != "Krakow" || dst.State != "MA" || dst.Zip != "30-001" {
		t.Error("location fields not filled")
	}
	if dst.DXCC != "269" || dst.CQZone != "15" || dst.ITUZone != "28" {
		t.Error("zone fields not filled")
	}
	if dst.Lat != "50" || dst.Lon != "19" {
		t.Error("coord fields not filled")
	}
	if dst.ImageURL != "https://img" || dst.Email != "a@b.com" {
		t.Error("misc fields not filled")
	}
}
