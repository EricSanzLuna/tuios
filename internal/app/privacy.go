package app

import "github.com/Gaurav-Gosain/tuios/internal/terminal"

// Privacy mode: while it is on, every pane marked private (Window.Private) is
// drawn as a placeholder, and nothing it printed reaches another surface. The
// marks are session state; the mode is this client's own (OS.privacy) and is
// never saved. See docs/SESSIONS.md#privacy-mode.

// privatePaneLabel is what the placeholder shows for a pane with no name of
// its own. The title a program sets is pane output, so it is never used.
const privatePaneLabel = "Private pane"

// privatePaneBadge is what the title bar of a private pane carries after its
// name while the mode is off. Text, like markedPaneBadge, so it reads on every
// theme and a test can find it.
const privatePaneBadge = "[private]"

// PrivacyOn reports whether this client is in privacy mode.
func (m *OS) PrivacyOn() bool { return m.privacy }

// hidesPane reports whether w is to be drawn as a placeholder and kept off
// every other surface: the mode is on and the pane is marked private.
func (m *OS) hidesPane(w *terminal.Window) bool {
	return m.privacy && w != nil && w.Private
}

// TogglePrivacy is toggle_privacy: it turns this client's privacy mode on or
// off.
func (m *OS) TogglePrivacy() { m.SetPrivacy(!m.privacy) }

// SetPrivacy turns this client's privacy mode on or off and says so. Nothing
// is sent to the daemon: the mode is this client's own.
func (m *OS) SetPrivacy(on bool) {
	if m.privacy != on {
		m.privacy = on
		// Every cached frame may hold a private pane's cells, or its
		// placeholder.
		m.MarkAllDirty()
	}
	if on {
		m.ShowNotification("Privacy mode on: private panes are hidden", "info", m.Settings.NotificationDuration)
	} else {
		m.ShowNotification("Privacy mode off", "info", m.Settings.NotificationDuration)
	}
}

// TogglePanePrivate is toggle_pane_private: it marks the focused pane private,
// or removes the mark. The mark is session state, so it is pushed to the
// daemon at once.
func (m *OS) TogglePanePrivate() {
	w := m.GetFocusedWindow()
	if w == nil {
		return
	}
	w.Private = !w.Private
	m.MarkAllDirty()
	name := w.CustomName
	if name == "" {
		name = "This pane"
	}
	switch {
	case w.Private && m.privacy:
		m.ShowNotification(name+" is private and hidden", "info", m.Settings.NotificationDuration)
	case w.Private:
		m.ShowNotification(name+" is private: hidden while privacy mode is on", "info", m.Settings.NotificationDuration)
	default:
		m.ShowNotification(name+" is no longer private", "info", m.Settings.NotificationDuration)
	}
	m.SyncStateToDaemon()
}

// privateLabel is the name a hidden pane shows: the name the user gave it, or
// privatePaneLabel.
func privateLabel(w *terminal.Window) string {
	if w != nil && w.CustomName != "" {
		return w.CustomName
	}
	return privatePaneLabel
}
