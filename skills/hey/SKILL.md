---
name: hey
description: |
  Interact with HEY email via the HEY CLI. Read and send emails, manage boxes,
  download attachments, schedule and edit calendar events, track todos, habits,
  time, and journal entries, and inspect/configure the CLI itself. Use for ANY
  HEY-related question or action.
triggers:
  # Direct invocations
  - hey
  - /hey
  # Email actions
  - hey boxes
  - hey box
  - hey threads
  - hey reply
  - hey compose
  - hey drafts
  - hey attachments
  - download attachment
  - download attachments
  - save attachment
  # Calendar actions
  - hey calendars
  - hey recordings
  - hey event
  - create event
  - update event
  - delete event
  - schedule event
  # Todos
  - hey todo
  # Seen/unseen
  - hey seen
  - hey unseen
  - mark as read
  - mark as seen
  - mark as unseen
  - mark as unread
  # Habits
  - hey habit
  # Time tracking
  - hey timetrack
  # Journal
  - hey journal
  # Auth
  - hey auth
  # Config & diagnostics
  - hey config
  - hey doctor
  - hey setup
  # Common actions
  - check my email
  - read email
  - send email
  - reply to email
  - compose email
  - list mailboxes
  - check calendar
  - add todo
  - complete todo
  - track time
  - write journal
  # Questions
  - can I hey
  - how do I hey
  - what's in hey
  - what hey
  - does hey
  # My work
  - my emails
  - my inbox
  - my imbox
  - my todos
  - my calendar
  - my journal
  # URLs
  - hey.com
invocable: true
argument-hint: "[command] [args...]"
---

# /hey - HEY Email Workflow Command

CLI for HEY email: mailboxes, email threads, replies, compose, attachments, calendars, calendar events, todos, habits, time tracking, journal entries, plus auth/config/diagnostics.

## Agent Invariants

**MUST follow these rules:**

1. **Always use `--json`** for structured, predictable output
2. **Authentication required** for all data commands — run `hey auth login` first
3. **HTML output** is available via `--html` for commands that return HTML content

### Global output flags

These work on every command:

| Flag | Purpose |
|------|---------|
| `--json` | JSON envelope `{ok, data, code, summary, ...}` (preferred for parsing) |
| `--quiet` | Raw data with no envelope (for piping into another tool) |
| `--ids-only` | One ID per line (good for `xargs`-style pipelines) |
| `--count` | Print only the count of results |
| `--markdown` | Markdown table (for human-readable digests) |
| `--stats` | Include request stats in JSON `meta` |
| `--agent` | Agent mode: JSON envelope, no TTY formatting (hidden flag, intended for agents) |
| `--base-url <url>` | Override the HEY server URL |
| `-v` / `--verbose` | Log HTTP requests to stderr (repeat for more detail) |

### Pagination

List commands (`box`, `boxes`, `drafts`, `recordings`, `todo list`, `timetrack list`, `journal list`) accept:

- `--limit <n>` — cap results at N (default varies per command)
- `--all` — fetch every page, overriding `--limit`

## Quick Reference

