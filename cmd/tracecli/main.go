package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
	"github.com/baoerzuikeai/tracecli/internal/app"
	"github.com/baoerzuikeai/tracecli/internal/device"
	"github.com/baoerzuikeai/tracecli/internal/protocol"
	"github.com/baoerzuikeai/tracecli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Serial    serialConfig              `yaml:"serial"`
	Protocols map[string]protocolConfig `yaml:"protocols"`
	Behavior  behaviorConfig            `yaml:"behavior"`
}

type serialConfig struct {
	DefaultPort string `yaml:"default_port"`
	BaudRate    int    `yaml:"baud_rate"`
	LatencyMS   int    `yaml:"latency_ms"`
}

type protocolConfig struct {
	Clock uint32 `yaml:"clock"`
	Mode  uint8  `yaml:"mode"`
}

type behaviorConfig struct {
	AutoRescanOnConnect bool `yaml:"auto_rescan_on_connect"`
	AutoReconnect       bool `yaml:"auto_reconnect"`
	TerminalCheck       bool `yaml:"terminal_check"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tracecli:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("tracecli", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "config.yaml", "configuration file")
	devMapsPath := flags.String("devmaps", "devmaps", "device map directory")
	if err := flags.Parse(args); err != nil {
		return err
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	devMaps, err := device.LoadDir(*devMapsPath)
	if err != nil {
		return err
	}
	for _, warning := range devMaps.Warnings() {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}

	mock := adapter.NewMock()
	application, err := app.New(mock, app.Config{
		AdapterConfig: adapter.Config{
			PortName: config.Serial.DefaultPort,
			BaudRate: config.Serial.BaudRate,
			Latency:  time.Duration(config.Serial.LatencyMS) * time.Millisecond,
		},
		AutoReconnect: config.Behavior.AutoReconnect,
		EventBuffer:   32,
	})
	if err != nil {
		return err
	}
	defer application.Close()

	model := tui.NewWithApp(application, protocol.Supported(), devMaps)
	if err := application.Start(context.Background()); err != nil {
		return err
	}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func loadConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, err
	}

	var config fileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fileConfig{}, err
	}
	return config, nil
}
