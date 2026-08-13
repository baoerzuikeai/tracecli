package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	traceui "github.com/baoerzuikeai/tracecli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/mbndr/figlet4go"
)

const (
	pulseInterval = 420 * time.Millisecond
)

func rootStyle(p traceui.Palette) lipgloss.Style {
	return lipgloss.NewStyle().Background(p.Base).Foreground(p.Text)
}

func headerStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Background(p.Base).Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).BorderBackground(p.Base).Padding(0, 1)
}

func helpPanelStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Background(p.Base).Border(lipgloss.NormalBorder()).BorderForeground(p.Border).BorderBackground(p.Base).Padding(0, 1)
}

func statusbarStyle(p traceui.Palette) lipgloss.Style {
	return lipgloss.NewStyle().Background(p.Base).Foreground(p.Ready).Border(lipgloss.NormalBorder()).BorderForeground(p.Border).BorderBackground(p.Base).Padding(0, 1)
}

func instrumentBarStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Background(p.Base).Border(lipgloss.NormalBorder()).BorderForeground(p.InstrumentBorder).BorderBackground(p.Base).Padding(0, 1)
}

func instrumentRowStyle(p traceui.Palette, foreground lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Background(p.Base).Foreground(foreground)
}

func tableHeaderStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Foreground(p.TableHeader).Bold(true).Underline(true)
}

func selectedRowStyle(p traceui.Palette) lipgloss.Style {
	return lipgloss.NewStyle().Background(p.SelectedRowBackground).Foreground(p.Text).Bold(true)
}

func tableRowStyle(p traceui.Palette, selected bool) lipgloss.Style {
	if selected {
		return selectedRowStyle(p)
	}
	return rootStyle(p)
}

func spinnerStyle(p traceui.Palette) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p.Spinner)
}

func groupBarStyle(p traceui.Palette) lipgloss.Style {
	return lipgloss.NewStyle().Background(p.Base).Foreground(p.GroupForeground)
}

func bitFieldStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Background(p.Base).Foreground(p.Text)
}

func bitFieldSeparatorStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Background(p.Base).Foreground(p.Surface1)
}

func mutedStyle(p traceui.Palette) lipgloss.Style {
	return rootStyle(p).Foreground(p.Muted)
}

type focusPanel uint8

const (
	focusDevices focusPanel = iota
	focusRegisters
)

type previewStatus uint8

const (
	statusReady previewStatus = iota
	statusScanning
	statusError
)

type pulseMsg struct{}

type startupDoneMsg struct{}

type previewModel struct {
	width            int
	height           int
	palette          traceui.Palette
	startup          bool
	banner           string
	activeProtocol   int
	focus            focusPanel
	selectedDevice   int
	selectedGroup    int
	groupOffset      int
	selectedReg      int
	showBinary       bool
	showHelp         bool
	showDeviceDrawer bool
	errorCount       int
	status           previewStatus
	frame            int
	scanProgress     int
	tickCount        int
	traffic          []trafficSample
	protocols        []previewProtocol
	devices          []previewDevice
}

type previewProtocol struct {
	Name string
	Rate string
}

type previewDevice struct {
	Address   string
	Name      string
	Registers []previewRegister
}

type previewRegister struct {
	Address     uint16
	Value       byte
	Field       string
	Description string
	Bits        []previewBit
}

type previewBit struct {
	Name  string
	Start int
	End   int
	RW    string
}

type registerGroup struct {
	Label     string
	Registers []previewRegister
}

type trafficSample struct {
	Read  int
	Write int
}

type keyBinding struct {
	Key         string
	Description string
}

var keyBindings = []keyBinding{
	{Key: "1-9", Description: "protocol"},
	{Key: "Tab", Description: "focus next"},
	{Key: "←/→", Description: "focus"},
	{Key: "↑/↓", Description: "move"},
	{Key: "x/b", Description: "BIN"},
	{Key: "[ ]", Description: "groups"},
	{Key: "d", Description: "devices"},
	{Key: "s", Description: "scan"},
	{Key: "?", Description: "help"},
	{Key: "q/Esc", Description: "quit"},
}

func main() {
	themeName := flag.String("theme", "catppuccin", "color theme: catppuccin or pink")
	flag.Parse()
	theme, err := traceui.PaletteByName(*themeName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "preview:", err)
		os.Exit(2)
	}
	program := tea.NewProgram(newPreviewModelWithPalette(theme), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "preview:", err)
		os.Exit(1)
	}
}