| Task | Command |
|------|---------|
| List mailboxes | `hey boxes --json` |
| List emails in a box | `hey box imbox --json` |
| Read email thread | `hey threads <topic_id> --json` |
| Reply to email | `hey reply <topic_id> -m "Thanks!"` |
| Compose email | `hey compose --to user@example.com --subject "Hello"` |
| Compose with CC/BCC | `hey compose --to alice@example.com --cc bob@example.com --bcc carol@example.org --subject "Hello"` |
| List drafts | `hey drafts --json` |
| List attachments in a thread | `hey attachments <topic_id> --json` |
| Download all attachments in a thread | `hey attachments download <topic_id> --output ~/Downloads` |
| Download a specific attachment | `hey attachments download <topic_id> --entry <entry_id> --index 1 --output ~/Downloads` |
| List calendars | `hey calendars --json` |
| List calendar events | `hey recordings 123 --json` |
| List events in date range | `hey recordings 123 --starts-on 2026-01-01 --ends-on 2026-01-31 --json` |
| Create timed event | `hey event create "Standup" --calendar-id 1 --starts-at 2026-01-20 --start-time 09:00 --end-time 09:30 --timezone America/New_York` |
| Create all-day event | `hey event create "Day off" --calendar-id 1 --starts-at 2026-01-20 --all-day` |
| Update event | `hey event update 123 --title "New title"` |
| Delete event | `hey event delete 123` |
| List todos | `hey todo list --json` |
| Add todo | `hey todo add "Buy milk"` |
| Add todo with date | `hey todo add "Pay invoice" --date 2026-01-31` |
| Complete todo | `hey todo complete 123` |
| Uncomplete todo | `hey todo uncomplete 123` |
| Delete todo | `hey todo delete 123` |
| Mark as seen | `hey seen 12345` |
| Mark as unseen | `hey unseen 12345` |
| Complete habit | `hey habit complete 123` |
| Uncomplete habit | `hey habit uncomplete 123` |
| Start time tracking | `hey timetrack start` |
| Stop time tracking | `hey timetrack stop` |
| Current timer | `hey timetrack current --json` |
| List time entries | `hey timetrack list --json` |
| List journal entries | `hey journal list --json` |
| Read journal entry | `hey journal read 2024-03-15 --json` |
| Write journal entry | `hey journal write "Today was great"` |
| Check auth status | `hey auth status` |
| Refresh access token | `hey auth refresh` |
| Print access token | `hey auth token` |
| Show config | `hey config show --json` |
| Set config value | `hey config set base_url https://app.hey.com` |
| Run health check | `hey doctor --json` |
| Initial setup wizard | `hey setup` |
| List all commands | `hey commands --json` |
| Launch TUI | `hey` (or `hey tui`) |

## Decision Trees

### Reading Email

```
Want to read email?
├── Which mailbox? → hey boxes --json
├── List emails in box? → hey box <name|id> --json
├── Read full thread? → hey threads <topic_id> --json
├── List attachments? → hey attachments <topic_id> --json
├── Download attachments? → hey attachments download <topic_id> --output <dir>
├── Mark as seen? → hey seen <posting-id>
├── Mark as unseen? → hey unseen <posting-id>
└── Launch interactive UI? → hey (no args, launches TUI)
```

### Sending Email

```
Want to send email?
├── Reply to thread? → hey reply <topic_id> -m "message"
│   └── Open editor? → hey reply <topic_id> (omit -m to open $EDITOR)
├── Compose new? → hey compose --to <email> --subject "Subject"
│   ├── With body? → hey compose --to <email> --subject "Subject" -m "Body"
│   ├── With CC? → add --cc <email>
│   └── With BCC? → add --bcc <email>
└── Check drafts? → hey drafts --json
```

### Managing Todos

```
Want to manage todos?
├── List todos? → hey todo list --json
├── Add todo? → hey todo add "Task description"
├── Add todo for date? → hey todo add "Task" --date YYYY-MM-DD
├── Complete? → hey todo complete <id>
├── Uncomplete? → hey todo uncomplete <id>
└── Delete? → hey todo delete <id>
```

### Calendar & Events

```
Want to work with the calendar?
├── List calendars? → hey calendars --json
├── List events/todos/habits in calendar? → hey recordings <calendar-id> --json
│   └── Custom range? → add --starts-on YYYY-MM-DD --ends-on YYYY-MM-DD
├── Create timed event? → hey event create "<title>" --calendar-id <id> --starts-at YYYY-MM-DD --start-time HH:MM --end-time HH:MM --timezone <iana>
├── Create all-day event? → hey event create "<title>" --calendar-id <id> --starts-at YYYY-MM-DD --all-day
├── Update event? → hey event update <id> [flags]
└── Delete event? → hey event delete <id>
```

### Diagnostics & Setup

```
Something not working / first run?
├── First-time setup? → hey setup
├── Check auth status? → hey auth status
├── System health check? → hey doctor --json
├── Show current config? → hey config show --json
├── Change config value? → hey config set <key> <value>
├── List all available commands? → hey commands --json
└── Refresh expired token? → hey auth refresh
```

## Resource Reference

### Email - Boxes

```bash
hey boxes --json                              # List all mailboxes
hey box imbox --json                          # List emails in Imbox (by name)
hey box 123 --json                            # List emails in box (by ID)
hey box imbox --limit 10 --json               # Cap at first 10 postings
hey box imbox --all --json                    # Fetch every page (overrides --limit)
```

