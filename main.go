// package main

// import (
// 	"fmt"
// 	"os"

// 	log "github.com/sirupsen/logrus"
// 	iterm2 "github.com/tjamet/goterm2"
// 	"github.com/tjamet/goterm2/api"
// )

// func main() {
// 	logger := log.New()
// 	logger.SetOutput(os.Stdout)
// 	logger.SetLevel(log.TraceLevel)
// 	i, err := iterm2.New()
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	i.Logger(logger)
// 	fmt.Println(i.ListSessions(&api.ListSessionsRequest{}))
// }

package main

import (
	"log"
	"os"

	"marwan.io/iterm2"
)

func main() {
	app, err := iterm2.NewApp("MyCoolPlugin")
	if err != nil {
		log.Fatal(err)
	}

	notifications, err := app.Focus()
	if err != nil {
		log.Fatal(err)
	}

	var (
		currentWindow  *iterm2.Window
		activeTabs     []string
		activeSessions []string
		currentTab     *iterm2.Tab
		currentSession string
	)

	for _, n := range notifications {
		if window := n.GetWindow(); window != nil {
			currentWindow = window
		}

		if tabID := n.GetSelectedTab(); tabID != "" {
			activeTabs = append(activeTabs, tabID)
		}

		if sessionID := n.GetSession(); sessionID != "" {
			activeSessions = append(activeSessions, sessionID)
		}
	}

	// from the current window, get the active tab
	windowTabs, err := currentWindow.ListTabs()
	if err != nil {
		log.Fatal(err)
	}

	for _, wt := range windowTabs {
		for _, at := range activeTabs {
			if wt.GetTabID() == at {
				currentTab = wt
			}
		}
	}

	// from the current tab, get the active session
	tabSessions, err := currentTab.ListSessions()
	if err != nil {
		log.Fatal(err)
	}

	for _, ts := range tabSessions {
		for _, as := range activeSessions {
			if ts.GetSessionID() == as {
				currentSession = ts.GetSessionID()
			}
		}
	}

	// newest events are last
	// loop notifications in reverse
	for i := len(notifications) - 1; i >= 0; i-- {
		n := notifications[i]
		if currentSession == "" {
			if id := n.GetSession(); id != "" {
				currentSession = id
				break
			}
		}
	}

	log.Println(currentSession)
	os.Exit(99)

	windows, err := app.ListWindows()
	if err != nil {
		log.Fatal(err)
	}

	// get a list of sessions
	sessions := []*iterm2.Session{}
	for _, w := range windows {
		tabs, err := w.ListTabs()
		if err != nil {
			log.Fatal(err)
		}

		for _, t := range tabs {
			ss, err := t.ListSessions()
			if err != nil {
				log.Fatal(err)
			}

			for _, s := range ss {
				// // get the process title of this session
				// vars, err := s.VariablesGet([]string{"*"})
				// if err != nil {
				// 	log.Fatal(err)
				// }
				// log.Println(vars)
				// title := vars["processTitle"]
				// log.Println(title)

				log.Println("appending session")
				sessions = append(sessions, s)
			}
		}
	}

	// get index of current session
	currentIndex := -1
	for i, s := range sessions {
		if s.GetSessionID() == currentSession {
			currentIndex = i
			break
		}
	}

	log.Println("current index", currentIndex)

	// if currentIndex is the last session, pick the first one
	next := sessions[0]

	// if the currentIndex is not the last session
	if currentIndex != len(sessions)-1 {
		next = sessions[currentIndex+1]
	}

	log.Println("next", next.GetSessionID())

	// activate the session
	err = next.Activate(true, true)
	if err != nil {
		log.Fatal(err)
	}

	defer app.Close()
	// use app to create or list windows, tabs, and sessions and send various commands to the terminal.
}