func newPreviewModel() *previewModel {
	return newPreviewModelWithPalette(traceui.Catppuccin())
}

func newPreviewModelWithPalette(theme traceui.Palette) *previewModel {
	model := &previewModel{
		width:          120,
		height:         32,
		palette:        theme,
		startup:        true,
		banner:         figletBanner(),
		focus:          focusRegisters,
		selectedDevice: 1,
		status:         statusReady,
		traffic:        fakeTraffic(),
		protocols: []previewProtocol{
			{Name: "I2C", Rate: "400kHz"},
			{Name: "SPI", Rate: "1MHz"},
			{Name: "UART", Rate: "115200"},
		},
		devices: fakeDevices(),
	}
	model.resetSelection()
	return model
}

func fakeDevices() []previewDevice {
	return []previewDevice{
		{
			Address: "0x50",
			Name:    "EEPROM AT24C32A",
			Registers: []previewRegister{
				{Address: 0x00, Value: 0x5A, Field: "DATA_00", Description: "configuration byte"},
				{Address: 0x10, Value: 0x24, Field: "DATA_10", Description: "calibration data"},
				{Address: 0x20, Value: 0x00, Field: "STATUS", Description: "write cycle status"},
			},
		},
		{
			Address: "0x68",
			Name:    "MPU6050",
			Registers: []previewRegister{
				{Address: 0x00, Value: 0x68, Field: "WHO_AM_I", Description: "device identification"},
				{Address: 0x20, Value: 0x1A, Field: "TEMP_OUT_H", Description: "temperature high byte"},
				{Address: 0x40, Value: 0x18, Field: "GYRO_CONFIG", Description: "gyroscope full scale", Bits: []previewBit{
					{Name: "FS_SEL", Start: 4, End: 3, RW: "rw"},
				}},
				{Address: 0x6B, Value: 0x40, Field: "PWR_MGMT_1", Description: "power management", Bits: []previewBit{
					{Name: "DEVICE_RESET", Start: 7, End: 7, RW: "w"},
					{Name: "SLEEP", Start: 6, End: 6, RW: "rw"},
					{Name: "CLKSEL", Start: 2, End: 0, RW: "rw"},
				}},
				{Address: 0x75, Value: 0x68, Field: "WHO_AM_I", Description: "identity register"},
			},
		},
		{
			Address: "0x7C",
			Name:    "Unknown@0x7C",
			Registers: []previewRegister{
				{Address: 0x00, Value: 0xFF, Field: "REG_00", Description: "unidentified register"},
			},
		},
	}
}

func fakeTraffic() []trafficSample {
	samples := make([]trafficSample, 60)
	for index := range samples {
		samples[index] = trafficSample{
			Read:  (index*5 + 3) % 9,
			Write: (index*3 + index/7 + 1) % 8,
		}
	}
	return samples
}

func figletBanner() string {
	renderer := figlet4go.NewAsciiRender()
	options := figlet4go.NewRenderOptions()
	options.FontName = "larry3d"
	banner, err := renderer.RenderOpts("TRACECLI", options)
	if err != nil {
		return "TRACECLI"
	}
	return strings.TrimRight(banner, "\r\n")
}

func (m *previewModel) Init() tea.Cmd {
	return startupCmd()
}

func startupCmd() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return startupDoneMsg{}
	})
}

func pulseCmd() tea.Cmd {
	return tea.Tick(pulseInterval, func(time.Time) tea.Msg {
		return pulseMsg{}
	})
}

