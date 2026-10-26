package app

import "github.com/Gaurav-Gosain/tuios/internal/terminal"

// Privacy mode: while it is on, every pane marked private (Window.Private) is
// drawn as a placeholder, and nothing it printed reaches another surface. The
// marks are session state; the mode is this client's own (OS.privacy) and is
// never saved. See docs/SESSIONS.md#privacy-mode.

// privatePaneLabel is what the placeholder shows for a pane with no name of
// its own. The title a program sets is pane output, so it is never used.
const privatePaneLabel = "Private pane"

// PrivacyOn reports whether this client is in privacy mode.
func (m *OS) PrivacyOn() bool { return m.privacy }

// hidesPane reports whether w is to be drawn as a placeholder and kept off
// every other surface: the mode is on and the pane is marked private.
func (m *OS) hidesPane(w *terminal.Window) bool {
	return m.privacy && w != nil && w.Private
}

// privateLabel is the name a hidden pane shows: the name the user gave it, or
// privatePaneLabel.
func privateLabel(w *terminal.Window) string {
	if w != nil && w.CustomName != "" {
		return w.CustomName
	}
	return privatePaneLabel
}
