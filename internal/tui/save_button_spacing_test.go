package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// TestConfigMenusSaveButtonFlush pins the [ Save & Back ] button directly
// under the last content row — no blank separator line — in every config
// menu.
func TestConfigMenusSaveButtonFlush(t *testing.T) {
	generalView := func() string {
		gm := NewGeneralMenu(config.DefaultConfig())
		gm.width = 100
		gm.height = 40
		return gm.View().Content
	}()
	callbookView := func() string {
		cm := NewCallbookMenu(config.DefaultConfig())
		cm.width = 100
		cm.height = 40
		return cm.View().Content
	}()
	notifView := func() string {
		nm := NewNotificationsMenu(config.DefaultConfig())
		nm.width = 100
		nm.height = 40
		return nm.View().Content
	}()

	cases := []struct {
		name string
		view string
	}{
		{"general", generalView},
		{"callbook", callbookView},
		{"notifications", notifView},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := strings.Split(tc.view, "\n")
			btn := lineIndex(lines, "Save & Back")
			if btn < 0 {
				t.Fatal("Save & Back row missing")
			}
			if btn == 0 || lines[btn-1] == "" {
				t.Errorf("blank line before Save & Back (button at line %d)", btn)
			}
		})
	}

	// Contest form — needs a chooser with app context.
	t.Run("contest", func(t *testing.T) {
		cc := newTestContestChooser(t, map[string]config.Contest{})
		cc.startCreate()
		cc.width = 100
		cc.height = 80 // full form visible, including the Save button
		lines := strings.Split(cc.View().Content, "\n")
		btn := lineIndex(lines, "Save & Back")
		if btn < 0 {
			t.Fatal("Save & Back row missing")
		}
		if btn == 0 || lines[btn-1] == "" {
			t.Errorf("blank line before Save & Back (button at line %d)", btn)
		}
	})
}
