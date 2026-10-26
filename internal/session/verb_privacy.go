package session

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// The privacy verb: turn an attached client's privacy mode on or off, or say
// whether it is on.
//
// The mode is the client's, not the session's: it decides what that client's
// screen shows of the panes marked private, and it is not in session state. So
// the daemon picks the client and hands the request to it, as the pip verb
// does. With no client named, that is the client run-command reaches. The
// marks themselves are session state and are not set here.

func (d *Daemon) verbPrivacy(cs *connState, params json.RawMessage) (any, *verbError) {
	var p struct {
		Session string `json:"session"`
		Client  string `json:"client"`
		State   string `json:"state"`
	}
	if verr := decodeParams(params, &p); verr != nil {
		return nil, verr
	}
	p.Session = strings.TrimSpace(p.Session)
	p.Client = strings.TrimSpace(p.Client)
	switch p.State {
	case "", "on", "off":
	default:
		return nil, invalidParam("state", "state is on or off, or empty to ask")
	}
	if p.Client != "" && p.Session != "" {
		return nil, invalidParam("session", "name a client or a session, not both")
	}

	var tui *connState
	if p.Client != "" {
		target, verr := d.attachedClientByID(p.Client)
		if verr != nil {
			return nil, verr
		}
		tui = target
	} else {
		sess, verr := d.resolveVerbSession(cs, p.Session)
		if verr != nil {
			return nil, verr
		}
		tui = d.findTUIClient(sess.ID)
		if tui == nil {
			return nil, hintedVerbError(ErrVerbNeedsClient,
				"privacy mode belongs to a client, and no client is attached",
				&VerbHint{
					Command: "tuios attach " + sess.Name(),
					Detail:  "Attach a client to the session, then retry.",
				})
		}
	}

	res, err := d.routeToTUISync(tui, uuid.New().String(), &RemoteCommandPayload{
		CommandType: "privacy",
		TapeArgs:    []string{p.State},
	}, routedVerbTimeout)
	if err != nil {
		return nil, &verbError{Code: ErrVerbTimeout, Message: "the attached client did not answer: " + err.Error()}
	}
	if res == nil || !res.Success {
		message := "the attached client refused the request"
		if res != nil && res.Message != "" {
			message = res.Message
		}
		return nil, &verbError{Code: ErrVerbCommandFailed, Message: message}
	}
	out := map[string]any{"type": "privacy", "privacy": false, "client_id": tui.clientID}
	if res.Data != nil {
		if v, ok := res.Data["privacy"]; ok {
			out["privacy"] = v
		}
	}
	return out, nil
}
