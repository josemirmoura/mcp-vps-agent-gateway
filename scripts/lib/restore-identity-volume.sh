#!/bin/sh
# Restore an integrated-identity data volume from a local snapshot.
# Runs only inside an isolated one-shot Docker container, with /target
# mounted to ONE volume and /backup mounted read-only.
set -eu
archive="${1:?snapshot file required}"
test -f "$archive"
test -d /target
cd /target
stage="$(mktemp -d /target/.portico-restore-stage.XXXXXXXX)"
previous="$(mktemp -d /target/.portico-restore-previous.XXXXXXXX)"
phase=staging

is_present() { [ -e "$1" ] || [ -L "$1" ]; }

on_failure() {
  code=$?
  trap - EXIT HUP INT TERM
  if [ "$phase" = saving ]; then
    # Original root contains some still-unmoved entries; only move the
    # original entries already staged in previous back into the root.
    for entry in "$previous"/* "$previous"/.[!.]* "$previous"/..?*; do
      is_present "$entry" || continue
      mv "$entry" /target/ || true
    done
  elif [ "$phase" = restoring ]; then
    # All old entries were moved into previous. Remove any already-restored
    # new entries, then put every original entry back.
    for entry in /target/* /target/.[!.]* /target/..?*; do
      is_present "$entry" || continue
      case "$entry" in "$stage"|"$previous") continue ;; esac
      rm -rf -- "$entry" || true
    done
    for entry in "$previous"/* "$previous"/.[!.]* "$previous"/..?*; do
      is_present "$entry" || continue
      mv "$entry" /target/ || true
    done
  fi
  # On failure retain both staging dirs for recovery/forensics if needed.
  exit "$code"
}
trap on_failure EXIT HUP INT TERM

# Snapshot decompression happens before disturbing a single current entry.
tar -tzf "$archive" >/dev/null
tar -xzf "$archive" -C "$stage"
phase=saving
for entry in /target/* /target/.[!.]* /target/..?*; do
  is_present "$entry" || continue
  case "$entry" in "$stage"|"$previous") continue ;; esac
  mv "$entry" "$previous/"
done

phase=restoring
for entry in "$stage"/* "$stage"/.[!.]* "$stage"/..?*; do
  is_present "$entry" || continue
  mv "$entry" /target/
done

phase=done
trap - EXIT HUP INT TERM
rm -rf -- "$stage" "$previous"
