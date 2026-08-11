package device

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadMPU6050(t *testing.T) {
	path := filepath.Join("..", "..", "devmaps", "mpu6050.yaml")
	devMap, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if devMap.Device != "MPU6050" || devMap.Vendor != "TDK InvenSense" {
		t.Fatalf("metadata = %q, %q", devMap.Device, devMap.Vendor)
	}
	if want := []uint16{0x68, 0x69}; !slices.Equal(devMap.Addresses, want) {
		t.Fatalf("Addresses = %v, want %v", devMap.Addresses, want)
	}
	if want := []Fingerprint{{Reg: 0x75, Value: 0x68}}; !slices.Equal(devMap.Fingerprint, want) {
		t.Fatalf("Fingerprint = %v, want %v", devMap.Fingerprint, want)
	}
	if len(devMap.Registers) != 2 {
		t.Fatalf("Registers length = %d, want 2", len(devMap.Registers))
	}
	if got := devMap.Registers[0]; got.Name != "WHO_AM_I" || got.Addr != 0x75 || got.RW != "r" || got.Reset != 0x68 {
		t.Fatalf("WHO_AM_I = %#v", got)
	}
	if got := devMap.Registers[1]; got.Default != 0 || len(got.Bits) != 3 {
		t.Fatalf("CTRL1 = %#v", got)
	} else {
		if !slices.Equal(got.Bits[0].Bit, BitRange{7}) || !slices.Equal(got.Bits[2].Bit, BitRange{3, 4}) {
			t.Fatalf("CTRL1 bits = %#v", got.Bits)
		}
	}
	if len(devMap.Operations) != 1 || len(devMap.Operations[0].Steps) != 2 {
		t.Fatalf("Operations = %#v", devMap.Operations)
	}
	if got := devMap.Operations[0].Steps[0]; got.Type != "write" || got.Reg != 0x6B || !slices.Equal(got.Data, []byte{0x80}) {
		t.Fatalf("write step = %#v", got)
	}
	if got := devMap.Operations[0].Steps[1]; got.Type != "delay" || got.MS != 100 {
		t.Fatalf("delay step = %#v", got)
	}
}

func TestLoadDirSkipsInvalidAndOverridesByDevice(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a-device.yaml"), `
device: TEST
addresses: [0x10]
registers:
  - {name: OLD, addr: 0x00, rw: rw}
`)
	writeFile(t, filepath.Join(dir, "bad.yaml"), "device: [invalid\n")
	writeFile(t, filepath.Join(dir, "z-device.yaml"), `
device: TEST
addresses: [0x11]
registers:
  - {name: NEW, addr: 0x01, rw: r}
`)

	index, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}
	warnings := index.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("Warnings() length = %d, want 1", len(warnings))
	}
	if warnings[0].Path != filepath.Join(dir, "bad.yaml") {
		t.Fatalf("warning path = %q, want %q", warnings[0].Path, filepath.Join(dir, "bad.yaml"))
	}
	devMap, ok := index.Get("z-device")
	if !ok || devMap.Device != "TEST" || devMap.Registers[0].Name != "NEW" {
		t.Fatalf("Get(z-device) = %#v, %v", devMap, ok)
	}
	if _, ok := index.Get("a-device"); ok {
		t.Fatal("old duplicate device remained in index")
	}
	if got := index.Lookup("TEST"); got != devMap {
		t.Fatalf("Lookup(TEST) = %#v, want %#v", got, devMap)
	}
	if got := index.ByAddress(0x10); len(got) != 0 {
		t.Fatalf("ByAddress(0x10) = %#v, want empty", got)
	}
	if got := index.ByAddress(0x11); len(got) != 1 || got[0] != devMap {
		t.Fatalf("ByAddress(0x11) = %#v", got)
	}
}

func TestMatchFingerprint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a-match.yaml"), `
device: MATCH
addresses: [0x68]
fingerprint:
  - {reg: 0x75, value: 0x68}
registers:
  - {name: ID, addr: 0x75, rw: r}
`)
	writeFile(t, filepath.Join(dir, "b-other.yaml"), `
device: OTHER
addresses: [0x68]
fingerprint:
  - {reg: 0x75, value: 0x69}
registers:
  - {name: ID, addr: 0x75, rw: r}
`)

	index, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}
	readCalls := 0
	got, err := index.MatchFingerprint(0x68, func(reg uint16) (byte, error) {
		readCalls++
		if reg != 0x75 {
			t.Fatalf("read register = 0x%X, want 0x75", reg)
		}
		return 0x68, nil
	})
	if err != nil || got == nil || got.Device != "MATCH" || readCalls != 1 {
		t.Fatalf("MatchFingerprint() = %#v, %v, calls=%d", got, err, readCalls)
	}

	got, err = index.MatchFingerprint(0x50, func(uint16) (byte, error) {
		t.Fatal("reader called for unknown address")
		return 0, nil
	})
	if err != nil || got != nil {
		t.Fatalf("unknown MatchFingerprint() = %#v, %v", got, err)
	}

	readErr := errors.New("read failed")
	if _, err := index.MatchFingerprint(0x68, func(uint16) (byte, error) {
		return 0, readErr
	}); !errors.Is(err, readErr) {
		t.Fatalf("read error = %v, want %v", err, readErr)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
