package iterm2

import (
	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

type FocusChangedNotification struct {
	c *client.Client
	*api.FocusChangedNotification
}

func (n *FocusChangedNotification) GetWindow() *Window {
	w := n.FocusChangedNotification.GetWindow()
	if w == nil {
		return nil
	}
    return &Window{c: n.c, id: *n.FocusChangedNotification.GetWindow().WindowId}
}
