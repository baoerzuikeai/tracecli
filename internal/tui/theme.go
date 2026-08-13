package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Palette struct {
	Base                   lipgloss.Color
	Mantle                 lipgloss.Color
	Crust                  lipgloss.Color
	Text                   lipgloss.Color
	Muted                  lipgloss.Color
	Primary                lipgloss.Color
	SelectedRowBackground  lipgloss.Color
	Scan                   lipgloss.Color
	Ready                  lipgloss.Color
	Error                  lipgloss.Color
	FocusBorder            lipgloss.Color
	Border                 lipgloss.Color
	Surface0               lipgloss.Color
	Surface1               lipgloss.Color
	Number                 lipgloss.Color
	Code                   lipgloss.Color
	Spinner                lipgloss.Color
	Pink                   lipgloss.Color
	Rosewater              lipgloss.Color
	Flamingo               lipgloss.Color
	GradientStart          lipgloss.Color
	GradientEnd            lipgloss.Color
	DeviceBorder           lipgloss.Color
	RegisterBorder         lipgloss.Color
	InstrumentBorder       lipgloss.Color
	TableHeader            lipgloss.Color
	GroupForeground        lipgloss.Color
	TrafficRead            lipgloss.Color
	TrafficWrite           lipgloss.Color
	BitOn                  lipgloss.Color
	BitOff                 lipgloss.Color
	ToastErrorBackground   lipgloss.Color
	ToastErrorForeground   lipgloss.Color
	ToastSuccessBackground lipgloss.Color
	ToastSuccessForeground lipgloss.Color
	ProtocolI2C            lipgloss.Color
	ProtocolSPI            lipgloss.Color
	ProtocolUART           lipgloss.Color
}

func Catppuccin() Palette {
	return Palette{
		Base:                   "#1E1E2E",
		Mantle:                 "#181825",
		Crust:                  "#11111B",
		Text:                   "#CDD6F4",
		Muted:                  "#6C7086",
		Primary:                "#CBA6F7",
		SelectedRowBackground:  "#313244",
		Scan:                   "#89DCEB",
		Ready:                  "#A6E3A1",
		Error:                  "#F38BA8",
		FocusBorder:            "#B4BEFE",
		Border:                 "#45475A",
		Surface0:               "#313244",
		Surface1:               "#45475A",
		Number:                 "#F9E2AF",
		Code:                   "#89B4FA",
		Spinner:                "#89DCEB",
		Pink:                   "#F5C2E7",
		Rosewater:              "#F5E0DC",
		Flamingo:               "#F2CDCD",
		GradientStart:          "#CBA6F7",
		GradientEnd:            "#89DCEB",
		DeviceBorder:           "#CBA6F7",
		RegisterBorder:         "#94E2D5",
		InstrumentBorder:       "#45475A",
		TableHeader:            "#94E2D5",
		GroupForeground:        "#CBA6F7",
		TrafficRead:            "#89B4FA",
		TrafficWrite:           "#A6E3A1",
		BitOn:                  "#A6E3A1",
		BitOff:                 "#45475A",
		ToastErrorBackground:   "#1E1E2E",
		ToastErrorForeground:   "#F38BA8",
		ToastSuccessBackground: "#1E1E2E",
		ToastSuccessForeground: "#A6E3A1",
		ProtocolI2C:            "#CBA6F7",
		ProtocolSPI:            "#89B4FA",
		ProtocolUART:           "#94E2D5",
	}
}

func Pink() Palette {
	palette := Catppuccin()
	palette.Primary = palette.Pink
	palette.SelectedRowBackground = "#3A2438"
	palette.GradientStart = palette.Pink
	palette.GradientEnd = palette.Rosewater
	palette.FocusBorder = palette.Flamingo
	palette.DeviceBorder = palette.Primary
	return palette
}

func PaletteByName(name string) (Palette, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "catppuccin", "":
		return Catppuccin(), nil
	case "pink":
		return Pink(), nil
	default:
		return Palette{}, fmt.Errorf("unknown theme %q", name)
	}
}

func CatppuccinMocha() Palette {
	return Catppuccin()
}
