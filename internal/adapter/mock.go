package adapter

import (
	"errors"
	"io"
	"sync"
	"time"
)

var (
	ErrAlreadyOpen  = errors.New("adapter: already open")
	ErrClosed       = errors.New("adapter: closed")
	ErrInvalidReadN = errors.New("adapter: read size must be non-negative")
	ErrReadTimeout  = errors.New("adapter: read timeout")
)

type Mock struct {
	mu         sync.Mutex
	opened     bool
	config     Config
	input      []byte
	output     []byte
	vendor     string
	serial     string
	resetCount int
	notify     chan struct{}
}

var _ Adapter = (*Mock)(nil)

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) Open(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.opened {
		return ErrAlreadyOpen
	}

	m.opened = true
	m.config = cfg
	m.signalLocked()
	return nil
}

func (m *Mock) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.opened {
		return ErrClosed
	}

	m.opened = false
	m.signalLocked()
	return nil
}

func (m *Mock) Write(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.opened {
		return ErrClosed
	}

	m.output = append(m.output, b...)
	return nil
}

func (m *Mock) Read(n int, timeout time.Duration) ([]byte, error) {
	if n < 0 {
		return nil, ErrInvalidReadN
	}
	if n == 0 {
		return []byte{}, nil
	}

	b := make([]byte, n)
	reader := mockReader{mock: m, deadline: time.Now().Add(timeout)}
	readN, err := io.ReadFull(&reader, b)
	return b[:readN], err
}

func (m *Mock) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.opened {
		return ErrClosed
	}

	m.resetCount++
	return nil
}

func (m *Mock) ID() (vendor, serial string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.vendor, m.serial
}

func (m *Mock) Latency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.config.Latency
}

func (m *Mock) Feed(b []byte) {
	if len(b) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.input = append(m.input, b...)
	m.signalLocked()
}

func (m *Mock) Written() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := make([]byte, len(m.output))
	copy(b, m.output)
	return b
}

func (m *Mock) SetIdentity(vendor, serial string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.vendor = vendor
	m.serial = serial
}

func (m *Mock) ResetCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.resetCount
}

func (m *Mock) signalLocked() {
	if m.notify == nil {
		m.notify = make(chan struct{})
	}

	close(m.notify)
	m.notify = make(chan struct{})
}

type mockReader struct {
	mock     *Mock
	deadline time.Time
}

func (r *mockReader) Read(p []byte) (int, error) {
	for {
		r.mock.mu.Lock()
		if !r.mock.opened {
			r.mock.mu.Unlock()
			return 0, ErrClosed
		}
		if len(r.mock.input) > 0 {
			n := copy(p, r.mock.input)
			r.mock.input = r.mock.input[n:]
			r.mock.mu.Unlock()
			return n, nil
		}
		notify := r.mock.notify
		r.mock.mu.Unlock()

		remaining := time.Until(r.deadline)
		if remaining <= 0 {
			return 0, ErrReadTimeout
		}

		timer := time.NewTimer(remaining)
		select {
		case <-notify:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			return 0, ErrReadTimeout
		}
	}
}
