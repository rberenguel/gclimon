package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"gclimon/internal/state"
)

// runClient sends a state update to the running TUI server over the Unix socket.
// Invoked as: gclimon update -w <window> -p <pane> -s <status> [--prompt <text>] [-a <text>]
func runClient() {
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	window := updateCmd.String("w", "", "Tmux window name/id")
	pane := updateCmd.String("p", "", "Tmux pane id")
	status := updateCmd.String("s", "waiting", "Status (waiting, active, tool, approval)")
	label := updateCmd.String("l", "", "Short description of what this session is about")
	prompt := updateCmd.String("prompt", "", "Last user prompt")
	agent := updateCmd.String("a", "", "Last agent message/action")

	updateCmd.Parse(os.Args[2:])

	if *window == "" || *pane == "" {
		fmt.Fprintln(os.Stderr, "Window and pane are required.")
		os.Exit(1)
	}

	payload := state.PaneState{
		Window: *window,
		Pane:   *pane,
		Status: *status,
		Label:  *label,
		Prompt: *prompt,
		Agent:  *agent,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		// TUI isn't running — exit silently so we don't break the hook.
		os.Exit(0)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\n", string(data))
}
