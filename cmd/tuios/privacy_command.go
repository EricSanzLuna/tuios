package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// newPrivacyCommand is tuios privacy: the CLI half of the privacy verb.
func newPrivacyCommand() *cobra.Command {
	var session, clientID string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "privacy [on|off]",
		Short: "Turn privacy mode on or off for the attached client",
		Long: `Turn privacy mode on or off for the attached client, or say whether it is on.

While privacy mode is on, the client draws every pane marked private as a
placeholder, and nothing such a pane prints reaches another surface: the rail,
the dock, previews, notifications or the host terminal title. Mark a pane
with the toggle_pane_private action.

The mode belongs to the client, not to the session. It needs an attached
client, and a detach forgets it. With several clients attached, the command
reaches the client run-command reaches, or the one --client names.

A program in a pane cannot run this command.`,
		Example: `  # Before a screen share
  tuios privacy on

  # After it
  tuios privacy off

  # Is it on?
  tuios privacy`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"on", "off"},
		RunE: func(_ *cobra.Command, args []string) error {
			state := ""
			if len(args) > 0 {
				state = args[0]
				if state != "on" && state != "off" {
					return fmt.Errorf("privacy takes on or off, not %q", state)
				}
			}
			if clientID != "" && session != "" {
				return fmt.Errorf("name a client or a session, not both")
			}
			return runPrivacy(session, clientID, state, jsonOutput)
		},
	}
	cmd.Flags().StringVarP(&session, "session", "s", "", "Target session (default: this pane's session, else the most recently active)")
	cmd.Flags().StringVar(&clientID, "client", "", "Id of the attached client to reach (see tuios list-clients)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output result as JSON")
	_ = cmd.RegisterFlagCompletionFunc("session", completeSessionNames)
	return cmd
}

// runPrivacy calls the privacy verb and says whether the mode is on.
func runPrivacy(sessionName, clientID, state string, jsonOutput bool) error {
	client, err := dialVerb()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	params := map[string]any{}
	if sessionName != "" {
		params["session"] = sessionName
	}
	if clientID != "" {
		params["client"] = clientID
	}
	if state != "" {
		params["state"] = state
	}
	raw, err := client.Call("privacy", params)
	if err != nil {
		return reportVerbError(explainVerbError("privacy", err), jsonOutput)
	}
	if jsonOutput {
		return printVerbResult(raw, jsonOutput)
	}
	var res struct {
		Privacy bool `json:"privacy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if res.Privacy {
		fmt.Println("Privacy mode is on.")
	} else {
		fmt.Println("Privacy mode is off.")
	}
	return nil
}
