package main

import (
	"testing"

	traceui "github.com/baoerzuikeai/tracecli/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

func TestPreviewRendersResponsiveWidths(t *testing.T) {
	for _, width := range []int{130, 110, 90, 70} {
		model := newPreviewModel()
		model.width = width
		model.height = 32
		if view := model.View(); view == "" {
			t.Fatalf("View() is empty at width %d", width)
		}
	}
}

func TestPreviewPulseUpdatesAnimation(t *testing.T) {
	model := newPreviewModel()
	initial := model.frame
	updated, command := model.Update(pulseMsg{})
	if command == nil {
		t.Fatal("pulse update did not schedule the next tick")
	}
	if updated.(*previewModel).frame == initial {
		t.Fatal("pulse update did not advance the animation frame")
	}
}

func TestPreviewHeaderShape(t *testing.T) {
	model := newPreviewModel()
	header := model.renderHeader(120)
	if got, want := lipgloss.Height(header), 4; got != want {
		t.Fatalf("header height = %d, want %d", got, want)
	}
	if got := lipgloss.Width(header); got != 120 {
		t.Fatalf("header width = %d, want 120", got)
	}
}

func TestPreviewPinkThemeRenders(t *testing.T) {
	model := newPreviewModelWithPalette(traceui.Pink())
	if view := model.View(); view == "" {
		t.Fatal("Pink theme View() is empty")
	}
	if model.palette.Primary != "#F5C2E7" {
		t.Fatalf("preview palette primary = %s", model.palette.Primary)
	}
}

func TestFillUsesTargetDimensions(t *testing.T) {
	model := newPreviewModel()
	filled := fill(rootStyle(model.palette), 40, 6, "设备")
	if got, want := lipgloss.Width(filled), 40; got != want {
		t.Fatalf("filled width = %d, want %d", got, want)
	}
	if got, want := lipgloss.Height(filled), 6; got != want {
		t.Fatalf("filled height = %d, want %d", got, want)
	}
}

func TestPreviewStartupTransitionsToMain(t *testing.T) {
	model := newPreviewModel()
	if !model.startup || model.banner == "" {
		t.Fatal("preview did not initialize with a startup banner")
	}
	if view := model.View(); view == "" {
		t.Fatal("startup View() is empty")
	}
	updated, command := model.Update(startupDoneMsg{})
	if command == nil {
		t.Fatal("startup transition did not schedule the main pulse")
	}
	if updated.(*previewModel).startup {
		t.Fatal("startup screen did not transition to the main view")
	}
}
