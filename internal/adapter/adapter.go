package adapter

import "time"

type Config struct {
	PortName string
	BaudRate int
	Latency  time.Duration
}

type Adapter interface {
	Open(cfg Config) error
	Close() error
	Write(b []byte) error
	Read(n int, timeout time.Duration) ([]byte, error)
	Reset() error
	ID() (vendor, serial string)
	Latency() time.Duration
}