Box names: `imbox`, `feedbox`, `trailbox`, `asidebox`, `laterbox`, `bubblebox`

**Response format:** `hey box` returns `{"box": {...}, "postings": [...]}`. Each posting has: `id` (posting ID), `topic_id` (topic ID), `name` (subject), `seen` (read status), `created_at`, `contacts`, `summary`, `app_url`. Use `topic_id` for `hey threads` and `hey reply`.

### Email - Threads

```bash
hey threads <topic_id> --json                 # Read full email thread
hey threads <topic_id> --html                 # Read with raw HTML content
```

**ID note:** `hey box` returns postings with an `id` (posting ID) and a `topic_id` (topic ID). `hey threads` and `hey reply` expect the **topic ID** — use `topic_id` directly. The `app_url` field also contains the topic ID as a fallback (e.g. `https://app.hey.com/topics/123` → `123`).

### Email - Reply & Compose

```bash
hey reply <topic_id> -m "Thanks!"             # Reply with inline message
hey reply <topic_id>                          # Reply via $EDITOR
hey compose --to user@example.com --subject "Hello"         # Compose new (opens $EDITOR)
hey compose --to user@example.com --subject "Hi" -m "Body"  # With inline body
hey compose --to alice@example.com --cc bob@example.com --bcc carol@example.org --subject "Project update" -m "Body"  # With CC/BCC
hey compose --subject "Update" --thread-id 12345 -m "msg"   # Post to existing thread
```

### Email - Attachments

```bash
hey attachments <topic_id> --json                                              # List attachments grouped by entry
hey attachments download <topic_id>                                            # Download all attachments to ./
hey attachments download <topic_id> --output ~/Downloads                       # Download all to a directory
hey attachments download <topic_id> --entry <entry_id> --output ~/Downloads    # Only attachments from one entry
hey attachments download <topic_id> --entry <entry_id> --index 1 --output ~/Downloads  # One specific attachment
```

**ID note:** `hey attachments` takes the **topic ID** (same as `hey threads`). Each entry's attachments come back with a 1-based `index` starting at 1 within that entry. `--index` requires `--entry` because indices reset per entry.

**Response format:** `hey attachments` returns an array of `{entry_id, from, created_at, attachments: [{url, filename, content_type}]}`. Entries with no attachments are omitted. `hey attachments download` returns an array of `{path, filename, entry_id, bytes}` for each saved file. Output directory is created if missing; existing files get a `-1`, `-2`, ... suffix rather than being overwritten.

**What's downloadable:** Real MIME attachments (PDFs, docs, archives) surfaced via HEY's "files" panel, plus Trix inline figures embedded in the email body (typically images). `<action-text-attachment>` markers without a download URL are skipped.

### Email - Seen/Unseen

```bash
hey seen 12345                                # Mark posting as seen
hey seen 12345 67890                          # Mark multiple postings as seen
hey unseen 12345                              # Mark posting as unseen
hey unseen 12345 67890                        # Mark multiple postings as unseen
```

Takes posting IDs (the `id` field from `hey box` output).

### Drafts

```bash
hey drafts --json                             # List drafts
hey drafts --limit 5 --json                   # Cap at 5
hey drafts --all --json                       # Fetch every page
```

### Calendars

```bash
hey calendars --json                          # List calendars (returns array of {id, name, kind})
hey recordings 123 --json                     # List events in calendar (defaults: today → +30 days)
hey recordings 123 --starts-on 2026-01-01 --ends-on 2026-01-31 --json   # Custom date range
hey recordings 123 --limit 50 --json          # Cap per type at 50
hey recordings 123 --all --json               # Fetch every page
```

**Response format:** `hey recordings` returns recordings grouped by type (e.g. `{"Calendar::Event": [...], "Calendar::Habit": [...], "Calendar::Todo": [...]}`). Each recording has: `id`, `title`, `starts_at`, `ends_at`, `all_day`, `recurring`, `starts_at_time_zone`. Access by type key in jq, e.g. `.["Calendar::Event"]`.

### Calendar - Events

