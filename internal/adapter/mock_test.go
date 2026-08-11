package adapter

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestMockOpenClose(t *testing.T) {
	mock := NewMock()
	cfg := Config{PortName: "COM3", BaudRate: 115200, Latency: 2 * time.Millisecond}

	if err := mock.Open(cfg); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := mock.Open(cfg); !errors.Is(err, ErrAlreadyOpen) {
		t.Fatalf("second Open() error = %v, want %v", err, ErrAlreadyOpen)
	}
	if got := mock.Latency(); got != cfg.Latency {
		t.Fatalf("Latency() = %s, want %s", got, cfg.Latency)
	}
	if err := mock.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := mock.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("second Close() error = %v, want %v", err, ErrClosed)
	}
}

func TestMockRead(t *testing.T) {
	tests := []struct {
		name    string
		feed    [][]byte
		readN   int
		timeout time.Duration
		want    []byte
		wantErr error
	}{
		{
			name:    "reads full data from multiple chunks",
			feed:    [][]byte{{0x01, 0x02}, {0x03, 0x04}},
			readN:   4,
			timeout: time.Second,
			want:    []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:    "returns partial data on timeout",
			feed:    [][]byte{{0x10, 0x20}},
			readN:   4,
			timeout: 10 * time.Millisecond,
			want:    []byte{0x10, 0x20},
			wantErr: ErrReadTimeout,
		},
		{
			name:    "returns timeout without data",
			readN:   1,
			timeout: 10 * time.Millisecond,
			wantErr: ErrReadTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMock()
			if err := mock.Open(Config{}); err != nil {
				t.Fatalf("Open() error = %v", err)
			}

			for i, data := range tt.feed {
				if i == 0 {
					mock.Feed(data)
					continue
				}
				go func(data []byte) {
					time.Sleep(time.Millisecond)
					mock.Feed(data)
				}(data)
			}

			got, err := mock.Read(tt.readN, tt.timeout)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Read() error = %v, want %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Read() = %x, want %x", got, tt.want)
			}
		})
	}
}

func TestMockReadClosed(t *testing.T) {
	mock := NewMock()
	if _, err := mock.Read(1, time.Second); !errors.Is(err, ErrClosed) {
		t.Fatalf("Read() error = %v, want %v", err, ErrClosed)
	}
	if _, err := mock.Read(-1, time.Second); !errors.Is(err, ErrInvalidReadN) {
		t.Fatalf("Read() with invalid size error = %v, want %v", err, ErrInvalidReadN)
	}
}

func TestMockWriteAndIdentity(t *testing.T) {
	mock := NewMock()
	mock.SetIdentity("CH341", "serial-1")
	if vendor, serial := mock.ID(); vendor != "CH341" || serial != "serial-1" {
		t.Fatalf("ID() = %q, %q", vendor, serial)
	}
	if err := mock.Write([]byte{0x01}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Write() before Open error = %v, want %v", err, ErrClosed)
	}
	if err := mock.Open(Config{}); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	data := []byte{0x01, 0x02}
	if err := mock.Write(data); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	data[0] = 0xff
	if got, want := mock.Written(), []byte{0x01, 0x02}; !slices.Equal(got, want) {
		t.Fatalf("Written() = %x, want %x", got, want)
	}
}

func TestMockReset(t *testing.T) {
	mock := NewMock()
	if err := mock.Reset(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Reset() before Open error = %v, want %v", err, ErrClosed)
	}
	if err := mock.Open(Config{}); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := mock.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if err := mock.Reset(); err != nil {
		t.Fatalf("second Reset() error = %v", err)
	}
	if got, want := mock.ResetCount(), 2; got != want {
		t.Fatalf("ResetCount() = %d, want %d", got, want)
	}
}
