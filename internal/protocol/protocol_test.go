package protocol

import (
	"errors"
	"testing"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
)

func TestRegistry(t *testing.T) {
	names := Supported()
	if len(names) == 0 {
		t.Fatal("Supported() returned no protocols")
	}
	for index := 1; index < len(names); index++ {
		if names[index-1] > names[index] {
			t.Fatalf("Supported() is not sorted: %v", names)
		}
	}

	mock := openMock(t)
	debugger, err := New("i2c", mock)
	if err != nil {
		t.Fatalf("New(i2c) error = %v", err)
	}
	if debugger.Name() != "i2c" {
		t.Fatalf("Name() = %q, want i2c", debugger.Name())
	}
	if _, err := New("missing", mock); !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("New(missing) error = %v, want %v", err, ErrUnknownProtocol)
	}
}

func TestRegistryRejectsNilI2CAdapter(t *testing.T) {
	if _, err := New("i2c", nil); !errors.Is(err, ErrNilAdapter) {
		t.Fatalf("New(i2c, nil) error = %v, want %v", err, ErrNilAdapter)
	}
}

func openMock(t *testing.T) *adapter.Mock {
	t.Helper()

	mock := adapter.NewMock()
	if err := mock.Open(adapter.Config{}); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return mock
}
