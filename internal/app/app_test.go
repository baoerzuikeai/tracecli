package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/baoerzuikeai/tracecli/internal/adapter"
)

func TestStateMigrationAndDrop(t *testing.T) {
	mock := newScriptedAdapter(nil)
	application, err := New(mock, Config{
		AutoReconnect:  true,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	events := application.Subscribe()
	if err := application.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	connecting := waitForState(t, events, Connecting)
	ready := waitForState(t, events, Ready)
	if connecting.Previous != Disconnected || ready.Previous != Connecting {
		t.Fatalf("initial transitions = %s -> %s -> %s", connecting.Previous, connecting.State, ready.State)
	}

	dropErr := errors.New("device removed")
	mock.ReportDrop(application, dropErr)
	disconnected := waitForState(t, events, Error)
	if !errors.Is(disconnected.Err, dropErr) {
		t.Fatalf("drop error = %v, want %v", disconnected.Err, dropErr)
	}

	application.Stop()
	stopped := waitForState(t, events, Disconnected)
	if stopped.Previous != Error {
		t.Fatalf("stop transition = %s -> %s, want Error -> Disconnected", stopped.Previous, stopped.State)
	}
}

func TestAutomaticReconnectUsesExponentialBackoff(t *testing.T) {
	firstReconnectErr := errors.New("first reconnect failed")
	mock := newScriptedAdapter([]error{nil, firstReconnectErr, nil})
	application, err := New(mock, Config{
		AutoReconnect:  true,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	events := application.Subscribe()
	if err := application.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForState(t, events, Ready)

	mock.ReportDrop(application, errors.New("device removed"))
	drop := waitForState(t, events, Error)
	firstConnecting := waitForState(t, events, Connecting)
	firstError := waitForState(t, events, Error)
	secondConnecting := waitForState(t, events, Connecting)
	ready := waitForState(t, events, Ready)

	if firstConnecting.Attempt != 2 || firstError.Attempt != 2 || secondConnecting.Attempt != 3 {
		t.Fatalf("retry attempts = %d, %d, %d", firstConnecting.Attempt, firstError.Attempt, secondConnecting.Attempt)
	}
	if !errors.Is(firstError.Err, firstReconnectErr) {
		t.Fatalf("first reconnect error = %v, want %v", firstError.Err, firstReconnectErr)
	}
	if firstConnecting.At.Before(drop.At) || firstError.At.Before(firstConnecting.At) || secondConnecting.At.Before(firstError.At) || ready.At.Before(secondConnecting.At) {
		t.Fatalf("reconnect events are not ordered by time: drop=%s firstConnecting=%s firstError=%s secondConnecting=%s ready=%s", drop.At, firstConnecting.At, firstError.At, secondConnecting.At, ready.At)
	}
	if calls := mock.OpenCalls(); calls != 3 {
		t.Fatalf("Open() calls = %d, want 3", calls)
	}

	application.Stop()
}

func TestBackoff(t *testing.T) {
	mock := newScriptedAdapter(nil)
	application, err := New(mock, Config{
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 5 * time.Millisecond},
		{attempt: 2, want: 10 * time.Millisecond},
		{attempt: 3, want: 20 * time.Millisecond},
		{attempt: 4, want: 20 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := application.backoff(tt.attempt); got != tt.want {
			t.Errorf("backoff(%d) = %s, want %s", tt.attempt, got, tt.want)
		}
	}
}

func TestEventBus(t *testing.T) {
	bus := NewEventBusWithBuffer(1)
	events := bus.Subscribe()
	want := Event{Type: EventStateChanged, State: Ready}
	bus.Publish(want)
	select {
	case got := <-events:
		if got.Type != want.Type || got.State != want.State {
			t.Fatalf("event = %#v, want %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	bus.Close()
	if _, ok := <-events; ok {
		t.Fatal("event channel is still open")
	}
}

func waitForState(t *testing.T, events <-chan Event, state State) Event {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("event channel closed while waiting for %s", state)
			}
			if event.State == state {
				return event
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for state %s", state)
		}
	}
}

type scriptedAdapter struct {
	*adapter.Mock
	mu          sync.Mutex
	openResults []error
	openCalls   int
}

var _ adapter.Adapter = (*scriptedAdapter)(nil)

func newScriptedAdapter(results []error) *scriptedAdapter {
	return &scriptedAdapter{
		Mock:        adapter.NewMock(),
		openResults: results,
	}
}

func (m *scriptedAdapter) Open(config adapter.Config) error {
	m.mu.Lock()
	index := m.openCalls
	m.openCalls++
	var err error
	if index < len(m.openResults) {
		err = m.openResults[index]
	}
	m.mu.Unlock()
	if err != nil {
		return err
	}
	return m.Mock.Open(config)
}

func (m *scriptedAdapter) OpenCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.openCalls
}

func (m *scriptedAdapter) ReportDrop(application *App, err error) {
	application.ReportError(err)
}
