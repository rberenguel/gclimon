# Session Compaction Summary

## User Intent
- Fix multiple bugs and missing features in `gclimon`, a TUI mission control dashboard for monitoring CLI agents (Claude, Gemini) running in tmux panes
- Improve interactivity: working number keys, mouse click support, session removal, keyboard navigation
- Fix the hook configuration for Claude CLI to use the same robust pattern as Gemini

## Contextual Work Summary

### Layout Fix
- Box top-border dashes were 4 characters short because `len()` counted bytes but `┌` and `─` are 3-byte UTF-8 chars with 1 visual column each
- Fixed by switching to rune-count (`runeWidth`) for all visual-width calculations
- Also replaced `%-59s` format (byte-padded) with a custom `padRight()` function that pads by rune count

### Interactivity — Keyboard
- Number keys `1-9` now also update `selectedIdx` before jumping (previously only jumped)
- Arrow keys `↑`/`↓` move the selection through panes
- `Enter` jumps to the currently selected pane
- `d` removes the selected pane from the dashboard
- `jumpToPaneIndex` was refactored to release the mutex *before* running tmux commands (previously held the lock during exec)

### Interactivity — Mouse
- SGR mouse events (`ESC[<Cb;Cx;CyM`) are now fully parsed in `handleMouseEvent`
- Left click maps terminal row to box index using the formula `(row - 4) / 5` (layout: 3 header rows, 5 rows per box)
- Scroll wheel (buttons 64/65) scrolls the selection up/down

### Session Removal
- New `removeSelectedPane()` function deletes the selected pane from the state map
- `selectedIdx` is automatically clamped after removal

### Selection UI
- Added `selectedIdx int32` atomic global to track which pane is selected
- Selected pane shows `>` in its top border (same rune width as the space it replaces, so layout stays intact)
- `drawUI` clamps `selectedIdx` to valid range after state changes

### Claude Hook Configuration
- `claude.json` was rewritten to use the `gclimon-hook` intermediate script (matching the `gemini.json` pattern) instead of fragile inline shell substitutions
- Removed `"matcher": "Bash"` restriction — now all tool types (`Edit`, `Read`, `Write`, `Bash`, etc.) trigger approval/post-tool hooks
- Added `Stop` hook to set status back to `normal` when Claude finishes a turn

### Hook Script
- Added `after_tool` case: sets status to `running` + "Ran: ToolName" — clears the lingering `approval` state after a tool executes
- Added `stop` case: sets status to `normal` with no agent text override — preserves last message, just marks idle

### Module Init
- `go.mod` was missing; created it with `go mod init gclimon` to allow `go build` to work

## Files Touched

### Core Logic
- **`main.go`**: Complete rewrite — layout fix, mouse parsing, keyboard nav, session removal, selection state, mutex fix in `jumpToPaneIndex`, `padRight`/`runeWidth` helpers
- **`go.mod`**: Created (was missing); module name `gclimon`, Go 1.21

### Hook Scripts
- **`gclimon-hook`**: Added `after_tool` and `stop` event cases
- **`claude.json`**: Rewritten to delegate to `gclimon-hook` for all events; removed Bash-only restriction; added `Stop` hook entry

### Unchanged
- **`gemini.json`**: Reference for correct hook pattern (works as-is)
- **`gclimon-hook`** prompt/agent_response/before_tool cases: unchanged
