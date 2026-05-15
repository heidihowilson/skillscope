#!/usr/bin/env sh
# skillscope installer
#
# Usage:
#   curl -sSL https://heidihowilson.github.io/skillscope/install.sh | sh
#
# Environment overrides:
#   SKILLSCOPE_VERSION=v0.1.0   pin to a specific release (default: latest)
#   SKILLSCOPE_INSTALL_DIR=...  override install dir (default: /usr/local/bin if
#                                root, else $HOME/.local/bin)
#   SKILLSCOPE_ACCEPT_ROOT=1    allow running as root (refused by default)
#
# Safety:
#   - Verifies sha256 against checksums.txt from the GitHub release.
#   - Refuses to follow HTTP errors silently (curl -fSL).
#   - Cleans up the temp dir on any exit.
#   - Refuses to install as root unless explicitly allowed.

set -eu

# --- config -----------------------------------------------------------------

REPO_OWNER="heidihowilson"
REPO_NAME="skillscope"
BIN_NAME="skillscope"

VERSION="${SKILLSCOPE_VERSION:-}"
ACCEPT_ROOT="${SKILLSCOPE_ACCEPT_ROOT:-0}"

# --- helpers ----------------------------------------------------------------

err() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

info() {
    printf '%s\n' "$*"
}

have() {
    command -v "$1" >/dev/null 2>&1
}

# --- preconditions ----------------------------------------------------------

have curl || err "curl is required but not found on PATH"
have tar  || err "tar is required but not found on PATH"
have uname || err "uname is required but not found on PATH"

# Pick a sha256 tool — sha256sum on Linux, shasum on macOS.
SHA256_CMD=""
if have sha256sum; then
    SHA256_CMD="sha256sum"
elif have shasum; then
    SHA256_CMD="shasum -a 256"
else
    err "need sha256sum or shasum on PATH"
fi

# --- root refusal -----------------------------------------------------------

if [ "$(id -u 2>/dev/null || echo 0)" = "0" ] && [ "$ACCEPT_ROOT" != "1" ]; then
    err "running as root is refused by default; set SKILLSCOPE_ACCEPT_ROOT=1 to override"
fi

# --- OS / arch detection ----------------------------------------------------

OS_RAW=$(uname -s)
case "$OS_RAW" in
    Linux*)  OS="linux"  ;;
    Darwin*) OS="darwin" ;;
    *) err "unsupported OS: $OS_RAW (only linux and darwin are supported)" ;;
esac

ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64|amd64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) err "unsupported architecture: $ARCH_RAW (need x86_64 or arm64)" ;;
esac

# --- version resolution -----------------------------------------------------

if [ -z "$VERSION" ]; then
    info "==> resolving latest release tag…"
    # Use the GitHub API. Falls back gracefully without jq.
    LATEST_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    if have jq; then
        VERSION=$(curl -fsSL "$LATEST_URL" | jq -r '.tag_name')
    else
        VERSION=$(curl -fsSL "$LATEST_URL" \
            | grep -m1 '"tag_name":' \
            | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')
    fi
    [ -n "$VERSION" ] || err "could not determine latest release tag"
fi

# Strip leading 'v' for archive filename (goreleaser drops it).
VERSION_NO_V="${VERSION#v}"

ARCHIVE="${BIN_NAME}_${VERSION_NO_V}_${OS}_${ARCH}.tar.gz"
ARCHIVE_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/checksums.txt"

# --- install dir resolution -------------------------------------------------
#
# Strategy:
#   1. If SKILLSCOPE_INSTALL_DIR is set, honor it.
#   2. If running as root, install to /usr/local/bin.
#   3. Otherwise, prefer the first PATH directory we can write to. This
#      matters especially on macOS, where $HOME/.local/bin is NOT on the
#      default PATH (Linux distros typically have it, macOS doesn't), so
#      the previous fallback installed the binary somewhere `skillscope`
#      wouldn't be found.
#   4. Final fallback: $HOME/.local/bin, with a loud PATH warning later.

is_on_path() {
    case ":${PATH}:" in
        *":$1:"*) return 0 ;;
        *)        return 1 ;;
    esac
}

# Returns true if we can already write to $1, OR if its parent exists and
# is writable so we can mkdir $1.
is_writable() {
    if [ -d "$1" ] && [ -w "$1" ]; then return 0; fi
    parent=$(dirname "$1")
    [ -d "$parent" ] && [ -w "$parent" ]
}

