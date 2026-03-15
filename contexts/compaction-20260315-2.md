# Session Compaction Summary

## User Intent
- Fix incorrect hook configurations for both Claude and Gemini (wrong hooks were tracking all tool calls instead of only approval-needed ones)
- Implement a proper 4-state colour system (green/blue/yellow/red) aligned with solarized dark
- Refactor `main.go` into smaller focused files and tidy up the project structure
- Polish the TUI: cleaner header, per-pane labels editable in the TUI itself

## Contextual Work Summary

### Hook Config Fixes
- **Claude**: Replaced `PreToolUse` (fires for all tools) with `PermissionRequest` (fires only when user approval is needed); kept `PreToolUse` for the new "tool executing" blue state
- **Gemini**: Replaced `BeforeTool` as the approval signal with `Notification` + `ToolPermission` matcher; brought `BeforeTool` back for the "tool executing" blue state
- **Claude reply text**: `stop` case in hook script now reads `transcript_path` JSONL and extracts the last assistant message to populate the Bot: line

### Status & Colour Redesign
- Four statuses: `active` (green), `tool` (blue/bright), `waiting` (yellow), `approval` (bold red)
- `active`: agent is on its turn (prompt received or tool result being processed)
- `tool`: a tool is currently executing (between PreToolUse/BeforeTool and PostToolUse)
- `waiting`: agent finished its turn, idle
- `approval`: blocked waiting for explicit user approval

### Hook Script Updates (`hooks/gclimon-hook`)
- Added `pre_tool` case → sets `tool` status + "Running: X"
- Updated `prompt` → `active`, `after_tool` → `active`, `stop`/`agent_response` → `waiting`
- `before_tool` (approval) uses fallback jq chain for tool name field differences between Claude and Gemini

### Project Refactor
- Split 450-line `main.go` into: `main.go`, `client.go`, `server.go`, `input.go`, `state.go`, `ui.go` — all `package main`
- Moved `gclimon-hook`, `claude.json`, `gemini.json` into `hooks/` directory
- Added `plan.md` documenting both phases

### Click Fix
- `jumpToPaneIndex` was using `target.Window` (name) for `select-window`, causing it to jump to the wrong window when names collide; fixed to use `target.Pane` (pane ID) for both tmux commands

### TUI Polish
- Header reduced from two lines (bold cyan `=== gCLI Mission Control ===` + hints) to one dim line: `gclimon   q=quit  ↑↓/1-9=select  ↵=jump  d=remove  r=rename`
- Box titles no longer show `Window: X | Pane: Y`; show optional label or just the index number
- Per-pane labels: press `r` to enter rename mode inline (edit buffer shown in header, `Enter` confirms, `Esc` cancels); label persists via state merge

## Files Touched

### Core Logic
- **`main.go`**: Trimmed to entry point + `cleanupAndExit` only
- **`client.go`**: `runClient`; added `-l` label flag (kept for completeness, TUI is canonical way)
- **`server.go`**: `runServer`, `handleConnection`; added `Label` to merge logic
- **`state.go`**: `PaneState` (added `Label` field), state vars, `getSortedPanes`, `removeSelectedPane`, `jumpToPaneIndex`; added `editMode`/`editBuffer` globals
- **`input.go`**: `handleInput`, `handleMouseEvent`; added `r` key for rename mode, full edit-mode input handling; corrected `firstBoxRow` from 4 to 3 after header shrink
- **`ui.go`**: `drawUI` with 4-colour switch, new minimal header, label-based box titles, edit mode prompt rendering

### Hook Scripts
- **`hooks/gclimon-hook`**: Added `pre_tool` case; updated status names throughout; `stop` reads transcript for reply text
- **`hooks/claude.json`**: `PermissionRequest` for approval, `PreToolUse` for tool-executing state, all five hooks present
- **`hooks/gemini.json`**: `Notification/ToolPermission` for approval, `BeforeTool` for tool-executing state

### Docs
- **`plan.md`**: Two-phase plan (refactor + colour redesign) — now fully implemented