func (m *previewModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case startupDoneMsg:
		m.startup = false
		return m, pulseCmd()
	case pulseMsg:
		m.frame = (m.frame + 1) % 4
		m.tickCount++
		if m.status == statusScanning {
			m.scanProgress += 9
			if m.scanProgress >= 100 {
				m.scanProgress = 100
				m.status = statusReady
			}
		}
		if m.tickCount%2 == 0 {
			m.advanceTraffic()
		}
		return m, pulseCmd()
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		if m.width < 120 {
			m.showBinary = false
		}
	case tea.KeyMsg:
		switch message.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 2
		case "shift+tab":
			m.focus = (m.focus + 1) % 2
		case "right", "l":
			m.focus = focusRegisters
		case "left", "h":
			m.focus = focusDevices
		case "x", "b":
			if m.width >= 120 {
				m.showBinary = !m.showBinary
			}
		case "up", "k":
			m.moveFocus(-1)
		case "down", "j":
			m.moveFocus(1)
		case "s":
			m.status = statusScanning
			m.scanProgress = 0
		case "e":
			m.status = statusError
			m.errorCount++
		case "r":
			m.status = statusReady
		case "[":
			m.selectGroup(-1)
		case "]":
			m.selectGroup(1)
		case "?":
			m.showHelp = !m.showHelp
		case "d":
			m.showDeviceDrawer = !m.showDeviceDrawer
		default:
			if message.String() >= "1" && message.String() <= "9" {
				index, err := strconv.Atoi(message.String())
				if err == nil && index <= len(m.protocols) {
					m.selectProtocol(index - 1 - m.activeProtocol)
				}
			}
		}
	}
	return m, nil
}

func (m *previewModel) View() string {
	if m.startup {
		return m.startupView()
	}

	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 32
	}

	header := m.renderHeader(width)
	instrumentRows := 0
	if width >= 100 {
		instrumentRows = 1
		if width >= 120 {
			instrumentRows = 2
		}
	}
	helpRows := 1
	if m.showHelp {
		helpRows = 2
	}
	headerRows := 2 + headerStyle(m.palette).GetVerticalFrameSize()
	mainHeight := maxInt(8, height-headerRows-(instrumentRows+2)-(helpRows+2)-3)
	main := m.renderMain(width, mainHeight)
	parts := []string{header, main}
	if width >= 100 {
		parts = append(parts, m.renderInstrumentBar(width, width < 120))
	}
	parts = append(parts, renderFixed(helpPanelStyle(m.palette), width, helpRows, m.helpText(width)), m.renderStatus(width))
	content := strings.Join(parts, "\n")
	return fill(rootStyle(m.palette), width, height, content)
}

func (m *previewModel) startupView() string {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 32
	}

	bannerStyle := lipgloss.NewStyle().Foreground(m.palette.Primary).Bold(true)
	bannerLines := strings.Split(m.banner, "\n")
	centered := make([]string, 0, len(bannerLines))
	for _, line := range bannerLines {
		centered = append(centered, lipgloss.PlaceHorizontal(width, lipgloss.Center, bannerStyle.Render(line)))
	}
	loading := lipgloss.NewStyle().Foreground(m.palette.Muted).Render("loading  ·  initializing preview")
	centered = append(centered, "", lipgloss.PlaceHorizontal(width, lipgloss.Center, loading))
	top := maxInt(0, (height-len(centered))/2)
	bottom := maxInt(0, height-top-len(centered))
	content := strings.Repeat("\n", top) + strings.Join(centered, "\n") + strings.Repeat("\n", bottom)
	return fill(rootStyle(m.palette), width, height, content)
}

func (m *previewModel) renderHeader(width int) string {
	containerStyle := headerStyle(m.palette)
	innerWidth := maxInt(1, width-containerStyle.GetHorizontalFrameSize())
	line1 := lipgloss.NewStyle().Foreground(m.palette.Primary).Bold(true).Render("tracecli")

	tabs := make([]string, 0, len(m.protocols))
	for index, protocol := range m.protocols {
		label := "▌" + fmt.Sprintf("%d: %s", index+1, protocol.Name) + "▐"
		style := lipgloss.NewStyle().Foreground(m.palette.Muted)
		if index == m.activeProtocol {
			style = style.
				Foreground(m.palette.Primary).
				Bold(true)
		}
		tabs = append(tabs, style.Render(label))
	}
	address := "ADDR:--"
	if device := m.currentDevice(); device != nil {
		address = "ADDR:" + device.Address
	}
	state := m.statusIndicator() + " " + m.statusLabel()
	right := state + "  " + m.activeRate() + "  " + address + "  " + m.errorBadge() + "  [?] HELP"
	line2 := alignLine(innerWidth, strings.Join(tabs, " "), right)
	return fill(containerStyle, width, 2+containerStyle.GetVerticalFrameSize(), line1+"\n"+line2)
}

func (m *previewModel) renderMain(width, height int) string {
	showBinary := m.showBinary && width >= 120
	if width >= 80 {
		deviceWidth := width * 25 / 100
		return joinColumns(m.renderDevicePanel(deviceWidth, height), m.renderRegisterPanel(width-deviceWidth, height, showBinary, width < 100))
	}
	if m.showDeviceDrawer {
		deviceHeight := maxInt(5, height/3)
		return strings.Join([]string{
			m.renderDevicePanel(width, deviceHeight),
			m.renderRegisterPanel(width, height-deviceHeight, false, true),
		}, "\n")
	}
	return m.renderRegisterPanel(width, height, false, true)
}

