package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
serial:
  default_port: COM7
  baud_rate: 9600
  latency_ms: 2
protocols:
  i2c:
    clock: 400000
behavior:
  auto_reconnect: true
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.Serial.DefaultPort != "COM7" || config.Serial.BaudRate != 9600 || config.Serial.LatencyMS != 2 {
		t.Fatalf("serial config = %#v", config.Serial)
	}
	if !config.Behavior.AutoReconnect || config.Protocols["i2c"].Clock != 400000 {
		t.Fatalf("loaded config = %#v", config)
	}
}
