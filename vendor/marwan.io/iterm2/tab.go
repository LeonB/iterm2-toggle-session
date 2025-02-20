package iterm2

import (
	"fmt"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

type Tab struct {
	c        *client.Client
	id       string
	windowID string
}

func (t *Tab) GetTabID() string {
    return t.id
}

func (t *Tab) SetTitle(s string) error {
	_, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: str(fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)),
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &t.id,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not call set_title: %w", err)
	}
	return nil
}

func (t *Tab) ListSessions() ([]*Session, error) {
	list := []*Session{}
	resp, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error listing sessions for tab %q: %w", t.id, err)
	}
	lsr := resp.GetListSessionsResponse()
	for _, window := range lsr.GetWindows() {
		if window.GetWindowId() != t.windowID {
			continue
		}
		for _, wt := range window.GetTabs() {
			if wt.GetTabId() != t.id {
				continue
			}
			for _, link := range wt.GetRoot().GetLinks() {
				list = append(list, &Session{
					c:  t.c,
					id: link.GetSession().GetUniqueIdentifier(),
				})
			}
		}
	}
	return list, nil
}
