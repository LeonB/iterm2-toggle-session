package iterm2

import (
	"fmt"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

// NewApp establishes a connection
// with iTerm2 and returns an App.
// Name is an optional parameter that
// can be used to register your application
// name with iTerm2 so that it doesn't
// require explicit permissions every
// time you run the plugin.
func NewApp(name string) (*App, error) {
	c, err := client.New(name)
	if err != nil {
		return nil, err
	}

	return &App{c: c}, nil
}

type App struct {
	c *client.Client
}

func (a *App) CreateWindow() (*Window, error) {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create window tab: %w", err)
	}
	ctr := resp.GetCreateTabResponse()
	if ctr.GetStatus() != api.CreateTabResponse_OK {
		return nil, fmt.Errorf("unexpected window tab status: %s", ctr.GetStatus())
	}
	return &Window{
		c:       a.c,
		id:      ctr.GetWindowId(),
		session: ctr.GetSessionId(),
	}, nil
}

func (a *App) ListWindows() ([]*Window, error) {
	list := []*Window{}
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, w := range resp.GetListSessionsResponse().GetWindows() {
		list = append(list, &Window{
			c:  a.c,
			id: w.GetWindowId(),
		})
	}
	return list, nil
}

func (a *App) Focus() ([]FocusChangedNotification, error) {
	list := []FocusChangedNotification{}
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_FocusRequest{},
	})
	if err != nil {
		return nil, fmt.Errorf("could not focus: %w", err)
	}
	for _, w := range resp.GetFocusResponse().GetNotifications() {
		list = append(list, FocusChangedNotification{
			c:                        a.c,
			FocusChangedNotification: w,
		})
	}
	return list, nil
}

func (a *App) Close() error {
	return a.c.Close()
}

func str(s string) *string {
	return &s
}

func (a *App) SelectMenuItem(item string) error {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_MenuItemRequest{
			MenuItemRequest: &api.MenuItemRequest{
				Identifier: &item,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error selecting menu item %q: %w", item, err)
	}
	if resp.GetMenuItemResponse().GetStatus() != api.MenuItemResponse_OK {
		return fmt.Errorf("menu item %q returned unexpected status: %q", item, resp.GetMenuItemResponse().GetStatus().String())
	}
	return nil
}

func (a App) Activate(raiseAllWindows bool, ignoreOtherApps bool) error {
	orderWindowFront := true
	_, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{ActivateRequest: &api.ActivateRequest{
			OrderWindowFront: &orderWindowFront,
			ActivateApp: &api.ActivateRequest_App{
				RaiseAllWindows:   &raiseAllWindows,
				IgnoringOtherApps: &ignoreOtherApps,
			},
		}},
	})
	return err
}
