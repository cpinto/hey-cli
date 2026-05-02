#!/usr/bin/env bash
set -euo pipefail

# Build a self-contained skill bundle for Claude Desktop's cowork sandbox.
#
# Output (default: bin/hey-skill.zip):
#   hey/
#   ├── SKILL.md             (transformed: sh-prefixed dispatcher invocations,
#   │                         Binary location preamble inserted)
#   └── bin/
#       ├── hey              (POSIX dispatcher: stages binary, installs creds,
#       │                     symlinks onto $PATH; resilient to read-only
#       │                     mounts and stripped +x bits)
#       ├── hey-linux-amd64  (cross-compiled, statically linked, stripped)
#       ├── hey-linux-arm64  (cross-compiled, statically linked, stripped)
#       └── .credentials.json (optional: live OAuth tokens from your macOS
#                              keychain in map[origin]Credentials shape)
#
# Usage:
#   scripts/cowork-bundle.sh [--out PATH] [--no-creds] [--origin URL]
#
#   --out PATH      Output zip path (default: bin/hey-skill.zip).
#   --no-creds      Skip credential extraction (zip is shareable / unauthed).
#   --origin URL    Origin URL keying the credentials map and the keychain
#                   account name (default: https://app.hey.com).
#
# Cap on uncompressed size: Claude Desktop rejects skill zips whose
# uncompressed size exceeds 30 MB. Keep the binary stripped (-s -w) and
# avoid bundling additional binaries for platforms cowork doesn't run.

OUT="bin/hey-skill.zip"
INCLUDE_CREDS=1
ORIGIN="https://app.hey.com"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --out)      OUT="$2"; shift 2 ;;
        --no-creds) INCLUDE_CREDS=0; shift ;;
        --origin)   ORIGIN="$2"; shift 2 ;;
        -h|--help)
            sed -n '/^# /,/^$/p' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_SKILL="$REPO_ROOT/skills/hey/SKILL.md"
[[ -f "$SOURCE_SKILL" ]] || { echo "missing $SOURCE_SKILL" >&2; exit 1; }

