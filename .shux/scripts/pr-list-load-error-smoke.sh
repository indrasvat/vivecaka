#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="$ROOT/.shux/out"
SESSION="vivecaka-pr-list-error-smoke"
FAKE_BIN="$(mktemp -d)"
RUNNER="$FAKE_BIN/run-vivecaka.sh"

cleanup() {
  shux session kill "$SESSION" >/dev/null 2>&1 || true
  rm -rf "$FAKE_BIN"
}
trap cleanup EXIT

mkdir -p "$OUT_DIR"

cat >"$FAKE_BIN/gh" <<'SH'
#!/usr/bin/env sh
case "$1 $2" in
  "auth status")
    exit 0
    ;;
  "api user")
    printf 'indrasvat\n'
    ;;
  "api graphql")
    printf '{"data":{"repository":{"pullRequests":{"totalCount":7}}}}'
    ;;
  "pr list")
    sleep 3
    printf 'GraphQL: API rate limit exceeded\n' >&2
    exit 1
    ;;
  *)
    printf '{}\n'
    ;;
esac
SH
chmod +x "$FAKE_BIN/gh"

cat >"$RUNNER" <<SH
#!/usr/bin/env sh
export PATH="$FAKE_BIN:\$PATH"
exec "$ROOT/bin/vivecaka" --repo owner/repo
SH
chmod +x "$RUNNER"

make -C "$ROOT" build >/dev/null

shux session kill "$SESSION" >/dev/null 2>&1 || true
shux --format json session create "$SESSION" -d --title vivecaka-pr-list-error -- \
  "$RUNNER" >/dev/null
shux pane set-size -s "$SESSION" --cols 120 --rows 32 >/dev/null
shux pane wait-for -s "$SESSION" --text "Pull requests unavailable" --timeout-ms 15000 >/dev/null
sleep 9
shux pane capture -s "$SESSION" >"$OUT_DIR/pr-list-load-error.txt"
shux --format json pane snapshot -s "$SESSION" \
  | jq -r .png_base64 \
  | base64 -d >"$OUT_DIR/pr-list-load-error.png"

echo "wrote $OUT_DIR/pr-list-load-error.png"
