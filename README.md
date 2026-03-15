<div align="center">
<img src="gclimon.png" alt="gclimon" width="160">

# gclimon

![Version](https://img.shields.io/badge/version-0.0.1-blue)
![Work in progress](https://img.shields.io/badge/status-work%20in%20progress-red)

_Mission control for your AI agent fleet._

</div>

> `gclimon` — A portmanteau of _gCLI_ for Gemini CLI) and _mon_ (monitor). When you have enough agents running in parallel that you've lost track of what any of them are doing, this is what you open.

A terminal dashboard that watches every Claude Code and gemini-cli session running in your tmux, and shows you — at a glance — who is thinking, who is waiting, who just ran a tool, and who needs your approval.

---

## The problem it solves

Running one AI agent is manageable. Running four or five in parallel — each in its own tmux pane, across multiple windows — is not. You tab between panes, find one idle, tab again, find another mid-tool, lose track of which one asked for permission three minutes ago.

> When you're running several agents at once, the context-switch cost to check on each one adds up fast. You want a single place to see the whole fleet without touching a pane.

`gclimon` is that place. One glance, full picture.

---

## What you see

Each agent gets a box. The box border is coloured by status:

| Colour | Status | Meaning |
| ------ | ------ | ------- |
| Green | `active` | Agent is on its turn — thinking, processing tool results |
| Blue (bright) | `tool` | A tool is currently executing |
| Yellow | `waiting` | Agent finished its turn, idle, waiting for you |
| Bold red | `approval` | Blocked — needs your explicit permission before proceeding |

Inside every box:
```
┌─>[1] my-feature ────────────────────────────────────────────────────────────┐
│ Usr: Implement the search endpoint and write tests                           │
│                                                                              │
│ Bot: Done. Added POST /search with pagination. Tests pass (14/14). Ran      │
│      into a tricky edge case with empty queries — see the comment in        │
└──────────────────────────────────────────────────────────────────────────────┘
```

Boxes tile horizontally to fill your terminal width — typically three per row at a normal window size — and wrap to new rows as agents are added.

---

## How it works

`gclimon` is a client/server pair sharing a Unix socket at `/tmp/gcli_mission_control.sock`.

- **Server** (`gclimon`): renders the TUI, handles keyboard and mouse input, holds all pane state in memory.
- **Client** (`gclimon update ...`): called by the hook script to push a status update from an agent's tmux pane.
- **Hook script** (`gclimon-hook`): a single Bash script wired into both Claude Code and gemini-cli hooks. It reads the hook event payload, extracts the relevant text, and calls `gclimon update`.

```
Claude/Gemini hook fires
        │
        ▼
  gclimon-hook           ← shared Bash script
        │
        ▼
  gclimon update         ← client sends JSON to Unix socket
        │
        ▼
  gclimon (server)       ← merges state, redraws TUI
```

The server never reaches into tmux to poll anything. All state is event-driven, pushed from the agents themselves.

---

## Keyboard and mouse controls

| Key | Action |
| --- | ------ |
| `↑` / `↓` | Move selection up / down |
| `1`–`9` | Jump to and select pane by number |
| `↵ Enter` | Switch tmux focus to the selected pane |
| `d` | Remove the selected pane from the dashboard |
| `r` | Rename the selected pane (inline edit; `Enter` confirms, `Esc` cancels) |
| `q` / `Ctrl-C` | Quit |

Mouse: **left click** selects and jumps to a pane. **Scroll wheel** moves the selection up/down.

To return to `gclimon` after jumping, use `Prefix` + `l` (lowercase L) in tmux — it switches back to the last active window.

---

## Setup

**Requirements:** Go 1.21+, tmux, `jq` in PATH.

### 1. Build and install

```bash
go build -o gclimon .
# Put the binary somewhere in your PATH, e.g.:
cp gclimon ~/.local/bin/gclimon
```

### 2. Install the hook script

```bash
cp hooks/gclimon-hook ~/.local/bin/gclimon-hook
chmod +x ~/.local/bin/gclimon-hook
```

### 3. Wire up hooks

**Claude Code** — merge `hooks/claude.json` into your `~/.claude/settings.json` (or `settings.local.json`):

```json
{
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "gclimon-hook prompt" }] }],
    "PreToolUse":       [{ "hooks": [{ "type": "command", "command": "gclimon-hook pre_tool" }] }],
    "PermissionRequest":[{ "hooks": [{ "type": "command", "command": "gclimon-hook before_tool" }] }],
    "PostToolUse":      [{ "hooks": [{ "type": "command", "command": "gclimon-hook after_tool" }] }],
    "Stop":             [{ "hooks": [{ "type": "command", "command": "gclimon-hook stop" }] }]
  }
}
```

**gemini-cli** — merge `hooks/gemini.json` into your Gemini settings. See the file for the exact structure; it uses `BeforeAgent`, `Notification/ToolPermission`, and `AfterAgent`.

The hook script is safe to have registered even when `gclimon` is not running — if the socket is absent the client exits silently.

### 4. Run

```bash
gclimon
```

Open it in a dedicated tmux pane or a small floating window. Agents in other panes will start populating it as soon as they receive their first prompt.

---

## Architecture

`gclimon` is a single Go binary, no external dependencies beyond the standard library.

- **State** lives in a `map[paneID → PaneState]` protected by a mutex. Updates are merged: an incoming update that leaves a field empty preserves the existing value for that field, so partial updates (status-only, reply-text-only) work without clobbering context.
- **TUI** is drawn by clearing the screen and re-printing every ~200 ms, plus on every incoming update. No TUI framework — just ANSI escape codes.
- **Mouse** uses SGR extended mouse reporting (`\033[?1000h\033[?1006h`) so click coordinates work beyond column 223.
- **Layout** is computed at draw time from the live terminal width (`stty size`), so it adapts automatically when you resize.

---

## Known limitations / future improvements

- **Transcript extraction** — the `stop` hook tries to pull Claude's last reply from `transcript_path`; the format can vary and the extraction may fall back to preserving the previous bot text.
- **PreToolUse for Claude** — the `pre_tool` (blue) state requires a `PreToolUse` hook entry in your Claude settings; it is absent from older configs.
- **No persistence** — state is in-memory only; restarting `gclimon` starts fresh.
- **tmux required** — the jump-to-pane feature uses `tmux select-window` and `tmux select-pane`; the dashboard itself runs fine outside tmux but jumping won't work.

---

## Acknowledgements

- [Claude](https://claude.ai) — primary co-author and first user; ran in its own monitored pane while building this.
- [Gemini](https://gemini.google.com) — second supported agent; icon.
