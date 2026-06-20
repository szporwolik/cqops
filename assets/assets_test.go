package assets

import (
	"bytes"
	"testing"
)

func TestWorldMap_NotEmpty(t *testing.T) {
	if len(WorldMap) == 0 {
		t.Fatal("WorldMap is empty — embed may have failed")
	}
}

func TestWorldMap_JPEGHeader(t *testing.T) {
	if len(WorldMap) < 4 {
		t.Fatal("WorldMap too short for JPEG header check")
	}
	// JPEG files start with FF D8 FF.
	if WorldMap[0] != 0xFF || WorldMap[1] != 0xD8 || WorldMap[2] != 0xFF {
		t.Errorf("WorldMap does not start with JPEG magic bytes: got %02X %02X %02X",
			WorldMap[0], WorldMap[1], WorldMap[2])
	}
}

func TestWorldMap_MinimumSize(t *testing.T) {
	// The map image should be at least 50 KB (a real world map).
	if len(WorldMap) < 50*1024 {
		t.Errorf("WorldMap size = %d bytes, expected at least 50 KB", len(WorldMap))
	}
}

func TestWorldMap_Readable(t *testing.T) {
	// Verify the embedded data can be read as a byte stream.
	r := bytes.NewReader(WorldMap)
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("failed to read WorldMap: %v", err)
	}
	if n == 0 {
		t.Error("read 0 bytes from WorldMap")
	}
}
