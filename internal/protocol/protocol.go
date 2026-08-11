package protocol

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
	"github.com/baoerzuikeai/tracecli/internal/device"
)

type Target struct {
	Protocol string
	Address  uint16
	Name     string
	DevMap   *device.DevMap
}

type Capability uint32

const (
	CapReadWrite Capability = 1 << iota
	CapScan
	CapFingerprint
	CapReset
	CapWaveform
	CapBatch
)

type Debugger interface {
	Name() string
	Probe(ctx context.Context) ([]Target, error)
	Read(t Target, addr uint16, n int) ([]byte, error)
	Write(t Target, addr uint16, data []byte) error
	Reset(t Target) error
	Capabilities() []Capability
}

type Factory func(a adapter.Adapter) (Debugger, error)

var (
	ErrUnknownProtocol = errors.New("protocol: unknown protocol")
	ErrNilFactory      = errors.New("protocol: nil factory")
)

var (
	registry   = map[string]Factory{}
	registryMu sync.RWMutex
)

func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[name] = f
}

func Supported() []string {
	registryMu.RLock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	registryMu.RUnlock()

	sort.Strings(names)
	return names
}

func New(name string, a adapter.Adapter) (Debugger, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProtocol, name)
	}
	if factory == nil {
		return nil, fmt.Errorf("%w: %s", ErrNilFactory, name)
	}

	return factory(a)
}
