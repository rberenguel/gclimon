package tui

import (
	"os"
	"strconv"
	"strings"

	"gclimon/internal/state"
)

func HandleInput(cleanup func()) {
	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		// --- Edit mode: capture label text ---
		state.Mu.Lock()
		inEdit := state.EditMode
		state.Mu.Unlock()

		if inEdit {
			switch {
			case n == 1 && buf[0] == 0x1b: // Esc — cancel
				state.Mu.Lock()
				state.EditMode = false
				state.EditBuffer = ""
				state.Mu.Unlock()

			case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter — confirm
				sel := int(state.Sel.Load())
				state.Mu.Lock()
				panes := state.GetSorted()
				if sel < len(panes) {
					p := state.M[panes[sel].Pane]
					p.Label = state.EditBuffer
					state.M[panes[sel].Pane] = p
				}
				state.EditMode = false
				state.EditBuffer = ""
				state.Mu.Unlock()

			case n == 1 && (buf[0] == 0x7f || buf[0] == 0x08): // Backspace
				state.Mu.Lock()
				if len(state.EditBuffer) > 0 {
					runes := []rune(state.EditBuffer)
					state.EditBuffer = string(runes[:len(runes)-1])
				}
				state.Mu.Unlock()

			case n == 1 && buf[0] >= 0x20 && buf[0] < 0x7f: // printable ASCII
				state.Mu.Lock()
				state.EditBuffer += string(buf[0])
				state.Mu.Unlock()
			}
			continue
		}

		// --- Normal mode ---
		input := string(buf[:n])

		if input == "q" || input == "\x03" {
			cleanup()
		}

		if n == 1 && buf[0] >= '1' && buf[0] <= '9' {
			idx := int(buf[0] - '1')
			state.Sel.Store(int32(idx))
			state.JumpTo(idx)
			continue
		}

		if n == 1 && buf[0] == 'd' {
			state.RemoveSelected()
			continue
		}

		if n == 1 && buf[0] == 'r' {
			sel := int(state.Sel.Load())
			state.Mu.Lock()
			panes := state.GetSorted()
			if sel < len(panes) {
				state.EditMode = true
				state.EditBuffer = panes[sel].Label
			}
			state.Mu.Unlock()
			continue
		}

		if n == 1 && (buf[0] == '\r' || buf[0] == '\n') {
			state.JumpTo(int(state.Sel.Load()))
			continue
		}

		if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up arrow — move up one grid row
				state.Mu.Lock()
				cols := int32(state.Layout.NumCols)
				state.Mu.Unlock()
				if cols < 1 {
					cols = 1
				}
				if cur := state.Sel.Load(); cur >= cols {
					state.Sel.Store(cur - cols)
				}
				continue
			case 'B': // Down arrow — move down one grid row
				state.Mu.Lock()
				cols := int32(state.Layout.NumCols)
				count := int32(len(state.M))
				state.Mu.Unlock()
				if cols < 1 {
					cols = 1
				}
				if cur := state.Sel.Load(); cur+cols < count {
					state.Sel.Store(cur + cols)
				}
				continue
			case 'C': // Right arrow — next pane
				cur := state.Sel.Load()
				state.Mu.Lock()
				count := int32(len(state.M))
				state.Mu.Unlock()
				if cur < count-1 {
					state.Sel.Store(cur + 1)
				}
				continue
			case 'D': // Left arrow — previous pane
				if cur := state.Sel.Load(); cur > 0 {
					state.Sel.Store(cur - 1)
				}
				continue
			case '<': // SGR mouse event
				HandleMouseEvent(buf[:n])
				continue
			}
		}
	}
}

func HandleMouseEvent(data []byte) {
	s := string(data)
	if !strings.HasPrefix(s, "\x1b[<") {
		return
	}
	s = s[3:]
	if len(s) == 0 {
		return
	}

	press := s[len(s)-1] == 'M'
	s = s[:len(s)-1]

	parts := strings.Split(s, ";")
	if len(parts) != 3 {
		return
	}
	button, err1 := strconv.Atoi(parts[0])
	cx, err2 := strconv.Atoi(parts[1])
	cy, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return
	}

	// Scroll wheel
	if button == 64 && press {
		if cur := state.Sel.Load(); cur > 0 {
			state.Sel.Store(cur - 1)
		}
		return
	}
	if button == 65 && press {
		cur := state.Sel.Load()
		state.Mu.Lock()
		count := int32(len(state.M))
		state.Mu.Unlock()
		if cur < count-1 {
			state.Sel.Store(cur + 1)
		}
		return
	}

	// Left click: map (cx, cy) to box index using current grid layout.
	if button != 0 || !press {
		return
	}
	if cy < FirstBoxRow {
		return
	}

	state.Mu.Lock()
	numCols := state.Layout.NumCols
	boxWidth := state.Layout.BoxWidth
	count := len(state.M)
	state.Mu.Unlock()

	rowsPerBoxRow := LinesPerBox + BoxGap
	offset := cy - FirstBoxRow
	gridRow := offset / rowsPerBoxRow
	if offset%rowsPerBoxRow >= LinesPerBox { // clicked on blank gap row
		return
	}

	gridCol := 0
	if numCols > 1 && boxWidth > 0 {
		gridCol = (cx - 1) / (boxWidth + colPad)
		if gridCol >= numCols {
			gridCol = numCols - 1
		}
	}

	boxIdx := gridRow*numCols + gridCol
	if boxIdx >= count {
		return
	}

	state.Sel.Store(int32(boxIdx))
	state.JumpTo(boxIdx)
}
