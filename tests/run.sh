#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI="${ROOT_DIR}/bin/teams-quiet-hours"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

export TEAMS_QUIET_HOURS_SYSTEM_PREFIX="$TMP_DIR"

assert_file() {
  local path="$1"
  [[ -f "$path" ]] || {
    printf 'Expected file to exist: %s\n' "$path" >&2
    exit 1
  }
}

assert_no_file() {
  local path="$1"
  [[ ! -f "$path" ]] || {
    printf 'Expected file to be absent: %s\n' "$path" >&2
    exit 1
  }
}

assert_contains() {
  local path="$1"
  local expected="$2"
  grep -F "$expected" "$path" >/dev/null || {
    printf 'Expected %s to contain: %s\n' "$path" "$expected" >&2
    exit 1
  }
}

"$CLI" --version >/dev/null
"$CLI" install --browser all --start 09:00 --end 17:00 --timezone Africa/Lagos >/dev/null

assert_file "$TMP_DIR/usr/local/bin/teams-quiet-hours"
assert_file "$TMP_DIR/etc/cron.d/teams-quiet-hours"
assert_file "$TMP_DIR/etc/teams-quiet-hours/config"
assert_contains "$TMP_DIR/etc/cron.d/teams-quiet-hours" "CRON_TZ=Africa/Lagos"
assert_contains "$TMP_DIR/etc/cron.d/teams-quiet-hours" "0 9 * * 1-5 root"
assert_contains "$TMP_DIR/etc/cron.d/teams-quiet-hours" "0 17 * * 1-5 root"
assert_contains "$TMP_DIR/etc/cron.d/teams-quiet-hours" "0 0 * * 6,0 root"

"$CLI" block >/dev/null
assert_file "$TMP_DIR/etc/opt/chrome/policies/managed/teams-quiet-hours.json"
assert_file "$TMP_DIR/etc/chromium/policies/managed/teams-quiet-hours.json"
assert_file "$TMP_DIR/etc/chromium-browser/policies/managed/teams-quiet-hours.json"
assert_contains "$TMP_DIR/etc/opt/chrome/policies/managed/teams-quiet-hours.json" "NotificationsBlockedForUrls"

"$CLI" allow >/dev/null
assert_no_file "$TMP_DIR/etc/opt/chrome/policies/managed/teams-quiet-hours.json"
assert_no_file "$TMP_DIR/etc/chromium/policies/managed/teams-quiet-hours.json"
assert_no_file "$TMP_DIR/etc/chromium-browser/policies/managed/teams-quiet-hours.json"

"$CLI" block --browser chrome >/dev/null
assert_file "$TMP_DIR/etc/opt/chrome/policies/managed/teams-quiet-hours.json"
"$CLI" status --browser chrome >/dev/null
"$CLI" config >/dev/null

"$CLI" uninstall >/dev/null
assert_no_file "$TMP_DIR/usr/local/bin/teams-quiet-hours"
assert_no_file "$TMP_DIR/etc/cron.d/teams-quiet-hours"
assert_no_file "$TMP_DIR/etc/teams-quiet-hours/config"
assert_no_file "$TMP_DIR/etc/opt/chrome/policies/managed/teams-quiet-hours.json"

if "$CLI" install --start 9:00 >/dev/null 2>&1; then
  printf 'Expected invalid --start to fail\n' >&2
  exit 1
fi

printf 'All tests passed\n'
