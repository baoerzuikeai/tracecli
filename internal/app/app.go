package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
)

type State uint8

const (
	Disconnected State = iota
	Connecting
	Ready
	Error
)

const (
	StateDisconnected = Disconnected
	StateConnecting   = Connecting
	StateReady        = Ready
	StateError        = Error
)

func (s State) String() string {
	switch s {
	case Disconnected:
		return "Disconnected"
	case Connecting:
		return "Connecting"
	case Ready:
		return "Ready"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}

type Config struct {
	AdapterConfig  adapter.Config
	AutoReconnect  bool
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	EventBuffer    int
}

type App struct {
	adapter adapter.Adapter
	config  Config
	events  *EventBus

	mu         sync.RWMutex
	state      State
	started    bool
	stopping   bool
	cancel     context.CancelFunc
	done       chan struct{}
	disconnect chan error
}

var (
	ErrNilAdapter     = errors.New("app: nil adapter")
	ErrNilContext     = errors.New("app: nil context")
	ErrAlreadyStarted = errors.New("app: already started")
	ErrDisconnected   = errors.New("app: adapter disconnected")
)

const (
	defaultInitialBackoff = 100 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
)

var _ interface {
	Start(context.Context) error
	Stop()
} = (*App)(nil)

func New(a adapter.Adapter, config Config) (*App, error) {
	if a == nil {
		return nil, ErrNilAdapter
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = defaultInitialBackoff
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = defaultMaxBackoff
	}
	if config.MaxBackoff < config.InitialBackoff {
		config.MaxBackoff = config.InitialBackoff
	}

	return &App{
		adapter: a,
		config:  config,
		events:  NewEventBusWithBuffer(config.EventBuffer),
		state:   Disconnected,
	}, nil
}

func (a *App) Start(parent context.Context) error {
	if parent == nil {
		return ErrNilContext
	}
	if err := parent.Err(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		cancel()
		return ErrAlreadyStarted
	}
	a.started = true
	a.stopping = false
	a.cancel = cancel
	a.done = make(chan struct{})
	disconnect := make(chan error, 1)
	a.disconnect = disconnect
	a.mu.Unlock()

	go a.run(ctx, disconnect, a.done)
	return nil
}

func (a *App) Stop() {
	a.mu.Lock()
	if !a.started {
		shouldDisconnect := a.state != Disconnected
		a.mu.Unlock()
		if shouldDisconnect {
			a.transition(Disconnected, nil, 0)
		}
		return
	}
	a.stopping = true
	cancel := a.cancel
	done := a.done
	a.mu.Unlock()

	cancel()
	<-done
}

func (a *App) Close() {
	a.Stop()
	a.events.Close()
}

func (a *App) State() State {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.state
}

func (a *App) Events() *EventBus {
	return a.events
}

func (a *App) Subscribe() <-chan Event {
	return a.events.Subscribe()
}

func (a *App) ReportError(err error) {
	if err == nil {
		err = ErrDisconnected
	}

	a.mu.Lock()
	if !a.started || a.stopping || a.state != Ready {
		a.mu.Unlock()
		return
	}
	previous := a.state
	a.state = Error
	disconnect := a.currentDisconnectLocked()
	a.mu.Unlock()

	a.publish(Event{
		Type:     EventStateChanged,
		Previous: previous,
		State:    Error,
		Err:      err,
		Attempt:  1,
		At:       time.Now(),
	})
	select {
	case disconnect <- err:
	default:
	}
}

func (a *App) NotifyError(err error) {
	a.ReportError(err)
}

func (a *App) Disconnect(err error) {
	a.ReportError(err)
}

func (a *App) run(ctx context.Context, disconnect <-chan error, done chan struct{}) {
	defer a.finish(done)

	attempt := 0
	for {
		if ctx.Err() != nil {
			a.transition(Disconnected, nil, 0)
			return
		}

		attempt++
		a.transition(Connecting, nil, attempt)
		err := a.adapter.Open(a.config.AdapterConfig)
		if ctx.Err() != nil {
			_ = a.adapter.Close()
			a.transition(Disconnected, nil, 0)
			return
		}
		if err == nil {
			attempt = 0
			a.transition(Ready, nil, 0)
			select {
			case <-ctx.Done():
				_ = a.adapter.Close()
				a.transition(Disconnected, nil, 0)
				return
			case disconnectErr := <-disconnect:
				_ = a.adapter.Close()
				if disconnectErr == nil {
					disconnectErr = ErrDisconnected
				}
				if a.State() != Error {
					a.transition(Error, disconnectErr, 1)
				}
				attempt = 1
			}
		} else {
			_ = a.adapter.Close()
			a.transition(Error, err, attempt)
		}

		if !a.config.AutoReconnect {
			return
		}
		if !a.wait(ctx, a.backoff(attempt)) {
			a.transition(Disconnected, nil, 0)
			return
		}
	}
}

func (a *App) finish(done chan struct{}) {
	a.mu.Lock()
	if a.done == done {
		close(done)
		a.done = nil
		a.cancel = nil
		a.disconnect = nil
		a.started = false
		a.stopping = false
	}
	a.mu.Unlock()
}

func (a *App) wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (a *App) backoff(attempt int) time.Duration {
	delay := a.config.InitialBackoff
	for index := 1; index < attempt; index++ {
		if delay >= a.config.MaxBackoff/2 {
			return a.config.MaxBackoff
		}
		delay *= 2
	}
	if delay > a.config.MaxBackoff {
		return a.config.MaxBackoff
	}
	return delay
}

func (a *App) transition(state State, err error, attempt int) {
	a.mu.Lock()
	previous := a.state
	if previous == state {
		a.mu.Unlock()
		return
	}
	a.state = state
	a.mu.Unlock()

	a.publish(Event{
		Type:     EventStateChanged,
		Previous: previous,
		State:    state,
		Err:      err,
		Attempt:  attempt,
		At:       time.Now(),
	})
}

func (a *App) currentDisconnectLocked() chan error {
	return a.disconnect
}

func (a *App) publish(event Event) {
	a.events.Publish(event)
}
