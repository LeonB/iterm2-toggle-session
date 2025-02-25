package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/LeonB/iterm2-toggle-session/iterm2"
)

var (
	pipeFile = path.Join(os.TempDir(), "iterm2-toggle.fifo")
)

func main() {
	ctx := context.Background()
	code, err := run(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(code)
	}
}

func run(ctx context.Context) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	// handle interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("received interrupt")
		cancel()
		log.Println("closed shutdown channel")
	}()

	if fifoExists(pipeFile) {
		if arg != "" {
			err := sendArgToPipe(pipeFile, arg, ctx)
			if err != nil {
				return 2, err
			}
		}
		return 0, nil
	}

	err := createPipe(pipeFile)
	if err != nil {
		return 3, err
	}

	// use O_RDWR so the named pipe doesn't disconnect: keeps it open
	// this is not standard, but it works on mac
	file, err := os.OpenFile(pipeFile, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return 4, fmt.Errorf("Open named pipe file (%s) error: %s", pipeFile, err)
	}

	// when this process stops, close & also remove the named pipe
	defer func() {
		log.Printf("Going to clean up named pipe '%s'", file.Name())
		log.Println("Closing named pipe")
		file.Close()
		log.Println("Removing named pipe")
		os.Remove(file.Name())
		log.Println("done cleaning up named pipe")
	}()

	// create the app
	app, err := createApp()
	if err != nil {
		return 5, err
	}
	defer app.Close()

	if arg != "" {
		err = handleArg(app, arg)
		if err != nil {
			return 6, err
		}
	}

	inputChan, pipeErrChan := readFromPipe(ctx, file)
	argErrChan := make(chan error)
	go func() {
		for arg := range inputChan {
			log.Println("received arg", arg)
			err := handleArg(app, arg)
			if err != nil {
				argErrChan <- err
			}
			log.Println("handleArg done")
		}
	}()

	log.Println("starting select")
	select {
	case <-ctx.Done():
		// wait until context is cancelled
		log.Println("context cancelled")
		return 0, nil
	case err := <-pipeErrChan:
		return 7, err
	case err := <-argErrChan:
		return 7, err
	}

	// unreachable, select runs until one of the channels receives a value
}

func createApp() (*iterm2.App, error) {
	return iterm2.NewApp("iterm2-toggle")
}

func handleArg(app *iterm2.App, arg string) error {
	notifications, err := app.Focus()
	if err != nil {
		return err
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
		return err
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
		return err
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

	windows, err := app.ListWindows()
	if err != nil {
		return err
	}

	// get a list of sessions
	sessions := []*iterm2.Session{}
	for _, w := range windows {
		tabs, err := w.ListTabs()
		if err != nil {
			return err
		}

		for _, t := range tabs {
			ss, err := t.ListSessions()
			if err != nil {
				return err
			}

			for _, s := range ss {
				// get the process title of this session
				vars, err := s.VariablesGet([]string{"processTitle"})
				if err != nil {
					return err
				}

				title := vars["processTitle"]
				if !strings.Contains(title, arg) {
					log.Printf("skipping session '%s', does not match arg '%s'", title, arg)
					continue
				}

				log.Printf("appending session, matches %s", title)
				sessions = append(sessions, s)
			}
		}
	}

	if len(sessions) == 0 {
		log.Println("no matching sessions found")
		return nil
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

	// if the currentIndex is not the last session, pick the next one in the
	// list
	if currentIndex != len(sessions)-1 {
		next = sessions[currentIndex+1]
	}

	log.Println("next", next.GetSessionID())

	// activate the session
	log.Println("activating session", next.GetSessionID())
	err = next.Activate(true, true)
	if err != nil {
		return err
	}

	log.Println("activating app")
	err = app.Activate(false, true)
	if err != nil {
		return err
	}

	return nil
}

func fifoExists(pipeFile string) bool {
	// check if the file exists
	_, err := os.Stat(pipeFile)
	return err == nil
}

func sendArgToPipe(pipeFile string, arg string, ctx context.Context) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()

	errChan := make(chan error)
	go func() {
		f, err := os.OpenFile(pipeFile, os.O_WRONLY, 0000)
		if err != nil {
			errChan <- fmt.Errorf("Cannot open named file '%s': %s", pipeFile, err)
		}
		defer f.Close()

		// send argument to pipe
		_, err = f.WriteString(arg + "\n")
		if err != nil {
			errChan <- err
		}

		log.Printf("sent arg '%s' to named pipe", arg)

		// signal done
		errChan <- nil
	}()

	// block until timeout received, or writing argument is done
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Send argument to pipe (%s) error: %s", pipeFile, ctx.Err())
		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("Send argument to pipe (%s) error: %s", pipeFile, err)
			}
			return nil
		}
	}
}

func createPipe(pipeFile string) error {
	// at this point the named pipe doesn't exist, so create it
	err := syscall.Mkfifo(pipeFile, 0666)
	if err != nil {
		return fmt.Errorf("Make named pipe file (%s) error: %s", pipeFile, err)
	}

	return nil
}

func readFromPipe(_ context.Context, file *os.File) (<-chan string, <-chan error) {
	inputChan := make(chan string)
	errChan := make(chan error)

	go func() {
		// keep listening for messages on the named pipe
		reader := bufio.NewReader(file)
		for {
			log.Println("started reading from pipe")
			line, _, err := reader.ReadLine()
			log.Println("read line", string(line))
			if err != nil {
				log.Println("error reading from pipe", err)
				errChan <- err
			}

			log.Println("sending line to input chan", string(line))
			inputChan <- string(line)
		}
	}()

	log.Println("returning")

	return inputChan, errChan

}
