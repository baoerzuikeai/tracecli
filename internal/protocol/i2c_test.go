package protocol

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/baoerzuikeai/tracecli/internal/device"
)

func TestI2CProbeAndFingerprint(t *testing.T) {
	mock := openMock(t)
	devMap := &device.DevMap{
		Device:    "MPU6050",
		Vendor:    "TDK InvenSense",
		Addresses: []uint16{0x68},
		Fingerprint: []device.Fingerprint{
			{Reg: 0x75, Value: 0x68},
		},
	}
	debugger, err := NewI2CWithOptions(mock, I2COptions{
		Addresses:   []uint16{0x20, 0x50, 0x68},
		DevMaps:     []*device.DevMap{devMap},
		ReadTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewI2CWithOptions() error = %v", err)
	}

	mock.Feed([]byte{0x00, 0x01, 0x01, 0x68})
	targets, err := debugger.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	want := []Target{
		{Protocol: "i2c", Address: 0x50, Name: "Unknown@0x50"},
		{Protocol: "i2c", Address: 0x68, Name: "MPU6050", DevMap: devMap},
	}
	if len(targets) != len(want) {
		t.Fatalf("Probe() returned %d targets, want %d: %#v", len(targets), len(want), targets)
	}
	for index := range want {
		if targets[index].Protocol != want[index].Protocol ||
			targets[index].Address != want[index].Address ||
			targets[index].Name != want[index].Name ||
			targets[index].DevMap != want[index].DevMap {
			t.Fatalf("Probe()[%d] = %#v, want %#v", index, targets[index], want[index])
		}
	}

	wantWritten := append([]byte{}, encodeProbe(0x20)...)
	wantWritten = append(wantWritten, encodeProbe(0x50)...)
	wantWritten = append(wantWritten, encodeProbe(0x68)...)
	wantWritten = append(wantWritten, encodeRead(0x68, 0x75, 1)...)
	if got := mock.Written(); !slices.Equal(got, wantWritten) {
		t.Fatalf("Written() = %x, want %x", got, wantWritten)
	}
}

func TestI2CProbeFingerprintMiss(t *testing.T) {
	mock := openMock(t)
	devMap := &device.DevMap{
		Device:      "MPU6050",
		Addresses:   []uint16{0x68},
		Fingerprint: []device.Fingerprint{{Reg: 0x75, Value: 0x68}},
	}
	debugger, err := NewI2CWithOptions(mock, I2COptions{
		Addresses: []uint16{0x68},
		DevMaps:   []*device.DevMap{devMap},
	})
	if err != nil {
		t.Fatalf("NewI2CWithOptions() error = %v", err)
	}

	mock.Feed([]byte{0x01, 0x00})
	targets, err := debugger.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if want := []Target{{Protocol: "i2c", Address: 0x68, Name: "Unknown@0x68"}}; len(targets) != 1 ||
		targets[0].Name != want[0].Name || targets[0].DevMap != nil {
		t.Fatalf("Probe() = %#v, want %#v", targets, want)
	}
}

func TestI2CReadWriteReset(t *testing.T) {
	mock := openMock(t)
	debugger, err := NewI2CWithOptions(mock, I2COptions{ReadTimeout: time.Second})
	if err != nil {
		t.Fatalf("NewI2CWithOptions() error = %v", err)
	}
	target := Target{Protocol: "i2c", Address: 0x50}

	mock.Feed([]byte{0x11, 0x22, 0x33})
	got, err := debugger.Read(target, 0x10, 3)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if want := []byte{0x11, 0x22, 0x33}; !slices.Equal(got, want) {
		t.Fatalf("Read() = %x, want %x", got, want)
	}

	if err := debugger.Write(target, 0x20, []byte{0xaa, 0xbb}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	wantWritten := append([]byte{}, encodeRead(0x50, 0x10, 3)...)
	wantWritten = append(wantWritten, encodeWrite(0x50, 0x20, []byte{0xaa, 0xbb})...)
	if got := mock.Written(); !slices.Equal(got, wantWritten) {
		t.Fatalf("Written() = %x, want %x", got, wantWritten)
	}

	if err := debugger.Reset(target); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if got, want := mock.ResetCount(), 1; got != want {
		t.Fatalf("ResetCount() = %d, want %d", got, want)
	}
}

func TestI2CValidationAndCapabilities(t *testing.T) {
	mock := openMock(t)
	if _, err := NewI2CWithOptions(mock, I2COptions{Addresses: []uint16{0x80}}); !errors.Is(err, ErrInvalidI2CAddress) {
		t.Fatalf("invalid option address error = %v, want %v", err, ErrInvalidI2CAddress)
	}

	debugger, err := NewI2C(mock)
	if err != nil {
		t.Fatalf("NewI2C() error = %v", err)
	}
	wantCapabilities := []Capability{CapReadWrite, CapScan, CapFingerprint, CapReset}
	if got := debugger.Capabilities(); !slices.Equal(got, wantCapabilities) {
		t.Fatalf("Capabilities() = %v, want %v", got, wantCapabilities)
	}
	if _, err := debugger.Read(Target{Protocol: "spi", Address: 0x50}, 0, 1); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("invalid target Read() error = %v, want %v", err, ErrInvalidTarget)
	}
	if _, err := debugger.Read(Target{Address: 0x50}, 0, -1); !errors.Is(err, ErrInvalidTransferSize) {
		t.Fatalf("invalid size Read() error = %v, want %v", err, ErrInvalidTransferSize)
	}
}

func TestI2CProbeContext(t *testing.T) {
	mock := openMock(t)
	debugger, err := NewI2CWithOptions(mock, I2COptions{Addresses: []uint16{0x50}})
	if err != nil {
		t.Fatalf("NewI2CWithOptions() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := debugger.Probe(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Probe() error = %v, want %v", err, context.Canceled)
	}
	if got := mock.Written(); len(got) != 0 {
		t.Fatalf("Probe() wrote %x for cancelled context", got)
	}
}
