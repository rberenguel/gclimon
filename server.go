package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func runServer() {
	os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen on socket: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()
	defer os.Remove(sockPath)

	setTerminalMode(true)
	defer setTerminalMode(false)

	// Enable ANSI mouse tracking (SGR mode) and hide cursor.
	fmt.Print("\033[?1000h\033[?1006h\033[?25l")
	defer fmt.Print("\033[?1000l\033[?1006l\033[?25h")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cleanupAndExit()
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go handleConnection(conn)
		}
	}()

	go handleInput()

	for {
		drawUI()
		time.Sleep(200 * time.Millisecond)
	}
}

// sanitizeText collapses newlines, tabs, and other control characters into
// single spaces so they never break the fixed-height box layout.
func sanitizeText(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var update PaneState
		if err := json.Unmarshal(scanner.Bytes(), &update); err == nil {
			stateMu.Lock()
			update.LastSeen = time.Now()
			update.Prompt = sanitizeText(update.Prompt)
			update.Agent = sanitizeText(update.Agent)

			// Merge: preserve existing fields if not provided in update.
			if existing, exists := state[update.Pane]; exists {
				if update.Label == "" {
					update.Label = existing.Label
				}
				if update.Prompt == "" {
					update.Prompt = existing.Prompt
				}
				if strings.TrimSpace(update.Agent) == "" {
					update.Agent = existing.Agent
				}
			}

			state[update.Pane] = update
			stateMu.Unlock()
			drawUI()
		}
	}
}
