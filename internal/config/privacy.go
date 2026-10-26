package config

// PrivacyConfig is the [privacy] section: privacy mode, which draws every pane
// marked private (toggle_pane_private) as a placeholder and keeps its contents
// off every other surface, for screen sharing.
//
// The mode is what this client's screen shows, like the spotlight, so a second
// client attached to the same session keeps its own. The marks are session
// state and are not set here.
type PrivacyConfig struct {
	// Enabled is the mode a client starts in (default: false). toggle_privacy
	// and tuios privacy change it for that client until it detaches; nothing
	// writes it back.
	Enabled bool `toml:"enabled"`
}
