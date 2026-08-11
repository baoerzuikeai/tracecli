package tui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/baoerzuikeai/tracecli/internal/app"
	"github.com/baoerzuikeai/tracecli/internal/device"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	events      <-chan app.Event
	protocols   []string
	activeTab   int
	state       app.State
	lastError   string
	devMapCount int
	width       int
	height      int
}

var _ tea.Model = (*Model)(nil)

func New(events <-chan app.Event, protocols []string) *Model {
	protocols = append([]string(nil), protocols...)
	sort.Strings(protocols)
	return &Model{
		events:    events,
		protocols: protocols,
		state:     app.Disconnected,
	}
}

func NewWithApp(application *app.App, protocols []string, index *device.Index) *Model {
	var events <-chan app.Event
	if application != nil {
		events = application.Subscribe()
	}

	model := New(events, protocols)
	if index != nil {
		model.devMapCount = len(index.All())
	}
	return model
}

func (m *Model) Init() tea.Cmd {
	return waitForEvent(m.events)
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case app.Event:
		if message.Type == app.EventStateChanged {
			m.state = message.State
			if message.Err != nil {
				m.lastError = message.Err.Error()
			} else if message.State == app.Ready {
				m.lastError = ""
			}
		}
		return m, waitForEvent(m.events)
	case eventStreamClosed:
		m.events = nil
		return m, nil
	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "right", "l":
			m.selectTab(1)
		case "shift+tab", "left", "h":
			m.selectTab(-1)
		default:
			if message.String() >= "1" && message.String() <= "9" {
				index, err := strconv.Atoi(message.String())
				if err == nil && index <= len(m.protocols) {
					m.activeTab = index - 1
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	}
	return m, nil
}

func (m *Model) View() string {
	var view strings.Builder
	view.WriteString("tracecli\n\n")
	view.WriteString("Connection: ")
	view.WriteString(m.state.String())
	if m.lastError != "" {
		view.WriteString(" (" + m.lastError + ")")
	}
	view.WriteString("\n")
	if m.devMapCount > 0 {
		view.WriteString("Device maps: ")
		view.WriteString(strconv.Itoa(m.devMapCount))
		view.WriteString("\n")
	}
	view.WriteString("\n")

	for index, protocol := range m.protocols {
		if index > 0 {
			view.WriteString("  ")
		}
		if index == m.activeTab {
			view.WriteString("[ ")
			view.WriteString(protocol)
			view.WriteString(" ]")
		} else {
			view.WriteString("  ")
			view.WriteString(protocol)
			view.WriteString("  ")
		}
	}
	view.WriteString("\n\n")
	view.WriteString("Tab/Left/Right: switch protocol    q: quit\n")
	return view.String()
}

func (m *Model) ActiveTab() int {
	return m.activeTab
}

func (m *Model) ActiveProtocol() string {
	if m.activeTab < 0 || m.activeTab >= len(m.protocols) {
		return ""
	}
	return m.protocols[m.activeTab]
}

func (m *Model) State() app.State {
	return m.state
}

func (m *Model) selectTab(delta int) {
	if len(m.protocols) == 0 {
		return
	}
	m.activeTab = (m.activeTab + delta + len(m.protocols)) % len(m.protocols)
}

func waitForEvent(events <-chan app.Event) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return eventStreamClosed{}
		}
		return event
	}
}

type eventStreamClosed struct{}
