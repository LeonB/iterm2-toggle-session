package iterm2

import (
	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

type FocusChangedNotification struct {
	c *client.Client
	*api.FocusChangedNotification
}
