package protocol

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
	"github.com/baoerzuikeai/tracecli/internal/device"
)

const (
	i2cCommandProbe byte = iota + 1
	i2cCommandRead
	i2cCommandWrite

	i2cAck byte = 1

	defaultI2CReadTimeout = time.Second
	firstI2CAddress       = 0x03
	lastI2CAddress        = 0x77
)

var (
	ErrNilAdapter          = errors.New("protocol: nil adapter")
	ErrNilContext          = errors.New("protocol: nil context")
	ErrInvalidI2CAddress   = errors.New("protocol: invalid i2c address")
	ErrInvalidTransferSize = errors.New("protocol: invalid i2c transfer size")
	ErrInvalidTarget       = errors.New("protocol: invalid target")
	ErrMalformedResponse   = errors.New("protocol: malformed adapter response")
)

type I2COptions struct {
	Clock       uint32
	Addresses   []uint16
	DevMaps     []*device.DevMap
	ReadTimeout time.Duration
}

type I2C struct {
	adapter adapter.Adapter
	options I2COptions
}

var _ Debugger = (*I2C)(nil)

func init() {
	Register("i2c", func(a adapter.Adapter) (Debugger, error) {
		return NewI2C(a)
	})
}

func NewI2C(a adapter.Adapter) (*I2C, error) {
	return NewI2CWithOptions(a, I2COptions{})
}

func NewI2CWithOptions(a adapter.Adapter, options I2COptions) (*I2C, error) {
	if a == nil {
		return nil, ErrNilAdapter
	}

	addresses := options.Addresses
	if len(addresses) == 0 {
		addresses = make([]uint16, lastI2CAddress-firstI2CAddress+1)
		for index := range addresses {
			addresses[index] = firstI2CAddress + uint16(index)
		}
	} else {
		addresses = append([]uint16(nil), addresses...)
	}
	for _, address := range addresses {
		if address > 0x7f {
			return nil, fmt.Errorf("%w: 0x%X", ErrInvalidI2CAddress, address)
		}
	}

	if options.ReadTimeout <= 0 {
		options.ReadTimeout = defaultI2CReadTimeout
	}
	options.Addresses = addresses
	options.DevMaps = append([]*device.DevMap(nil), options.DevMaps...)
	return &I2C{adapter: a, options: options}, nil
}

func (i *I2C) Name() string {
	return "i2c"
}

func (i *I2C) Probe(ctx context.Context) ([]Target, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	targets := make([]Target, 0)
	for _, address := range i.options.Addresses {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := i.adapter.Write(encodeProbe(address)); err != nil {
			return nil, err
		}
		response, err := i.adapter.Read(1, i.options.ReadTimeout)
		if err != nil {
			return nil, err
		}
		if len(response) != 1 {
			return nil, ErrMalformedResponse
		}
		if response[0] != i2cAck {
			continue
		}

		target, err := i.identify(ctx, address)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}

func (i *I2C) Read(t Target, addr uint16, n int) ([]byte, error) {
	if err := validateTarget(t); err != nil {
		return nil, err
	}
	if n < 0 || n > math.MaxUint16 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidTransferSize, n)
	}

	if err := i.adapter.Write(encodeRead(t.Address, addr, n)); err != nil {
		return nil, err
	}
	return i.adapter.Read(n, i.options.ReadTimeout)
}

func (i *I2C) Write(t Target, addr uint16, data []byte) error {
	if err := validateTarget(t); err != nil {
		return err
	}
	if len(data) > math.MaxUint16 {
		return fmt.Errorf("%w: %d", ErrInvalidTransferSize, len(data))
	}

	return i.adapter.Write(encodeWrite(t.Address, addr, data))
}

func (i *I2C) Reset(t Target) error {
	if err := validateTarget(t); err != nil {
		return err
	}

	return i.adapter.Reset()
}

func (i *I2C) Capabilities() []Capability {
	return []Capability{CapReadWrite, CapScan, CapFingerprint, CapReset}
}

func (i *I2C) identify(ctx context.Context, address uint16) (Target, error) {
	target := Target{
		Protocol: "i2c",
		Address:  address,
		Name:     fmt.Sprintf("Unknown@0x%02X", address),
	}

	for _, devMap := range i.options.DevMaps {
		if devMap == nil || !containsAddress(devMap.Addresses, address) {
			continue
		}

		matched, err := i.matchesFingerprint(ctx, address, devMap)
		if err != nil {
			return Target{}, err
		}
		if matched {
			target.Name = devMap.Device
			target.DevMap = devMap
			return target, nil
		}
	}

	return target, nil
}

func (i *I2C) matchesFingerprint(ctx context.Context, address uint16, devMap *device.DevMap) (bool, error) {
	for _, fingerprint := range devMap.Fingerprint {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		data, err := i.Read(Target{Protocol: "i2c", Address: address}, fingerprint.Reg, 1)
		if err != nil {
			return false, err
		}
		if len(data) != 1 {
			return false, ErrMalformedResponse
		}
		if data[0] != fingerprint.Value {
			return false, nil
		}
	}

	return true, nil
}

func containsAddress(addresses []uint16, want uint16) bool {
	for _, address := range addresses {
		if address == want {
			return true
		}
	}
	return false
}

func validateTarget(target Target) error {
	if target.Protocol != "" && target.Protocol != "i2c" {
		return fmt.Errorf("%w: %s", ErrInvalidTarget, target.Protocol)
	}
	if target.Address > 0x7f {
		return fmt.Errorf("%w: 0x%X", ErrInvalidI2CAddress, target.Address)
	}
	return nil
}

func encodeProbe(address uint16) []byte {
	return []byte{i2cCommandProbe, byte(address)}
}

func encodeRead(address, register uint16, n int) []byte {
	frame := make([]byte, 6)
	frame[0] = i2cCommandRead
	frame[1] = byte(address)
	binary.BigEndian.PutUint16(frame[2:4], register)
	binary.BigEndian.PutUint16(frame[4:6], uint16(n))
	return frame
}

func encodeWrite(address, register uint16, data []byte) []byte {
	frame := make([]byte, 6+len(data))
	frame[0] = i2cCommandWrite
	frame[1] = byte(address)
	binary.BigEndian.PutUint16(frame[2:4], register)
	binary.BigEndian.PutUint16(frame[4:6], uint16(len(data)))
	copy(frame[6:], data)
	return frame
}
