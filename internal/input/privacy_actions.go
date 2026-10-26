package input

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/app"
)

// Privacy actions. Neither has a default key. A user binds them in any
// [keybindings] section, and the palette has a row for each. See
// docs/SESSIONS.md#privacy-mode.
const (
	actionTogglePrivacy     = "toggle_privacy"
	actionTogglePanePrivate = "toggle_pane_private"
)

func (d *ActionDispatcher) registerPrivacyHandlers() {
	d.Register(actionTogglePrivacy, handleTogglePrivacy)
	d.Register(actionTogglePanePrivate, handleTogglePanePrivate)
}

// handleTogglePrivacy turns this client's privacy mode on or off.
func handleTogglePrivacy(_ tea.KeyPressMsg, o *app.OS) (*app.OS, tea.Cmd) {
	o.TogglePrivacy()
	return o, nil
}

// handleTogglePanePrivate marks the focused pane private, or removes the mark.
func handleTogglePanePrivate(_ tea.KeyPressMsg, o *app.OS) (*app.OS, tea.Cmd) {
	o.TogglePanePrivate()
	return o, nil
}
