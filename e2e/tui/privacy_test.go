package tuie2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Gaurav-Gosain/tuitest"
)

// Privacy mode (toggle_privacy, tuios privacy) and private panes
// (toggle_pane_private) driven through a real PTY: a pane marked private and
// seen while the mode is on shows a placeholder, and nothing it printed reaches
// another surface or the host terminal.
//
// How this could pass wrongly, written down first:
//   - The secrets could be missing everywhere because they were never shown.
//     The pane prints them through arithmetic, so the typed command never
//     holds them, and the same walk with the mode off has to find each one on
//     at least one surface before the walk with the mode on means anything.
//   - A surface could be read before it is drawn, or after it closed. Each
//     one is opened, waited for by its own marker, read, and closed again.
//   - The mark alone could be hiding the pane. With the mode off the marked
//     pane still shows its text.
//   - A host write could trail the frame. The host stream is read after a
//     pause, from a mark set when the mode came on.
//   - The placeholder could be a blank pane. It has to name the pane.
//
// NEGATIVE CONTROLS (e2e/tui/NEGATIVE_CONTROLS.md, "Privacy mode hides a
// private pane").

// The secrets the private pane prints. Each is built by the shell, so it is on
// the screen only as output.
const (
	privacyBody  = "BODY42LEAK"
	privacyTitle = "TITLE42LEAK"
	privacyCwd   = "CWD42LEAK"
	privacyNote  = "NOTE42LEAK"
	// privacyNoteOn is the notification sent while the mode is on. It differs
	// from privacyNote, which the dock may still be showing from the walk
	// with the mode off.
	privacyNoteOn = "NOTE43LEAK"
)

var privacySecrets = []string{privacyBody, privacyTitle, privacyCwd, privacyNote, privacyNoteOn}

// privacyConfig shows the rail with the sections that list panes, so the walk
// reads them without toggling it.
const privacyConfig = "[appearance.sidebar]\nenabled = true\nwidth = 28\nposition = 'left'\n" +
	"sections = 'terminals,sessions'\n"

// leaked returns the secrets found in text.
func leaked(text string) []string {
	var found []string
	for _, s := range privacySecrets {
		if strings.Contains(text, s) {
			found = append(found, s)
		}
	}
	return found
}

// closeSurface presses esc and waits for marker to leave the screen.
func closeSurface(t *testing.T, term *tuitest.Terminal, marker, what string) {
	t.Helper()
	sendKeys(t, term, tuitest.Esc)
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return !strings.Contains(s.Text(), marker)
	}, uiTimeout); err != nil {
		t.Fatalf("%s never closed\n%s", what, term.Snapshot())
	}
}

// settled reads the screen after a beat, for a surface that fills in after
// its frame first shows.
func settled(term *tuitest.Terminal) string {
	time.Sleep(300 * time.Millisecond)
	return term.Screen().Text()
}

// privacyWalk opens every surface that shows pane-derived text, from the pane
// that is not private, and returns what each one showed. phase names the saved
// frames.
func privacyWalk(t *testing.T, term *tuitest.Terminal, phase string) map[string]string {
	t.Helper()
	toWindowMode(t, term)
	seen := map[string]string{}

	// The layout as it stands: both panes, their borders, the rail and the
	// dock.
	seen["screen"] = settled(term)
	seen["dock"] = dockRow(term.Screen())
	saveFrame(t, term, "privacy-"+phase+"-screen")

	// The command palette, with its window rows.
	sendKeys(t, term, tuitest.Ctrl('p'))
	waitPaletteOpen(t, term, phase)
	seen["palette"] = settled(term)
	saveFrame(t, term, "privacy-"+phase+"-palette")
	closePalette(t, term, phase)

	// The go-to picker, with its pane rows.
	openGotoPicker(t, term)
	seen["goto"] = settled(term)
	saveFrame(t, term, "privacy-"+phase+"-goto")
	closeSurface(t, term, "@ # ! filter", "the go-to picker")

	// The session switcher.
	sendKeys(t, term, tuitest.Ctrl('b'), "S")
	waitScreen(t, term, "the session switcher never showed", "Sessions")
	seen["switcher"] = settled(term)
	saveFrame(t, term, "privacy-"+phase+"-switcher")
	closeSurface(t, term, "Sessions", "the session switcher")

	// The navigator, with a preview of every pane in turn.
	openNavigator(t, term)
	var nav strings.Builder
	for range 4 {
		nav.WriteString(settled(term))
		sendKeys(t, term, tuitest.Down)
	}
	nav.WriteString(settled(term))
	seen["navigator"] = nav.String()
	saveFrame(t, term, "privacy-"+phase+"-navigator")
	closeSurface(t, term, "Press / to search", "the navigator")

	return seen
}

