package tui

import (
	"strings"
	"testing"

	"github.com/baoerzuikeai/tracecli/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelTabsAndView(t *testing.T) {
	model := New(nil, []string{"spi", "i2c"})
	if got, want := model.ActiveProtocol(), "i2c"; got != want {
		t.Fatalf("initial ActiveProtocol() = %q, want %q", got, want)
	}
	view := model.View()
	for _, text := range []string{"Connection: Disconnected", "[ i2c ]", "spi"} {
		if !strings.Contains(view, text) {
			t.Fatalf("initial View() does not contain %q: %s", text, view)
		}
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(*Model)
	if got, want := model.ActiveProtocol(), "spi"; got != want {
		t.Fatalf("right ActiveProtocol() = %q, want %q", got, want)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(*Model)
	if got, want := model.ActiveProtocol(), "i2c"; got != want {
		t.Fatalf("left ActiveProtocol() = %q, want %q", got, want)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model = updated.(*Model)
	if got, want := model.ActiveProtocol(), "spi"; got != want {
		t.Fatalf("number ActiveProtocol() = %q, want %q", got, want)
	}
}

func TestModelSubscribesToStateEvents(t *testing.T) {
	events := make(chan app.Event, 1)
	model := New(events, []string{"i2c"})
	command := model.Init()
	if command == nil {
		t.Fatal("Init() returned nil command")
	}
	events <- app.Event{
		Type:  app.EventStateChanged,
		State: app.Ready,
	}
	message := command()
	updated, next := model.Update(message)
	model = updated.(*Model)
	if model.State() != app.Ready {
		t.Fatalf("State() = %s, want %s", model.State(), app.Ready)
	}
	if next == nil {
		t.Fatal("state event did not resubscribe")
	}
	if !strings.Contains(model.View(), "Connection: Ready") {
		t.Fatalf("View() does not show Ready: %s", model.View())
	}
}

func TestModelDisplaysStateErrorAndHandlesClosedEvents(t *testing.T) {
	events := make(chan app.Event, 1)
	model := New(events, []string{"i2c"})
	events <- app.Event{
		Type:  app.EventStateChanged,
		State: app.Error,
		Err:   errTestConnection,
	}
	message := model.Init()()
	updated, _ := model.Update(message)
	model = updated.(*Model)
	if !strings.Contains(model.View(), "Connection: Error (connection failed)") {
		t.Fatalf("View() does not show error: %s", model.View())
	}

	close(events)
	message = model.Init()()
	updated, next := model.Update(message)
	model = updated.(*Model)
	if next != nil || model.events != nil {
		t.Fatal("closed event stream was not handled")
	}
}

var errTestConnection = testError("connection failed")

type testError string

func (e testError) Error() string {
	return string(e)
}
