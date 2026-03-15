package main

import (
	"fmt"
	"os"
)

const sockPath = "/tmp/gcli_mission_control.sock"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		runClient()
		return
	}
	runServer()
}

func cleanupAndExit() {
	setTerminalMode(false)
	fmt.Print("\033[?1000l\033[?1006l\033[?25h")
	os.Remove(sockPath)
	os.Exit(0)
}