```bash
hey event create "Team standup" --calendar-id 1 --starts-at 2026-01-20 --start-time 09:00 --end-time 09:30 --timezone America/New_York
hey event create "Day off" --calendar-id 1 --starts-at 2026-01-20 --all-day
hey event create "Lunch" --calendar-id 1 --starts-at 2026-01-20 --start-time 12:00 --end-time 13:00 --timezone America/New_York --reminder 15m --reminder 5m
hey event update 123 --title "New title"
hey event update 123 --starts-at 2026-01-21 --start-time 10:00 --end-time 11:00
hey event delete 123
```

**Required flags for `event create`:** `--calendar-id`, `--starts-at` (YYYY-MM-DD), and either `--all-day` or the trio `--start-time` / `--end-time` / `--timezone` (HH:MM, IANA tz). Title can be passed positionally or via `-t/--title`. `--reminder` accepts Go durations (`15m`, `1h`, `1d`) and may be repeated.

**Required for `event update`:** the event ID (positional). Any field flag is optional; pass only what's changing. Use `--all-day=false` to convert an all-day event back to timed.

Use `hey calendars --json` to find calendar IDs, and `hey recordings <calendar-id>` to find existing event IDs.

### Todos

```bash
hey todo list --json                          # List todos
hey todo list --limit 20 --json               # Cap at 20
hey todo list --all --json                    # Fetch every page
hey todo add "Task description"               # Add a todo (positional title)
hey todo add --title "Task" --date 2026-01-31 # Add with explicit flags + due date
hey todo complete 123                         # Mark complete
hey todo uncomplete 123                       # Mark incomplete
hey todo delete 123                           # Delete a todo
```

### Habits

```bash
hey habit complete 123                        # Mark habit complete for today
hey habit complete 123 --date 2024-01-15      # Mark complete for specific date
hey habit uncomplete 123                      # Unmark habit for today
hey habit uncomplete 123 --date 2024-01-15    # Unmark for specific date
```

Habit IDs can be found via `hey recordings <calendar-id> --json`.

### Time Tracking

```bash
hey timetrack start                           # Start timer
hey timetrack stop                            # Stop timer
hey timetrack current --json                  # Show current timer
hey timetrack list --json                     # List time entries
hey timetrack list --limit 50 --json          # Cap at 50
hey timetrack list --all --json               # Fetch every page
```

### Journal

```bash
hey journal list --json                       # List journal entries
hey journal list --limit 30 --json            # Cap at 30
hey journal list --all --json                 # Fetch every page
hey journal read 2024-03-15 --json            # Read entry by date
hey journal write "Today's entry"             # Write entry inline (positional)
hey journal write --content "Today's entry"   # Write entry via flag
hey journal write                             # Write entry via $EDITOR
hey journal write 2024-03-15 "Backdated"      # Write entry for a specific date
```

### Authentication

```bash
hey auth login                                # Log in (browser-based OAuth, primary)
hey auth login --no-browser                   # Print auth URL instead of opening browser
hey auth login --token <bearer-token>         # Log in with a pre-generated bearer token
hey auth login --cookie <session_token>       # Log in with a HEY session cookie
hey auth status                               # Check if authenticated
hey auth status --json                        # Machine-readable auth state
hey auth refresh                              # Force refresh of the access token
hey auth token                                # Print the current (live, refreshed) access token
hey auth token --stored                       # Print the stored token without refreshing
hey auth logout                               # Clear stored credentials
```

If a command fails with an auth error, run `hey auth status` to check, then `hey auth login` to re-authenticate. Tokens auto-refresh on expiry; `hey auth refresh` forces it explicitly.

### Configuration

```bash
hey config show --json                        # Show resolved config + sources (env, flags, file)
hey config set base_url https://app.hey.com   # Persist a config value to ~/.config/hey-cli/config.json
```

Config is also overridable per-invocation via `--base-url` (transient) or the `HEY_BASE_URL` env var.

### Diagnostics

```bash
hey doctor --json                             # Full system health report (auth, config, network, version)
hey setup                                     # Interactive first-time setup wizard
hey commands --json                           # Machine-readable list of every command + flag
```

Run `hey doctor` first when something behaves unexpectedly — it surfaces auth state, config resolution, and reachability.

### Skill (this file)

```bash
hey skill                                     # Print this SKILL.md to stdout
hey skill install                             # Copy SKILL.md to ~/.agents/skills/hey/ and symlink to ~/.claude/skills/hey
```
