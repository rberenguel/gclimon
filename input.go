package main

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

func handleInput() {
	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		// --- Edit mode: capture label text ---
		stateMu.Lock()
		inEdit := editMode
		stateMu.Unlock()

		if inEdit {
			switch {
			case n == 1 && buf[0] == 0x1b: // Esc — cancel
				stateMu.Lock()
				editMode = false
				editBuffer = ""
				stateMu.Unlock()

			case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // Enter — confirm
				sel := int(atomic.LoadInt32(&selectedIdx))
				stateMu.Lock()
				panes := getSortedPanes()
				if sel < len(panes) {
					p := state[panes[sel].Pane]
					p.Label = editBuffer
					state[panes[sel].Pane] = p
				}
				editMode = false
				editBuffer = ""
				stateMu.Unlock()

			case n == 1 && (buf[0] == 0x7f || buf[0] == 0x08): // Backspace
				stateMu.Lock()
				if len(editBuffer) > 0 {
					runes := []rune(editBuffer)
					editBuffer = string(runes[:len(runes)-1])
				}
				stateMu.Unlock()

			case n == 1 && buf[0] >= 0x20 && buf[0] < 0x7f: // printable ASCII
				stateMu.Lock()
				editBuffer += string(buf[0])
				stateMu.Unlock()
			}
			continue
		}

		// --- Normal mode ---
		input := string(buf[:n])

		if input == "q" || input == "\x03" {
			cleanupAndExit()
		}

		if n == 1 && buf[0] >= '1' && buf[0] <= '9' {
			idx := int(buf[0] - '1')
			atomic.StoreInt32(&selectedIdx, int32(idx))
			jumpToPaneIndex(idx)
			continue
		}

		if n == 1 && buf[0] == 'd' {
			removeSelectedPane()
			continue
		}

		if n == 1 && buf[0] == 'r' {
			sel := int(atomic.LoadInt32(&selectedIdx))
			stateMu.Lock()
			panes := getSortedPanes()
			if sel < len(panes) {
				editMode = true
				editBuffer = panes[sel].Label
			}
			stateMu.Unlock()
			continue
		}

		if n == 1 && (buf[0] == '\r' || buf[0] == '\n') {
			jumpToPaneIndex(int(atomic.LoadInt32(&selectedIdx)))
			continue
		}

		if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up arrow — move up one grid row
				stateMu.Lock()
				cols := int32(curLayout.numCols)
				stateMu.Unlock()
				if cols < 1 {
					cols = 1
				}
				if cur := atomic.LoadInt32(&selectedIdx); cur >= cols {
					atomic.StoreInt32(&selectedIdx, cur-cols)
				}
				continue
			case 'B': // Down arrow — move down one grid row
				stateMu.Lock()
				cols := int32(curLayout.numCols)
				count := int32(len(state))
				stateMu.Unlock()
				if cols < 1 {
					cols = 1
				}
				if cur := atomic.LoadInt32(&selectedIdx); cur+cols < count {
					atomic.StoreInt32(&selectedIdx, cur+cols)
				}
				continue
			case 'C': // Right arrow — next pane
				cur := atomic.LoadInt32(&selectedIdx)
				stateMu.Lock()
				count := int32(len(state))
				stateMu.Unlock()
				if cur < count-1 {
					atomic.StoreInt32(&selectedIdx, cur+1)
				}
				continue
			case 'D': // Left arrow — previous pane
				if cur := atomic.LoadInt32(&selectedIdx); cur > 0 {
					atomic.StoreInt32(&selectedIdx, cur-1)
				}
				continue
			case '<': // SGR mouse event
				handleMouseEvent(buf[:n])
				continue
			}
		}
	}
}

func handleMouseEvent(data []byte) {
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
		if cur := atomic.LoadInt32(&selectedIdx); cur > 0 {
			atomic.StoreInt32(&selectedIdx, cur-1)
		}
		return
	}
	if button == 65 && press {
		cur := atomic.LoadInt32(&selectedIdx)
		stateMu.Lock()
		count := int32(len(state))
		stateMu.Unlock()
		if cur < count-1 {
			atomic.StoreInt32(&selectedIdx, cur+1)
		}
		return
	}

	// Left click: map (cx, cy) to box index using current grid layout.
	if button != 0 || !press {
		return
	}
	if cy < firstBoxRow {
		return
	}

	stateMu.Lock()
	numCols := curLayout.numCols
	boxWidth := curLayout.boxWidth
	count := len(state)
	stateMu.Unlock()

	rowsPerBoxRow := linesPerBox + boxGap
	offset := cy - firstBoxRow
	gridRow := offset / rowsPerBoxRow
	if offset%rowsPerBoxRow >= linesPerBox { // clicked on blank gap row
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

	atomic.StoreInt32(&selectedIdx, int32(boxIdx))
	jumpToPaneIndex(boxIdx)
}
