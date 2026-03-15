# gclimon — Implementation Plan

## Phase 1: Tidy Up

### 1.1 Folder structure

```
gclimon/
├── main.go          # main(), cleanupAndExit()
├── client.go        # runClient()
├── server.go        # runServer(), handleConnection()
├── input.go         # handleInput(), handleMouseEvent()
├── state.go         # PaneState, state map, getSortedPanes(), removeSelectedPane(), jumpToPaneIndex()
├── ui.go            # drawUI(), padRight(), runeWidth(), setTerminalMode()
├── go.mod
├── plan.md
└── hooks/
    ├── gclimon-hook     # shared hook script (Claude + Gemini)
    ├── claude.json      # example Claude Code settings (hooks section)
    └── gemini.json      # example Gemini CLI settings (hooks section)
```

All Go files remain in `package main` — no need for sub-packages in a tool this size.

### 1.2 main.go splits

| File | Contents |
|------|----------|
| `main.go` | `main()`, `cleanupAndExit()`, package-level consts (`sockPath`) |
| `client.go` | `runClient()` |
| `server.go` | `runServer()`, `handleConnection()` |
| `input.go` | `handleInput()`, `handleMouseEvent()` |
| `state.go` | `PaneState` struct, `state`/`stateMu`/`selectedIdx` vars, `getSortedPanes()`, `removeSelectedPane()`, `jumpToPaneIndex()` |
| `ui.go` | `drawUI()`, `padRight()`, `runeWidth()`, `setTerminalMode()` |

### 1.3 Move hook files

Move `gclimon-hook`, `claude.json`, `gemini.json` → `hooks/` directory.
Update any install/readme instructions accordingly.

---

## Phase 2: Status & Colour Redesign

### 2.1 Status model

| Status | Colour | Meaning |
|--------|--------|---------|
| `active` | Green `\033[32m` | Agent is on its turn (thinking, processing tool results) |
| `tool` | Blue `\033[94m` | A tool is currently executing (between pre- and post-tool hooks) |
| `waiting` | Yellow `\033[33m` | Agent finished its turn, idle, waiting for user input |
| `approval` | Red `\033[31;1m` | Agent is blocked, needs explicit user approval |

Old names (`running`, `normal`) handled as aliases in `drawUI` during any transition period, then removed.

### 2.2 Hook event → status transitions

```
UserPromptSubmit  → active   "prompt text"
PreToolUse        → tool     "Running: ToolName"
PermissionRequest → approval "Wants: ToolName"
PostToolUse       → active   "Ran: ToolName"
Stop              → waiting  <last assistant reply from transcript>

BeforeAgent (Gemini)       → active   "prompt text"
BeforeTool (Gemini)        → tool     "Running: ToolName"
Notification/ToolPermission → approval "Wants: ToolName"
AfterAgent (Gemini)        → waiting  <reply text>
```

### 2.3 gclimon-hook new cases

```
prompt        → active  + prompt text        (unchanged)
pre_tool      → tool    + "Running: X"       (new — PreToolUse / BeforeTool)
before_tool   → approval + "Wants: X"        (PermissionRequest / ToolPermission)
after_tool    → active  + "Ran: X"           (unchanged)
stop          → waiting + last reply text    (reads transcript_path JSONL)
agent_response → waiting + reply text        (Gemini AfterAgent, unchanged)
```

### 2.4 claude.json additions

Add `PreToolUse` hook calling `gclimon-hook pre_tool` (Blue — tool executing).
Keep `PermissionRequest` calling `gclimon-hook before_tool` (Red — approval).

### 2.5 gemini.json additions

Add `BeforeTool` hook calling `gclimon-hook pre_tool` (Blue — tool executing).
Keep `Notification/ToolPermission` calling `gclimon-hook before_tool` (Red — approval).

### 2.6 ui.go colour changes

```go
switch p.Status {
case "approval":
    color = "\033[31;1m"   // bold red   — solarized red
case "tool":
    color = "\033[94m"     // bright blue — solarized blue
case "active", "running":
    color = "\033[32m"     // green       — solarized green
default: // "waiting", "normal"
    color = "\033[33m"     // yellow      — solarized yellow
}
```