func (m *previewModel) renderDevicePanel(width, height int) string {
	style := panelStyle(m.palette, m.palette.DeviceBorder, m.focus == focusDevices, 1)
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	lines := []string{panelTitle(m.palette, "DEVICES", m.palette.DeviceBorder, m.focus == focusDevices, innerWidth)}
	if m.activeProtocol != 0 {
		lines = append(lines, fill(mutedStyle(m.palette), innerWidth, 1, "No preview targets"))
	} else {
		for index, device := range m.devices {
			line := fmt.Sprintf("%-5s %s", device.Address, truncateText(device.Name, maxInt(1, width-10)))
			if index == m.selectedDevice {
				lines = append(lines, fill(selectedRowStyle(m.palette), innerWidth, 1, truncateText(line, innerWidth)))
			} else {
				lines = append(lines, fill(rootStyle(m.palette), innerWidth, 1, truncateText(line, innerWidth)))
			}
		}
	}
	return fill(style, width, height, strings.Join(lines, "\n"))
}

func (m *previewModel) renderRegisterPanel(width, height int, showBinary, showGroup bool) string {
	focused := m.focus == focusRegisters
	style := panelStyle(m.palette, m.palette.RegisterBorder, focused, 2)
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	lines := []string{panelTitle(m.palette, "REGISTERS", m.palette.RegisterBorder, focused, innerWidth)}
	lines = append(lines, m.renderGroupBar(innerWidth, showGroup))
	mode := "[R] READ  [W] WRITE  [x/b] BIN: " + map[bool]string{true: "ON", false: "OFF"}[showBinary]
	lines = append(lines, fill(mutedStyle(m.palette), innerWidth, 1, mode))
	lines = append(lines, fill(tableHeaderStyle(m.palette), innerWidth, 1, tableHeader(innerWidth, showBinary)))

	var selected *previewRegister
	if group := m.currentGroup(); group != nil {
		for index, register := range group.Registers {
			line := tableRow(register, showBinary, innerWidth)
			isSelected := index == m.selectedReg
			lines = append(lines, fill(tableRowStyle(m.palette, isSelected), innerWidth, 1, line))
			if isSelected {
				selected = &group.Registers[index]
			}
		}
	} else {
		lines = append(lines, fill(mutedStyle(m.palette), innerWidth, 1, "No registers for this protocol"))
	}
	lines = append(lines, bitFieldDetail(m.palette, selected, innerWidth))
	return fill(style, width, height, strings.Join(lines, "\n"))
}

func (m *previewModel) renderGroupBar(width int, collapsed bool) string {
	groups := groupsFor(m.currentDevice())
	if len(groups) == 0 {
		return fill(groupBarStyle(m.palette), width, 1, "GROUPS: none")
	}
	if m.selectedGroup >= len(groups) {
		m.selectedGroup = 0
	}
	if collapsed {
		return fill(groupBarStyle(m.palette), width, 1, "GROUP "+groups[m.selectedGroup].Label+" ▾")
	}

	parts := make([]string, 0, len(groups))
	remaining := width
	for index := m.groupOffset; index < len(groups); index++ {
		group := groups[index]
		label := "  " + group.Label
		if index == m.selectedGroup {
			label = "▸ " + group.Label
		}
		pillWidth := lipgloss.Width(label)
		if pillWidth > remaining && len(parts) > 0 {
			break
		}
		parts = append(parts, label)
		remaining -= pillWidth + 1
	}
	return fill(groupBarStyle(m.palette), width, 1, strings.Join(parts, " "))
}

func (m *previewModel) renderStatus(width int) string {
	rate := m.activeRate()
	status := "状态:" + m.statusLabelCN() + " | 速率:" + rate + " | 包:1204 OK"
	if m.status == statusScanning {
		status = "状态:" + m.statusLabelCN() + " " + scanBar(m.scanProgress) + " | 速率:" + rate + " | 包:1204 OK"
	}
	if m.status == statusError {
		status = "状态:" + m.statusLabelCN() + " | 速率:" + rate + " | 包:1204 ERR"
	}
	return renderFixed(statusbarStyle(m.palette).Foreground(m.statusColor()), width, 1, status)
}