// clickPane clicks a pane and returns to window mode.
func clickPane(t *testing.T, term *tuitest.Terminal, col, row int) {
	t.Helper()
	mouseClick(t, term, col, row, tuitest.MouseLeft, 0)
	toWindowMode(t, term)
}

// TestPrivacyHidesAPrivatePane is the leak walk: a pane named vault prints a
// body, a title, a directory and a notification, is marked private, and every
// surface is read with the mode off and then on. With the mode on, the pane is
// a placeholder that names it, PiP shows the placeholder, copy mode on it is
// refused, and nothing it printed reaches any surface or the host terminal.
// With the mode off again, the pane shows its text.
func TestPrivacyHidesAPrivatePane(t *testing.T) {
	base := t.TempDir()
	writeConfig(t, base, privacyConfig)
	host := &paneHostCopy{}
	term := startIn(t, base, startOpts{cols: 140, rows: 40, args: []string{"new", "work"}, out: host})
	waitBoot(t, term)
	newWindow(t, term)
	newWindow(t, term)
	enableTiling(t, term)

	// The right pane is focused. It becomes vault and prints its secrets.
	renameWindow(t, term, "vault")
	enterTerminalMode(t, term)
	runInShell(t, term,
		`mkdir -p CWD$((6*7))LEAK && cd CWD$((6*7))LEAK && printf '\033]2;TITLE%sLEAK\007' $((6*7)) && echo BODY$((6*7))LEAK`,
		privacyBody, shellTimeout)
	leaveTerminalMode(t, term)

	// Mark it. The mark alone hides nothing.
	if out, err := tuiosCLI(t, base, "run-command", "-s", "work", "toggle_pane_private"); err != nil {
		t.Fatalf("toggle_pane_private: %v\n%s", err, out)
	}
	out, err := tuiosCLI(t, base, "get-window", "vault", "--json", "--session", "work")
	var marked struct {
		Private bool `json:"private"`
	}
	if err != nil || json.Unmarshal([]byte(out), &marked) != nil || !marked.Private {
		t.Fatalf("get-window vault does not say private after the mark: %v\n%s", err, out)
	}
	if !strings.Contains(term.Screen().Text(), privacyBody) {
		t.Fatalf("the mark hid the pane with the mode off\n%s", term.Snapshot())
	}

	// The other pane takes the focus, the vault pane raises a notification,
	// and the walk with the mode off finds every secret somewhere.
	cols, _ := term.Screen().Size()
	leftCol, rightCol := 28+(cols-28)/4, 28+3*(cols-28)/4
	clickPane(t, term, leftCol, 10)
	if out, err := tuiosCLI(t, base, "send-text", "-s", "work", "-w", "vault",
		`printf '\033]9;NOTE%sLEAK\007' $((6*7))`+"\n"); err != nil {
		t.Fatalf("raise the notification: %v\n%s", err, out)
	}
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return strings.Contains(dockRow(s), privacyNote)
	}, uiTimeout); err != nil {
		t.Fatalf("the notification from the vault pane never reached the dock\n%s", term.Snapshot())
	}
	off := privacyWalk(t, term, "off")
	var all strings.Builder
	for _, text := range off {
		all.WriteString(text)
	}
	for _, s := range []string{privacyBody, privacyTitle, privacyCwd, privacyNote} {
		if !strings.Contains(all.String(), s) {
			t.Fatalf("with the mode off no surface showed %s, so the walk cannot show it is hidden", s)
		}
	}
	for name, text := range off {
		t.Logf("mode off, %s shows %v", name, leaked(text))
	}

	// The mode comes on for this client.
	out, err = tuiosCLI(t, base, "privacy", "on", "-s", "work", "--json")
	var mode struct {
		Privacy  bool   `json:"privacy"`
		ClientID string `json:"client_id"`
	}
	if err != nil || json.Unmarshal([]byte(out), &mode) != nil || !mode.Privacy || mode.ClientID == "" {
		t.Fatalf("tuios privacy on: %v\n%s", err, out)
	}
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		text := s.Text()
		return len(leaked(text)) == 0 && strings.Contains(text, "vault")
	}, uiTimeout); err != nil {
		t.Fatalf("with the mode on the vault pane still shows %v\n%s", leaked(term.Screen().Text()), term.Snapshot())
	}
	if out, err := tuiosCLI(t, base, "privacy", "-s", "work"); err != nil || !strings.Contains(out, "is on") {
		t.Fatalf("tuios privacy does not say on: %v\n%s", err, out)
	}
	hostMark := len(host.String())

	// A notification raised while the mode is on reaches neither the dock nor
	// the host.
	if out, err := tuiosCLI(t, base, "send-text", "-s", "work", "-w", "vault",
		`printf '\033]9;NOTE%sLEAK\007' $((6*7+1))`+"\n"); err != nil {
		t.Fatalf("raise the notification: %v\n%s", err, out)
	}
	if out, err := tuiosCLI(t, base, "wait-for", "window-output", "-s", "work", "-w", "vault",
		"--pattern", "NOTE", "--timeout", "10000"); err != nil {
		t.Fatalf("the vault pane never ran the notification: %v\n%s", err, out)
	}
	time.Sleep(time.Second)

	for name, text := range privacyWalk(t, term, "on") {
		if found := leaked(text); len(found) > 0 {
			t.Errorf("with the mode on, %s shows %v", name, found)
		}
	}

	// PiP of the vault pane shows the placeholder.
	clickPane(t, term, rightCol, 10)
	if out, err := tuiosCLI(t, base, "run-command", "-s", "work", "toggle_pip"); err != nil {
		t.Fatalf("toggle_pip: %v\n%s", err, out)
	}
	clickPane(t, term, leftCol, 10)
	box := waitPiP(t, term, "vault", "with the mode on")
	if found := leaked(pipBoxText(term.Screen(), box)); len(found) > 0 {
		t.Errorf("with the mode on, the picture-in-picture view shows %v\n%s", found, term.Snapshot())
	}
	saveFrame(t, term, "privacy-on-pip")
	if out, err := tuiosCLI(t, base, "run-command", "-s", "work", "toggle_pip"); err != nil {
		t.Fatalf("toggle_pip: %v\n%s", err, out)
	}
	waitNoPiP(t, term, "vault", "after the unpin")

	// Copy mode on the vault pane is refused with a notice.
	clickPane(t, term, rightCol, 10)
	sendKeys(t, term, tuitest.Ctrl('b'), "[")
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return strings.Contains(dockRow(s), "Private pane")
	}, uiTimeout); err != nil {
		t.Fatalf("copy mode on the vault pane gave no notice\n%s", term.Snapshot())
	}
	if found := leaked(settled(term)); len(found) > 0 {
		t.Errorf("copy mode on the vault pane shows %v\n%s", found, term.Snapshot())
	}
	saveFrame(t, term, "privacy-on-copy-mode")
	sendKeys(t, term, tuitest.Esc)

	// A beat for any host write that trails the last frame.
	time.Sleep(500 * time.Millisecond)
	if found := leaked(host.String()[hostMark:]); len(found) > 0 {
		t.Errorf("with the mode on, the host terminal was sent %v", found)
	}

	// The mode goes off and the pane shows its text again.
	if out, err := tuiosCLI(t, base, "privacy", "off", "-s", "work"); err != nil {
		t.Fatalf("tuios privacy off: %v\n%s", err, out)
	}
	if err := term.WaitForText(privacyBody, uiTimeout); err != nil {
		t.Fatalf("with the mode off again the vault pane is still hidden\n%s", term.Snapshot())
	}
}
