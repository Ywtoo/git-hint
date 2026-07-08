package keymap

import (
	"git-hint/engine/state"
	"testing"
)

func TestKeyHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		buffer       string
		selected     int
		wantWidget   string
		wantSelected int
	}{
		{
			name:         "Arrow Up - stay in list",
			key:          "arrowUP",
			buffer:       "git commit",
			selected:     1,
			wantWidget:   "",
			wantSelected: 0,
		},
		{
			name:         "Arrow Up - go to shell history",
			key:          "arrowUP",
			buffer:       "git commit",
			selected:     0,
			wantWidget:   "up-line-or-history",
			wantSelected: -1,
		},
		{
			name:         "Arrow Down - move down",
			key:          "arrowDOWN",
			buffer:       "git commit",
			selected:     0,
			wantWidget:   "",
			wantSelected: 1,
		},
		{
			name:         "TAB - complete",
			key:          "TAB",
			buffer:       "git com",
			selected:     0,
			wantWidget:   "",
			wantSelected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state.Selected = tt.selected
			state.Buffer = tt.buffer
			gotWidget, gotSelected := KeyHandler(tt.key)
			if gotWidget != tt.wantWidget {
				t.Errorf("KeyHandler() widget = %v, want %v", gotWidget, tt.wantWidget)
			}
			if gotSelected != tt.wantSelected {
				t.Errorf("KeyHandler() selected = %v, want %v", gotSelected, tt.wantSelected)
			}
		})
	}
}