func (m *previewModel) renderInstrumentBar(width int, compact bool) string {
	style := instrumentBarStyle(m.palette)
	lines := trafficLines(m.traffic, maxInt(1, width-style.GetHorizontalFrameSize()))
	if compact {
		content := lines[0] + "  " + strings.TrimSpace(lines[1])
		return renderFixed(style, width, 1, content)
	}
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	readLine := fill(instrumentRowStyle(m.palette, m.palette.TrafficRead), innerWidth, 1, lines[0])
	writeLine := fill(instrumentRowStyle(m.palette, m.palette.TrafficWrite), innerWidth, 1, lines[1])
	return fill(style, width, 2+style.GetVerticalFrameSize(), readLine+"\n"+writeLine)
}

func (m *previewModel) helpText(width int) string {
	bindings := keyBindings
	if !m.showHelp && len(bindings) > 5 {
		bindings = bindings[:5]
	}
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		parts = append(parts, binding.Key+" "+binding.Description)
	}
	text := strings.Join(parts, "   ")
	if !m.showHelp {
		return truncateText(text, width-4)
	}
	return wrapText(text, maxInt(1, width-4), 2)
}

func (m *previewModel) statusIndicator() string {
	switch m.status {
	case statusScanning:
		return spinnerStyle(m.palette).Render([]string{"●", "○", "◐", "◑"}[m.frame])
	case statusError:
		if m.frame%2 == 0 {
			return lipgloss.NewStyle().Foreground(m.palette.Error).Render("●")
		}
		return lipgloss.NewStyle().Foreground(m.palette.Error).Render("○")
	default:
		return lipgloss.NewStyle().Foreground(m.palette.Ready).Render("●")
	}
}

func (m *previewModel) statusLabel() string {
	switch m.status {
	case statusScanning:
		return "SCANNING"
	case statusError:
		return "ERROR"
	default:
		return "CONNECTED"
	}
}

func (m *previewModel) statusLabelCN() string {
	switch m.status {
	case statusScanning:
		return "扫描中"
	case statusError:
		return "错误"
	default:
		return "就绪"
	}
}

func (m *previewModel) statusColor() lipgloss.Color {
	switch m.status {
	case statusScanning:
		return m.palette.Scan
	case statusError:
		return m.palette.Error
	default:
		return m.palette.Ready
	}
}

func (m *previewModel) errorBadge() string {
	if m.errorCount > 0 {
		return "ERR:" + strconv.Itoa(m.errorCount)
	}
	return "ERR:0"
}

func (m *previewModel) activeRate() string {
	if m.activeProtocol < 0 || m.activeProtocol >= len(m.protocols) {
		return "--"
	}
	return m.protocols[m.activeProtocol].Rate
}

func (m *previewModel) currentDevice() *previewDevice {
	if m.activeProtocol != 0 || len(m.devices) == 0 {
		return nil
	}
	if m.selectedDevice < 0 || m.selectedDevice >= len(m.devices) {
		return nil
	}
	return &m.devices[m.selectedDevice]
}

func (m *previewModel) currentGroup() *registerGroup {
	device := m.currentDevice()
	if device == nil {
		return nil
	}
	groups := groupsFor(device)
	if m.selectedGroup < 0 || m.selectedGroup >= len(groups) {
		return nil
	}
	return &groups[m.selectedGroup]
}

func (m *previewModel) selectProtocol(delta int) {
	if len(m.protocols) == 0 {
		return
	}
	m.activeProtocol = (m.activeProtocol + delta + len(m.protocols)) % len(m.protocols)
	m.resetSelection()
}

func (m *previewModel) resetSelection() {
	m.selectedDevice = 0
	m.selectedGroup = 0
	m.groupOffset = 0
	m.selectedReg = 0
	if m.activeProtocol == 0 && len(m.devices) > 1 {
		m.selectedDevice = 1
	}
	if device := m.currentDevice(); device != nil {
		for index, group := range groupsFor(device) {
			for _, register := range group.Registers {
				if len(register.Bits) > 0 {
					m.selectedGroup = index
					return
				}
			}
		}
	}
}

func (m *previewModel) moveFocus(delta int) {
	switch m.focus {
	case focusDevices:
		m.selectDevice(delta)
	case focusRegisters:
		m.selectRegister(delta)
	}
}

