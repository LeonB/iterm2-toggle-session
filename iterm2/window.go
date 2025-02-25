package iterm2

import (
	"fmt"
	"strconv"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

type Window struct {
	c       *client.Client
	id      string
	session string
}

func (w *Window) CreateTab() (*Tab, error) {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{
				WindowId: str(w.id),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create tab for window %q: %w", w.id, err)
	}
	ctr := resp.GetCreateTabResponse()
	if ctr.GetStatus() != api.CreateTabResponse_OK {
		return nil, fmt.Errorf("unexpected tab status: %s", ctr.GetStatus())
	}
	return &Tab{
		c:        w.c,
		id:       strconv.Itoa(int(ctr.GetTabId())),
		windowID: w.id,
	}, nil
}

func (w *Window) ListTabs() ([]*Tab, error) {
	list := []*Tab{}
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, window := range resp.GetListSessionsResponse().GetWindows() {
		if window.GetWindowId() != w.id {
			continue
		}
		for _, t := range window.GetTabs() {
			list = append(list, &Tab{
				c:        w.c,
				id:       t.GetTabId(),
				windowID: w.id,
			})
		}
	}
	return list, nil
}

func (w *Window) SetTitle(s string) error {
	_, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: str(fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)),
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &w.id,
					},
				},
			},
		},
	})
	return err
}

func (w *Window) Activate() error {
	orderWindowFront := true
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: &api.ActivateRequest{
				Identifier: &api.ActivateRequest_WindowId{
					WindowId: w.id,
				},
				OrderWindowFront: &orderWindowFront,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error activating window %q: %w", w.id, err)
	}
	if status := resp.GetActivateResponse().GetStatus(); status != api.ActivateResponse_OK {
		return fmt.Errorf("unexpected status for activate request: %s", status)
	}
	return nil
}