# Resolve OUT to absolute before we cd elsewhere.
case "$OUT" in
    /*) ABS_OUT="$OUT" ;;
    *)  ABS_OUT="$REPO_ROOT/$OUT" ;;
esac
mkdir -p "$(dirname "$ABS_OUT")"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/hey/bin"

echo "==> cross-compiling Linux binaries..."
(
    cd "$REPO_ROOT"
    GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "$STAGE/hey/bin/hey-linux-amd64" ./cmd/hey
    GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o "$STAGE/hey/bin/hey-linux-arm64" ./cmd/hey
)

echo "==> writing dispatcher..."
cat > "$STAGE/hey/bin/hey" <<'DISPATCHER'
#!/bin/sh
# Invoke as `sh ${CLAUDE_SKILL_DIR}/bin/hey ...`. On first run this:
#   1. Stages the platform binary to a writable cache and chmods +x.
#   2. Installs bundled OAuth credentials at ~/.config/hey-cli/credentials.json.
#   3. Tries to symlink onto $PATH so bare `hey ...` works:
#      a. /usr/local/bin/hey if writable (fully transparent).
#      b. $HOME/.local/bin/hey otherwise — requires $PATH to include it,
#         which most sandboxes' non-interactive shells do NOT by default.
#         A one-time tip is printed when this fallback is used.
# Subsequent runs are no-ops for setup. On hosts with no platform-matching
# bundled binary, falls through to `hey` on $PATH.
set -eu
DIR="$(cd "$(dirname "$0")" && pwd)"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
esac

SRC="$DIR/hey-${OS}-${ARCH}"

if [ -f "$SRC" ]; then
    CACHE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/hey-cli"
    BIN="$CACHE_DIR/hey"
    if [ ! -x "$BIN" ] || [ "$SRC" -nt "$BIN" ]; then
        mkdir -p "$CACHE_DIR"
        cp "$SRC" "$BIN"
        chmod +x "$BIN"
    fi

    CREDS_SRC="$DIR/.credentials.json"
    CREDS_DST="$HOME/.config/hey-cli/credentials.json"
    if [ -f "$CREDS_SRC" ] && [ ! -f "$CREDS_DST" ]; then
        mkdir -p "$HOME/.config/hey-cli"
        cp "$CREDS_SRC" "$CREDS_DST"
        chmod 600 "$CREDS_DST"
    fi

    INSTALLED_AT=""
    TIP_MARKER="$CACHE_DIR/.path-tip-printed"
    for tgt in /usr/local/bin/hey /usr/local/sbin/hey "$HOME/.local/bin/hey"; do
        if [ -L "$tgt" ] && [ "$(readlink "$tgt" 2>/dev/null)" = "$BIN" ]; then
            INSTALLED_AT="$tgt"
            break
        fi
        tgt_dir="$(dirname "$tgt")"
        if [ ! -e "$tgt" ] && { [ -d "$tgt_dir" ] || mkdir -p "$tgt_dir" 2>/dev/null; } && [ -w "$tgt_dir" ]; then
            if ln -s "$BIN" "$tgt" 2>/dev/null; then
                INSTALLED_AT="$tgt"
                break
            fi
        fi
    done

    case "$INSTALLED_AT" in
        "$HOME/.local/bin/hey")
            case ":$PATH:" in
                *":$HOME/.local/bin:"*) ;;
                *)
                    if [ ! -f "$TIP_MARKER" ]; then
                        echo "hey: installed dispatcher at $INSTALLED_AT but \$HOME/.local/bin is not on \$PATH." >&2
                        echo "hey: for bare 'hey ...' calls, run: export PATH=\"\$HOME/.local/bin:\$PATH\"" >&2
                        echo "hey: (this tip prints once; suppress by creating $TIP_MARKER)" >&2
                        : > "$TIP_MARKER" 2>/dev/null || true
                    fi
                    ;;
            esac
            ;;
    esac

    export HEY_NO_KEYRING=1
    exec "$BIN" "$@"
fi

if command -v hey >/dev/null 2>&1; then
    exec hey "$@"
fi
echo "hey: no binary for ${OS}/${ARCH} in $DIR and none on PATH" >&2
exit 127
DISPATCHER
chmod +x "$STAGE/hey/bin/hey"

echo "==> transforming SKILL.md..."
python3 - "$SOURCE_SKILL" "$STAGE/hey/SKILL.md" <<'PY'
import re, sys

src_path, dst_path = sys.argv[1], sys.argv[2]
with open(src_path) as f:
    text = f.read()

# Locate frontmatter end. The source file starts with "---\n", then YAML,
# then "\n---\n", then body.
m = re.search(r"\n---\n", text)
if not m:
    sys.exit("source SKILL.md has no closing frontmatter delimiter")
fm_text = text[: m.start()]
body = text[m.end():]

# Add allowed-tools to frontmatter if not already there.
if "\nallowed-tools:" not in fm_text:
    fm_text += "\nallowed-tools: Bash"

# Cowork preamble (replaces or prepends a "Binary location" section in the
# body). Keep this in sync with the dispatcher behavior above.
preamble = """## Binary location

The `hey` binary is bundled with this skill at `${CLAUDE_SKILL_DIR}/bin/`.
Always invoke it as `sh ${CLAUDE_SKILL_DIR}/bin/hey ...` (the `sh ` prefix
makes it work even when the skill directory is read-only or the +x bit
was stripped during zip extraction).

The first invocation performs three one-time setup steps automatically:

1. Stages the platform-matching binary into `${XDG_CACHE_HOME:-~/.cache}/hey-cli/hey`
   and marks it executable.
2. Installs bundled OAuth credentials at `~/.config/hey-cli/credentials.json`
   (only if absent) so the binary can refresh tokens in-place.
3. Symlinks the cached binary onto `$PATH` (`/usr/local/bin/hey` if writable,
   otherwise `~/.local/bin/hey`) so bare `hey ...` calls from external
   scripts and memory/workspace files work too.

**Bare `hey ...` calls** work after the first invocation **if** the dispatcher
managed to install the symlink at a directory already on `$PATH`. In sandboxes
where `/usr/local/bin` is owned by a different user (Anthropic's cowork
sandbox runs it as `nobody:nogroup`, read-only), the symlink ends up in
`$HOME/.local/bin`, which is **not on the default `bash -c` `$PATH`** in
those environments. The dispatcher prints a one-time tip showing the exact
`export PATH=...` to add when this happens.

Use `sh ${CLAUDE_SKILL_DIR}/bin/hey ...` everywhere if you don't want to
care about any of this — it always works regardless of `$PATH`, +x bits,
or sandbox ownership rules.

"""

# Strip an existing Binary location section if the source already has one.
body = re.sub(
    r"## Binary location\n\n[\s\S]*?(?=^# |^## )",
    "",
    body,
    count=1,
    flags=re.MULTILINE,
)
body = preamble + body

# Rewrite command invocations: `hey <subcommand>` -> `sh ${CLAUDE_SKILL_DIR}/bin/hey <subcommand>`.
SUBCOMMANDS = [
    "boxes", "box", "threads", "reply", "compose", "drafts", "attachments",
    "calendars", "recordings", "event", "todo", "habit", "timetrack",
    "journal", "seen", "unseen", "auth", "config", "doctor", "setup",
    "commands", "skill", "completion", "tui",
]
sub_pat = re.compile(r"\bhey (" + "|".join(SUBCOMMANDS) + r")\b")
body = sub_pat.sub(r"sh ${CLAUDE_SKILL_DIR}/bin/hey \1", body)

# Bare `hey` inside backticks (TUI launch hints, etc.).
body = re.sub(r"`hey`", r"`sh ${CLAUDE_SKILL_DIR}/bin/hey`", body)

with open(dst_path, "w") as f:
    f.write("---\n" + fm_text.lstrip("---\n").lstrip("\n") + "\n---\n" + body)
PY

if [[ $INCLUDE_CREDS -eq 1 ]]; then
    echo "==> bundling credentials from macOS keychain..."
    if creds_blob=$(security find-generic-password -s hey -a "hey::${ORIGIN}" -w 2>/dev/null); then
        printf '%s' "$creds_blob" \
            | sed 's/^go-keyring-base64://' \
            | base64 -d \
            | python3 -c "
import json, sys
inner = json.load(sys.stdin)
print(json.dumps({'$ORIGIN': inner}, indent=2))
" > "$STAGE/hey/bin/.credentials.json"
        chmod 600 "$STAGE/hey/bin/.credentials.json"
        echo "    bundled (origin: $ORIGIN)"
    else
        echo "    no credentials in keychain for $ORIGIN — skipping" >&2
        echo "    (run \`hey auth login\` first, or pass --no-creds to silence)" >&2
    fi
fi

echo "==> zipping..."
rm -f "$ABS_OUT"
( cd "$STAGE" && zip -qr "$ABS_OUT" hey -x "hey/.DS_Store" "hey/**/.DS_Store" )
chmod 600 "$ABS_OUT"

uncompressed=$(du -sh "$STAGE/hey" | cut -f1)
compressed=$(du -h "$ABS_OUT" | cut -f1)
echo
echo "✓ $ABS_OUT  (compressed: $compressed, uncompressed: $uncompressed)"
echo "  Upload via Claude Desktop's Skills panel + button."