func (m *previewModel) selectDevice(delta int) {
	if m.activeProtocol != 0 || len(m.devices) == 0 {
		return
	}
	m.selectedDevice = (m.selectedDevice + delta + len(m.devices)) % len(m.devices)
	m.selectedGroup = 0
	m.selectedReg = 0
}

func (m *previewModel) selectGroup(delta int) {
	groups := groupsFor(m.currentDevice())
	if len(groups) == 0 {
		return
	}
	m.selectedGroup = (m.selectedGroup + delta + len(groups)) % len(groups)
	m.groupOffset = m.selectedGroup
	m.selectedReg = 0
}

func (m *previewModel) selectRegister(delta int) {
	group := m.currentGroup()
	if group == nil || len(group.Registers) == 0 {
		return
	}
	m.selectedReg = (m.selectedReg + delta + len(group.Registers)) % len(group.Registers)
}

func (m *previewModel) advanceTraffic() {
	if len(m.traffic) >= 60 {
		m.traffic = m.traffic[1:]
	}
	index := m.tickCount
	m.traffic = append(m.traffic, trafficSample{
		Read:  (index*5 + 3) % 9,
		Write: (index*3 + index/7 + 1) % 8,
	})
}

func groupsFor(device *previewDevice) []registerGroup {
	if device == nil {
		return nil
	}
	grouped := make(map[uint16][]previewRegister)
	for _, register := range device.Registers {
		base := register.Address &^ 0x1F
		grouped[base] = append(grouped[base], register)
	}
	bases := make([]int, 0, len(grouped))
	for base := range grouped {
		bases = append(bases, int(base))
	}
	sort.Ints(bases)
	groups := make([]registerGroup, 0, len(bases))
	for _, base := range bases {
		start := uint16(base)
		groups = append(groups, registerGroup{
			Label:     fmt.Sprintf("%02X-%02X", start, start+0x1F),
			Registers: grouped[start],
		})
	}
	return groups
}

func panelStyle(p traceui.Palette, color lipgloss.Color, focused bool, padding int) lipgloss.Style {
	border := color
	if focused {
		border = p.FocusBorder
	}
	return rootStyle(p).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		BorderBackground(p.Base).
		Padding(0, padding)
}

func panelTitle(p traceui.Palette, label string, color lipgloss.Color, focused bool, width int) string {
	style := lipgloss.NewStyle().Background(p.Base).Foreground(color).Bold(true)
	if focused {
		style = style.Foreground(p.FocusBorder).Padding(0, 1)
	}
	return fill(style, width, 1, label)
}

func tableHeader(width int, showBinary bool) string {
	if showBinary {
		return truncateText(fmt.Sprintf("%-6s %-6s %-11s %-16s %s", "ADDR", "HEX", "BIN", "FIELD", "描述"), width)
	}
	return truncateText(fmt.Sprintf("%-6s %-6s %-16s %s", "ADDR", "HEX", "FIELD", "描述"), width)
}

func tableRow(register previewRegister, showBinary bool, width int) string {
	if showBinary {
		return truncateText(fmt.Sprintf("%-6s %-6s %-11s %-16s %s", fmt.Sprintf("0x%02X", register.Address), fmt.Sprintf("0x%02X", register.Value), binaryValue(register.Value), register.Field, register.Description), width)
	}
	return truncateText(fmt.Sprintf("%-6s %-6s %-16s %s", fmt.Sprintf("0x%02X", register.Address), fmt.Sprintf("0x%02X", register.Value), register.Field, register.Description), width)
}

func bitFieldDetail(p traceui.Palette, register *previewRegister, width int) string {
	separator := "  " + strings.Repeat("─", maxInt(0, width-2))
	if register == nil {
		return fill(bitFieldSeparatorStyle(p), width, 1, separator) + "\n" + fill(bitFieldStyle(p), width, 3, "")
	}
	blocks := make([]string, 0, 8)
	labels := make([]string, 0, len(register.Bits))
	for bit := 7; bit >= 0; bit-- {
		if register.Value&(1<<bit) != 0 {
			blocks = append(blocks, lipgloss.NewStyle().Foreground(p.BitOn).Render("█"))
		} else {
			blocks = append(blocks, lipgloss.NewStyle().Foreground(p.BitOff).Render("░"))
		}
	}
	for _, field := range register.Bits {
		rangeText := strconv.Itoa(field.Start)
		if field.Start != field.End {
			rangeText += ":" + strconv.Itoa(field.End)
		}
		name := lipgloss.NewStyle().Foreground(p.Number).Bold(true).Render(field.Name + "(" + rangeText + ")")
		labels = append(labels, name+" "+lipgloss.NewStyle().Foreground(p.Code).Render(field.RW))
	}
	lines := []string{
		"  " + fmt.Sprintf("0x%02X = 0x%02X  %s", register.Address, register.Value, register.Field),
		"  " + strings.Join(blocks, " "),
		"  " + strings.Join(labels, "  "),
	}
	return fill(bitFieldSeparatorStyle(p), width, 1, separator) + "\n" + fill(bitFieldStyle(p), width, 3, strings.Join(lines, "\n"))
}

