package main

import (
	"os/exec"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// PaneState holds the current information for a specific tmux pane.
type PaneState struct {
	Window   string    `json:"window"`
	Pane     string    `json:"pane"`
	Status   string    `json:"status"` // "waiting", "active", "tool", "approval"
	Label    string    `json:"label"`  // optional short description shown in the box header
	Prompt   string    `json:"prompt"`
	Agent    string    `json:"agent"`
	LastSeen time.Time
}

var (
	stateMu     sync.Mutex
	state       = make(map[string]PaneState)
	selectedIdx int32 // atomic, index of currently selected pane

	// editMode and editBuffer are protected by stateMu.
	editMode   bool
	editBuffer string

	// curLayout is set by drawUI and read by handleMouseEvent; protected by stateMu.
	curLayout struct {
		numCols  int
		boxWidth int
	}
)

func getSortedPanes() []PaneState {
	var panes []PaneState
	for _, p := range state {
		panes = append(panes, p)
	}
	sort.Slice(panes, func(i, j int) bool {
		return panes[i].Pane < panes[j].Pane
	})
	return panes
}

func removeSelectedPane() {
	idx := int(atomic.LoadInt32(&selectedIdx))
	stateMu.Lock()
	panes := getSortedPanes()
	if idx < len(panes) {
		delete(state, panes[idx].Pane)
	}
	stateMu.Unlock()

	stateMu.Lock()
	remaining := len(state)
	stateMu.Unlock()
	if idx > 0 && idx >= remaining {
		atomic.StoreInt32(&selectedIdx, int32(remaining-1))
	}
}

func jumpToPaneIndex(idx int) {
	stateMu.Lock()
	panes := getSortedPanes()
	stateMu.Unlock()

	if idx >= len(panes) {
		return
	}
	target := panes[idx]

	exec.Command("tmux", "select-window", "-t", target.Pane).Run()
	exec.Command("tmux", "select-pane", "-t", target.Pane).Run()
}
