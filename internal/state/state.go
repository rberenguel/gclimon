package state

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
	Mu sync.Mutex
	M  = make(map[string]PaneState)
	Sel atomic.Int32

	// EditMode and EditBuffer are protected by Mu.
	EditMode   bool
	EditBuffer string

	// Layout is set by DrawUI and read by HandleMouseEvent; protected by Mu.
	Layout struct {
		NumCols  int
		BoxWidth int
	}
)

// GetSorted returns panes sorted by pane ID. Caller must hold Mu if concurrent access is needed.
func GetSorted() []PaneState {
	var panes []PaneState
	for _, p := range M {
		panes = append(panes, p)
	}
	sort.Slice(panes, func(i, j int) bool {
		return panes[i].Pane < panes[j].Pane
	})
	return panes
}

func RemoveSelected() {
	idx := int(Sel.Load())
	Mu.Lock()
	panes := GetSorted()
	if idx < len(panes) {
		delete(M, panes[idx].Pane)
	}
	Mu.Unlock()

	Mu.Lock()
	remaining := len(M)
	Mu.Unlock()
	if idx > 0 && idx >= remaining {
		Sel.Store(int32(remaining - 1))
	}
}

func JumpTo(idx int) {
	Mu.Lock()
	panes := GetSorted()
	Mu.Unlock()

	if idx >= len(panes) {
		return
	}
	target := panes[idx]

	exec.Command("tmux", "select-window", "-t", target.Pane).Run()
	exec.Command("tmux", "select-pane", "-t", target.Pane).Run()
}
