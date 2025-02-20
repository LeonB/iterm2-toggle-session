package iterm2

import (
	"encoding/json"
	"fmt"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

// SplitPaneOptions for customizing the new pane session.
// More options can be added here as needed
type SplitPaneOptions struct {
	Vertical bool
}

type Session struct {
	c  *client.Client
	id string
}

func (s *Session) SendText(t string) error {
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SendTextRequest{
			SendTextRequest: &api.SendTextRequest{
				Session: &s.id,
				Text:    &t,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending text to session %q: %w", s.id, err)
	}
	if status := resp.GetSendTextResponse().GetStatus(); status != api.SendTextResponse_OK {
		return fmt.Errorf("unexpected status for session %q: %s", s.id, status)
	}
	return nil
}

func (s *Session) Activate(selectTab, orderWindowFront bool) error {
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: &api.ActivateRequest{
				Identifier: &api.ActivateRequest_SessionId{
					SessionId: s.id,
				},
				SelectTab:        &selectTab,
				OrderWindowFront: &orderWindowFront,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error activating session %q: %w", s.id, err)
	}
	if status := resp.GetActivateResponse().GetStatus(); status != api.ActivateResponse_OK {
		return fmt.Errorf("unexpected status for activate request: %s", status)
	}
	return nil
}

func (s *Session) SplitPane(opts SplitPaneOptions) (*Session, error) {
	direction := api.SplitPaneRequest_HORIZONTAL.Enum()
	if opts.Vertical {
		direction = api.SplitPaneRequest_VERTICAL.Enum()
	}
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SplitPaneRequest{
			SplitPaneRequest: &api.SplitPaneRequest{
				Session:        &s.id,
				SplitDirection: direction,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error splitting pane: %w", err)
	}
	spResp := resp.GetSplitPaneResponse()
	if len(spResp.GetSessionId()) < 1 {
		return nil, fmt.Errorf("expected at least one new session in split pane")
	}
	return &Session{
		c:  s.c,
		id: spResp.GetSessionId()[0],
	}, nil
}

func (s *Session) GetSessionID() string {
	return s.id
}

func (s *Session) VariablesGet(vars []string) (map[string]string, error) {
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_SessionId{
					SessionId: s.id,
				},
				Get: vars,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not get variables: %w", err)
	}

	if resp.GetVariableResponse().GetStatus() != api.VariableResponse_OK {
		return nil, fmt.Errorf("unexpected get variable status: %s", resp.GetVariableResponse().GetStatus())
	}

	// json decode the values
	m := map[string]string{}
	values := resp.GetVariableResponse().GetValues()
	for i, v := range values {
		s := ""
		err := json.Unmarshal([]byte(v), &s)
		if err == nil {
			key := vars[i]
			m[key] = s
			continue
		}

		// it probably is an object
		err = json.Unmarshal([]byte(v), &m)
		if err == nil {
			return m, nil
		}

		return nil, fmt.Errorf("could not unmarshal value: %w", err)
	}

	return m, nil
}
