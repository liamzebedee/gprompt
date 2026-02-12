package tui

import (
	"context"
	"fmt"
	"os"

	"p2p/cluster"

	"github.com/stukennedy/tooey/app"
	"golang.org/x/term"
)

// RunTUI creates and runs the tooey app for gcluster steer.
func RunTUI(client *cluster.SteerClient) error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to enter raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	a := &app.App{
		Init:   func() interface{} { return NewModel(client) },
		Update: tuiUpdate,
		View:   tuiView,
	}
	return a.Run(context.Background())
}

// --- Subscriptions ---

func stateSub(client *cluster.SteerClient) app.Sub {
	return func(send func(app.Msg)) app.Msg {
		for {
			p, ok := <-client.StateCh
			if !ok {
				return errMsg{err: fmt.Errorf("state channel closed")}
			}
			send(stateMsg(p))
		}
	}
}

func errSub(client *cluster.SteerClient) app.Sub {
	return func(send func(app.Msg)) app.Msg {
		for {
			e, ok := <-client.ErrCh
			if !ok {
				return nil
			}
			send(errMsg{err: e})
		}
	}
}

func reconnectSub(client *cluster.SteerClient) app.Sub {
	return func(send func(app.Msg)) app.Msg {
		for {
			_, ok := <-client.ReconnectCh
			if !ok {
				return nil
			}
			send(reconnectMsg{})
		}
	}
}

func disableMouseCmd() app.Msg {
	os.Stdout.WriteString("\x1b[?1006l\x1b[?1000l")
	return nil
}