func trafficLines(samples []trafficSample, width int) []string {
	if len(samples) == 0 {
		return []string{"INSTRUMENT 60s  no traffic data", ""}
	}
	graphWidth := maxInt(1, width-5)
	read := make([]int, len(samples))
	write := make([]int, len(samples))
	for index, sample := range samples {
		read[index] = sample.Read
		write[index] = sample.Write
	}
	return []string{
		"INSTRUMENT 60s  R " + sparkline(read, graphWidth),
		"W " + sparkline(write, graphWidth),
	}
}

func sparkline(values []int, width int) string {
	glyphs := []rune("▁▂▃▄▅▆▇█")
	if len(values) > width {
		values = values[len(values)-width:]
	}
	var line strings.Builder
	for _, value := range values {
		if value < 0 {
			value = 0
		}
		if value >= len(glyphs) {
			value = len(glyphs) - 1
		}
		line.WriteRune(glyphs[value])
	}
	return line.String()
}

func scanBar(progress int) string {
	filled := progress * 5 / 100
	return strings.Repeat("▰", filled) + strings.Repeat("▱", 5-filled) + " " + strconv.Itoa(progress) + "%"
}

func binaryValue(value byte) string {
	return fmt.Sprintf("%08b", value)
}

func joinColumns(columns ...string) string {
	lineSets := make([][]string, len(columns))
	height := 0
	for index, column := range columns {
		lineSets[index] = strings.Split(column, "\n")
		if len(lineSets[index]) > height {
			height = len(lineSets[index])
		}
	}

	lines := make([]string, height)
	for row := range lines {
		var line strings.Builder
		for _, set := range lineSets {
			if row < len(set) {
				line.WriteString(set[row])
			}
		}
		lines[row] = line.String()
	}
	return strings.Join(lines, "\n")
}

func fill(style lipgloss.Style, width, height int, content string) string {
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(1, height-style.GetVerticalFrameSize())
	lines := strings.Split(content, "\n")
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	for index, line := range lines {
		lines[index] = fillLine(line, innerWidth)
	}
	filled := lipgloss.PlaceVertical(innerHeight, lipgloss.Top, strings.Join(lines, "\n"))
	styleWidth := innerWidth + style.GetHorizontalPadding()
	styleHeight := innerHeight + style.GetVerticalPadding()
	return style.Width(styleWidth).Height(styleHeight).Render(filled)
}

func renderFixed(style lipgloss.Style, width, height int, content string) string {
	return fill(style, width, height+style.GetVerticalFrameSize(), content)
}

func fillLine(line string, width int) string {
	line = ansi.Truncate(line, width, "...")
	used := runewidth.StringWidth(ansi.Strip(line))
	if used < width {
		line += strings.Repeat(" ", width-used)
	}
	return line
}

func alignLine(width int, left, right string) string {
	left = truncateText(left, width)
	right = truncateText(right, width)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncateText(left+" "+right, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	result := ""
	for _, character := range value {
		candidate := result + string(character)
		if lipgloss.Width(candidate+"...") > width {
			break
		}
		result = candidate
	}
	return result + "..."
}

func wrapText(value string, width, rows int) string {
	if width <= 0 || rows <= 0 {
		return ""
	}
	words := strings.Fields(value)
	lines := make([]string, 0, rows)
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + "   " + word
		}
		if lipgloss.Width(candidate) > width && line != "" {
			lines = append(lines, line)
			line = word
			if len(lines) == rows-1 {
				break
			}
			continue
		}
		line = candidate
	}
	if len(lines) < rows && line != "" {
		lines = append(lines, line)
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	return strings.Join(lines, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
