package device

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Fingerprint struct {
	Reg   uint16 `yaml:"reg"`
	Value byte   `yaml:"value"`
}

type DevMap struct {
	Device      string        `yaml:"device"`
	Vendor      string        `yaml:"vendor"`
	Addresses   []uint16      `yaml:"addresses"`
	Fingerprint []Fingerprint `yaml:"fingerprint"`
	Registers   []Register    `yaml:"registers"`
	Operations  []Operation   `yaml:"operations"`
}

type Register struct {
	Name    string     `yaml:"name"`
	Addr    uint16     `yaml:"addr"`
	RW      string     `yaml:"rw"`
	Reset   byte       `yaml:"reset"`
	Default byte       `yaml:"default"`
	Bits    []BitField `yaml:"bits"`
}

type BitField struct {
	Name string   `yaml:"name"`
	Bit  BitRange `yaml:"bit"`
	RW   string   `yaml:"rw"`
}

type BitRange []uint8

type Operation struct {
	Name  string          `yaml:"name"`
	Steps []OperationStep `yaml:"steps"`
}

type OperationStep struct {
	Type string `yaml:"type"`
	Reg  uint16 `yaml:"reg"`
	Data []byte `yaml:"data"`
	MS   uint32 `yaml:"ms"`
}

func (r *BitRange) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var bit uint8
		if err := node.Decode(&bit); err != nil {
			return err
		}
		*r = BitRange{bit}
		return nil
	}
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("bit must be a scalar or sequence")
	}

	var bits []uint8
	if err := node.Decode(&bits); err != nil {
		return err
	}
	if len(bits) == 0 {
		return fmt.Errorf("bit range must not be empty")
	}
	*r = BitRange(bits)
	return nil
}

func (d *DevMap) Validate() error {
	if d == nil {
		return fmt.Errorf("devmap must not be nil")
	}
	if strings.TrimSpace(d.Device) == "" {
		return fmt.Errorf("device is required")
	}
	if len(d.Addresses) == 0 {
		return fmt.Errorf("addresses are required")
	}
	if len(d.Registers) == 0 {
		return fmt.Errorf("registers are required")
	}
	for _, register := range d.Registers {
		if strings.TrimSpace(register.Name) == "" {
			return fmt.Errorf("register name is required")
		}
		if register.RW != "" && !validAccess(register.RW) {
			return fmt.Errorf("invalid register access %q", register.RW)
		}
		for _, bit := range register.Bits {
			if len(bit.Bit) == 0 {
				return fmt.Errorf("bit range is required for %q", bit.Name)
			}
			for _, position := range bit.Bit {
				if position > 7 {
					return fmt.Errorf("bit position %d is out of range", position)
				}
			}
			if bit.RW != "" && !validAccess(bit.RW) {
				return fmt.Errorf("invalid bit access %q", bit.RW)
			}
		}
	}
	return nil
}

func validAccess(value string) bool {
	return value == "r" || value == "w" || value == "rw"
}
