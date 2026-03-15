package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"gclimon/internal/state"
)

const (
	minBoxWidth = 60 // minimum box width before reducing column count
	colPad      = 1  // spaces between columns
	LinesPerBox = 6  // top border, usr×2, bot×2, bottom border
	BoxGap      = 1  // blank lines between box rows
	FirstBoxRow = 3  // 1-indexed terminal row where boxes begin (header + blank = 2)
)

func SetTerminalMode(raw bool) {
	sttyFlag := "-F"
	if runtime.GOOS == "darwin" {
		sttyFlag = "-f"
	}
	args := []string{sttyFlag, "/dev/tty"}
	if raw {
		args = append(args, "-icanon", "-echo")
	} else {
		args = append(args, "icanon", "echo")
	}
	exec.Command("stty", args...).Run()
}

func GetTermWidth() int {
	sttyFlag := "-F"
	if runtime.GOOS == "darwin" {
		sttyFlag = "-f"
	}
	out, err := exec.Command("stty", sttyFlag, "/dev/tty", "size").Output()
	if err != nil {
		return 200
	}
	var rows, cols int
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d %d", &rows, &cols)
	if cols < 80 {
		cols = 80
	}
	return cols
}

// runeWidth returns the visual column width of s (correct for non-wide Unicode).
func runeWidth(s string) int {
	return len([]rune(s))
}

// padRight pads s with spaces to reach width rune-columns.
func padRight(s string, width int) string {
	if w := runeWidth(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

// splitContent splits s into two display lines each at most width rune-columns wide.
func splitContent(s string, width int) (string, string) {
	runes := []rune(s)
	if len(runes) <= width {
		return s, ""
	}
	line1 := string(runes[:width])
	rest := runes[width:]
	if len(rest) > width {
		return line1, string(rest[:width-1]) + "…"
	}
	return line1, string(rest)
}

// buildBoxLines renders a single pane box as 6 fixed-width strings (no newlines).
func buildBoxLines(p state.PaneState, idx, sel, boxWidth, contentWidth int) [LinesPerBox]string {
	color := "\033[33m" // yellow — waiting
	switch p.Status {
	case "approval":
		color = "\033[31;1m" // bold red   — needs user approval
	case "tool":
		color = "\033[94m" // bright blue — tool executing
	case "active", "running":
		color = "\033[32m" // green       — agent thinking / processing
	}

	selector := " "
	if idx == sel {
		selector = ">"
	}

	var topPrefix string
	if p.Label != "" {
		topPrefix = fmt.Sprintf("┌─%s[%d] %s ", selector, idx+1, p.Label)
	} else {
		topPrefix = fmt.Sprintf("┌─%s[%d] ", selector, idx+1)
	}
	topDashes := boxWidth - runeWidth(topPrefix) - 1
	if topDashes < 0 {
		topDashes = 0
	}

	prompt1, prompt2 := splitContent(p.Prompt, contentWidth)
	agent1, agent2 := splitContent(p.Agent, contentWidth)

	// Each content line: "│ " (2) + label (5) + content (contentWidth) + " │" (2) = boxWidth
	var lines [LinesPerBox]string
	lines[0] = fmt.Sprintf("%s%s%s┐\033[0m", color, topPrefix, strings.Repeat("─", topDashes))
	lines[1] = fmt.Sprintf("│ \033[0m\033[1mUsr:\033[0m %s %s│\033[0m", padRight(prompt1, contentWidth), color)
	lines[2] = fmt.Sprintf("│ \033[0m     %s %s│\033[0m", padRight(prompt2, contentWidth), color)
	lines[3] = fmt.Sprintf("│ \033[0m\033[1mBot:\033[0m %s %s│\033[0m", padRight(agent1, contentWidth), color)
	lines[4] = fmt.Sprintf("│ \033[0m     %s %s│\033[0m", padRight(agent2, contentWidth), color)
	lines[5] = fmt.Sprintf("%s└%s┘\033[0m", color, strings.Repeat("─", boxWidth-2))
	return lines
}

func DrawUI() {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	sel := int(state.Sel.Load())
	inEdit := state.EditMode
	buf := state.EditBuffer

	var sb strings.Builder
	sb.WriteString("\033[2J\033[H")
	if inEdit {
		sb.WriteString(fmt.Sprintf(
			"\033[2mgclimon  \033[0m rename [%d]: %s\033[2m█   esc=cancel  ↵=confirm\033[0m\n\n",
			sel+1, buf,
		))
	} else {
		sb.WriteString("\033[2mgclimon   q=quit  ←→=select  ↑↓=row  1-9=jump  ↵=focus  d=remove  r=rename\033[0m\n\n")
	}

	panes := state.GetSorted()

	if len(panes) == 0 {
		sb.WriteString("Waiting for agent activity...\n")
		fmt.Print(sb.String())
		return
	}

	// Clamp selection to valid range.
	if sel >= len(panes) {
		sel = len(panes) - 1
		state.Sel.Store(int32(sel))
	}

	termWidth := GetTermWidth()
	numCols := (termWidth + colPad) / (minBoxWidth + colPad)
	if numCols < 1 {
		numCols = 1
	}
	boxWidth := (termWidth - (numCols-1)*colPad) / numCols
	contentWidth := boxWidth - 9 // "│ Usr: "(7) + " │"(2)

	// Publish layout for mouse click mapping.
	state.Layout.NumCols = numCols
	state.Layout.BoxWidth = boxWidth

	for rowStart := 0; rowStart < len(panes); rowStart += numCols {
		rowEnd := rowStart + numCols
		if rowEnd > len(panes) {
			rowEnd = len(panes)
		}
		rowPanes := panes[rowStart:rowEnd]

		// Build all box lines for this row.
		rowLines := make([][LinesPerBox]string, len(rowPanes))
		for j, p := range rowPanes {
			rowLines[j] = buildBoxLines(p, rowStart+j, sel, boxWidth, contentWidth)
		}

		// Print each horizontal line of the row.
		for line := 0; line < LinesPerBox; line++ {
			for j, lines := range rowLines {
				if j > 0 {
					sb.WriteString(strings.Repeat(" ", colPad))
				}
				sb.WriteString(lines[line])
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
}
