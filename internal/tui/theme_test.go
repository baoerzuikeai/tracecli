package tui

import "testing"

func TestPaletteVariants(t *testing.T) {
	catppuccin := Catppuccin()
	pink := Pink()
	if pink.Primary != "#F5C2E7" || pink.SelectedRowBackground != "#3A2438" || pink.GradientStart != "#F5C2E7" || pink.GradientEnd != "#F5E0DC" || pink.FocusBorder != "#F2CDCD" {
		t.Fatalf("Pink() overrides = %#v", pink)
	}
	if pink.Base != catppuccin.Base || pink.Mantle != catppuccin.Mantle || pink.Text != catppuccin.Text || pink.ProtocolSPI != catppuccin.ProtocolSPI || pink.ProtocolUART != catppuccin.ProtocolUART {
		t.Fatal("Pink() changed non-pink semantic colors")
	}
}
