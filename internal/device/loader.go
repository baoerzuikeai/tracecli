package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FingerprintReader func(reg uint16) (byte, error)

type FingerprintKey struct {
	Address uint16
	Reg     uint16
	Value   byte
}

type Warning struct {
	Path string
	Err  error
}

func (w Warning) Error() string {
	return fmt.Sprintf("%s: %v", w.Path, w.Err)
}

func (w Warning) Unwrap() error {
	return w.Err
}

type Index struct {
	byKey         map[string]*DevMap
	byName        map[string]*DevMap
	byAddress     map[uint16][]*DevMap
	byFingerprint map[FingerprintKey][]*DevMap
	order         []string
	warnings      []Warning
}

func LoadDir(dir string) (*Index, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	index := newIndex()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension != ".yaml" && extension != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		devMap, err := LoadFile(path)
		if err != nil {
			index.warnings = append(index.warnings, Warning{Path: path, Err: err})
			continue
		}
		key := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		index.add(key, devMap)
	}

	return index, nil
}

func Load(dir string) (*Index, error) {
	return LoadDir(dir)
}

func LoadFile(path string) (*DevMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var devMap DevMap
	if err := yaml.Unmarshal(data, &devMap); err != nil {
		return nil, err
	}
	if err := devMap.Validate(); err != nil {
		return nil, err
	}
	return &devMap, nil
}

func (i *Index) Get(key string) (*DevMap, bool) {
	devMap, ok := i.byKey[key]
	return devMap, ok
}

func (i *Index) Lookup(name string) *DevMap {
	if devMap, ok := i.byKey[name]; ok {
		return devMap
	}
	return i.byName[name]
}

func (i *Index) All() []*DevMap {
	devMaps := make([]*DevMap, 0, len(i.order))
	for _, key := range i.order {
		devMaps = append(devMaps, i.byKey[key])
	}
	return devMaps
}

func (i *Index) ByAddress(address uint16) []*DevMap {
	return append([]*DevMap(nil), i.byAddress[address]...)
}

func (i *Index) ByFingerprint(address, reg uint16, value byte) []*DevMap {
	key := FingerprintKey{Address: address, Reg: reg, Value: value}
	return append([]*DevMap(nil), i.byFingerprint[key]...)
}

func (i *Index) Warnings() []Warning {
	return append([]Warning(nil), i.warnings...)
}

func (i *Index) MatchFingerprint(address uint16, read FingerprintReader) (*DevMap, error) {
	for _, devMap := range i.byAddress[address] {
		matched := true
		for _, fingerprint := range devMap.Fingerprint {
			value, err := read(fingerprint.Reg)
			if err != nil {
				return nil, err
			}
			if value != fingerprint.Value {
				matched = false
				break
			}
		}
		if matched {
			return devMap, nil
		}
	}
	return nil, nil
}

func (i *Index) Match(address uint16, read FingerprintReader) (*DevMap, error) {
	return i.MatchFingerprint(address, read)
}

func newIndex() *Index {
	return &Index{
		byKey:         make(map[string]*DevMap),
		byName:        make(map[string]*DevMap),
		byAddress:     make(map[uint16][]*DevMap),
		byFingerprint: make(map[FingerprintKey][]*DevMap),
	}
}

func (i *Index) add(key string, devMap *DevMap) {
	if oldKey, ok := i.byKey[key]; ok {
		delete(i.byName, oldKey.Device)
		i.removeOrder(key)
	}
	for oldKey, oldMap := range i.byKey {
		if oldMap.Device == devMap.Device && oldKey != key {
			delete(i.byKey, oldKey)
			i.removeOrder(oldKey)
		}
	}

	i.byKey[key] = devMap
	i.order = append(i.order, key)
	i.rebuild()
}

func (i *Index) removeOrder(key string) {
	for index, current := range i.order {
		if current == key {
			i.order = append(i.order[:index], i.order[index+1:]...)
			return
		}
	}
}

func (i *Index) rebuild() {
	i.byName = make(map[string]*DevMap)
	i.byAddress = make(map[uint16][]*DevMap)
	i.byFingerprint = make(map[FingerprintKey][]*DevMap)
	for _, key := range i.order {
		devMap := i.byKey[key]
		i.byName[devMap.Device] = devMap
		for _, address := range devMap.Addresses {
			i.byAddress[address] = append(i.byAddress[address], devMap)
			for _, fingerprint := range devMap.Fingerprint {
				fingerprintKey := FingerprintKey{
					Address: address,
					Reg:     fingerprint.Reg,
					Value:   fingerprint.Value,
				}
				i.byFingerprint[fingerprintKey] = append(i.byFingerprint[fingerprintKey], devMap)
			}
		}
	}
}
