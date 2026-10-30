package app

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Gaurav-Gosain/tuios/internal/overlay"
	"github.com/Gaurav-Gosain/tuios/internal/terminal"
)

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

// privacyChipLabel is the dock chip's name for the mode while it is on.
const privacyChipLabel = "Private"

// privacyChip is the dock chip drawn while the mode is on, with a column in
// front of it to keep it off its neighbour, or "" while the mode is off. It
// takes no click: turning the mode off from a stray click would show what
// the mode hides.
func (m *OS) privacyChip() string {
	if !m.privacy {
		return ""
	}
	pal := m.groundUI()
	ground := pal.AccentTint
	label := lipgloss.NewStyle().Background(ground).
		Foreground(overlay.Readable(pal.Accent, ground)).Bold(true).
		Render(" " + privacyChipLabel + " ")
	return " " + label
}

// PrivacyOn reports whether this client is in privacy mode.
func (m *OS) PrivacyOn() bool { return m.privacy }

// hidesPane reports whether w is to be drawn as a placeholder and kept off
// every other surface: the mode is on and the pane is marked private.
func (m *OS) hidesPane(w *terminal.Window) bool {
	return m.privacy && w != nil && w.Private
}

// publishPrivacy follows a change of the mode or of a mark: every pane's
// hidden flag is set again for the goroutines that hand output to the host,
// and every cached frame is dropped, since it may hold a private pane's cells
// or its placeholder.
func (m *OS) publishPrivacy() {
	for _, w := range m.Windows {
		if w != nil {
			w.SetPrivacyHidden(m.hidesPane(w))
		}
	}
	// The views that keep a copy of a pane's cells read it again.
	m.pip.dirty, m.pip.boxKey = true, ""
	m.Inbox.ck.live, m.Inbox.ck.liveDirty = "", true
	m.clearLinkHover()
	m.MarkAllDirty()
}

// TogglePrivacy is toggle_privacy: it turns this client's privacy mode on or
// off.
func (m *OS) TogglePrivacy() { m.SetPrivacy(!m.privacy) }

// SetPrivacy turns this client's privacy mode on or off and says so. Nothing
// is sent to the daemon: the mode is this client's own.
func (m *OS) SetPrivacy(on bool) {
	if m.privacy != on {
		m.privacy = on
		m.publishPrivacy()
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
	m.publishPrivacy()
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

// privatePlaceholder is what a hidden pane's body shows: its content area,
// blank, with the label in the middle and, under a name the user gave it, the
// words "Private pane". It is built from the window's size and name alone, so
// nothing the pane printed can reach it.
func privatePlaceholder(window *terminal.Window) string {
	return privatePlaceholderSized(privateLabel(window), window.ContentWidth(), window.ContentHeight())
}

// privatePlaceholderSized is the placeholder for a pane named label, cols
// by rows: the PiP and the navigator preview draw it at their own size.
func privatePlaceholderSized(label string, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	lines := []string{label}
	if lines[0] != privatePaneLabel {
		lines = append(lines, privatePaneLabel)
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	top := (rows - len(lines)) / 2
	blank := strings.Repeat(" ", cols)
	var b strings.Builder
	for y := range rows {
		if y > 0 {
			b.WriteByte('\n')
		}
		i := y - top
		if i < 0 || i >= len(lines) {
			b.WriteString(blank)
			continue
		}
		text := lines[i]
		if ansi.StringWidth(text) > cols {
			text = ansi.Truncate(text, cols, "")
		}
		w := ansi.StringWidth(text)
		left := (cols - w) / 2
		b.WriteString(strings.Repeat(" ", left))
		b.WriteString(text)
		b.WriteString(strings.Repeat(" ", cols-left-w))
	}
	return b.String()
}

// shownWindowName is getWindowDisplayName for a surface that shows it: the
// private label for a hidden pane. getWindowDisplayName itself stays real,
// since tape scripts find windows by it.
func (m *OS) shownWindowName(w *terminal.Window) string {
	if m.hidesPane(w) {
		return privateLabel(w)
	}
	return m.getWindowDisplayName(w)
}

// privateLabel is the name a hidden pane shows: the name the user gave it, or
// privatePaneLabel.
func privateLabel(w *terminal.Window) string {
	if w != nil && w.CustomName != "" {
		return w.CustomName
	}
	return privatePaneLabel
}