if [ -n "${SKILLSCOPE_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$SKILLSCOPE_INSTALL_DIR"
elif [ "$(id -u 2>/dev/null || echo 0)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR=""
    # Candidates, in priority order. Picked because they're conventional
    # user-bin dirs on macOS / Linux that are often (but not always) on
    # PATH.
    for candidate in \
        "$HOME/.local/bin" \
        "$HOME/bin" \
        "/opt/homebrew/bin" \
        "/usr/local/bin"; do
        if is_on_path "$candidate" && is_writable "$candidate"; then
            INSTALL_DIR="$candidate"
            break
        fi
    done
    # Nothing on PATH was writable — fall back to ~/.local/bin and warn
    # loudly later about the PATH not containing it.
    if [ -z "$INSTALL_DIR" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    fi
fi

# --- staging dir + cleanup --------------------------------------------------

TMPDIR=$(mktemp -d 2>/dev/null || mktemp -d -t skillscope) || err "mktemp failed"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT INT TERM HUP

# --- download + verify ------------------------------------------------------

info "==> downloading ${BIN_NAME} ${VERSION} for ${OS}/${ARCH}"
info "    archive:   $ARCHIVE_URL"
info "    checksums: $CHECKSUMS_URL"

curl -fsSL --output "$TMPDIR/$ARCHIVE"      "$ARCHIVE_URL"   || err "failed to download archive"
curl -fsSL --output "$TMPDIR/checksums.txt" "$CHECKSUMS_URL" || err "failed to download checksums"

info "==> verifying sha256…"
EXPECTED=$(grep "  ${ARCHIVE}$" "$TMPDIR/checksums.txt" | awk '{print $1}')
[ -n "$EXPECTED" ] || err "no checksum entry for $ARCHIVE in checksums.txt"

ACTUAL=$(cd "$TMPDIR" && $SHA256_CMD "$ARCHIVE" | awk '{print $1}')
[ -n "$ACTUAL" ] || err "could not compute sha256 of $ARCHIVE"

if [ "$EXPECTED" != "$ACTUAL" ]; then
    err "checksum mismatch
    expected: $EXPECTED
    actual:   $ACTUAL"
fi
info "    ok"

# --- extract ---------------------------------------------------------------

info "==> extracting…"
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR" || err "tar extraction failed"
[ -f "$TMPDIR/$BIN_NAME" ] || err "expected binary $BIN_NAME not found in archive"

# --- install --------------------------------------------------------------

mkdir -p "$INSTALL_DIR" || err "could not create install dir: $INSTALL_DIR"
DEST="$INSTALL_DIR/$BIN_NAME"

mv "$TMPDIR/$BIN_NAME" "$DEST" || err "could not move binary to $DEST"
chmod 0755 "$DEST"

info ""
info "  installed: $DEST"
info ""

# If the install dir isn't on PATH, the binary won't be reachable by
# name. This is the most common failure case (especially on macOS, where
# $HOME/.local/bin is not on the default PATH). Make the fix
# unmissable, including the exact shell-rc snippet to paste.
if ! is_on_path "$INSTALL_DIR"; then
    # Guess the user's shell rc file from $SHELL.
    case "${SHELL:-}" in
        */zsh)  rc_file="${ZDOTDIR:-$HOME}/.zshrc" ;;
        */bash) rc_file="$HOME/.bashrc"
                # macOS Terminal.app sources ~/.bash_profile, not ~/.bashrc
                [ "$OS" = "darwin" ] && rc_file="$HOME/.bash_profile" ;;
        */fish) rc_file="$HOME/.config/fish/config.fish" ;;
        *)      rc_file="your shell rc" ;;
    esac

    info "  ⚠  $INSTALL_DIR is not on your PATH — \`$BIN_NAME\` won't be found yet."
    info ""
    info "  To fix, add it to PATH. One-liner for your shell:"
    info ""
    case "${SHELL:-}" in
        */fish)
            info "      echo 'set -x PATH \"$INSTALL_DIR\" \$PATH' >> $rc_file && source $rc_file"
            ;;
        *)
            info "      echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> $rc_file && source $rc_file"
            ;;
    esac
    info ""
    info "  Or run \`$DEST --version\` directly to verify the install."
else
    info "  Run \`$BIN_NAME --version\` to verify."
fi
