#!/usr/bin/env bash
# ==============================================================================
# sync-obsidian.sh
#
# Synchronizes documentation (Markdown files and assets) from
# host-credential-manager-go to the Obsidian appdocs directory.
#
# Notes:
# - Excludes task management files: TODO.md, DONE.md
# - Excludes source code, binaries, runtime data (TOML credentials), and certificates
# - Preserves directory hierarchy (docs/, hcm-client/, front/, etc.)
# - Uses --delete-excluded so excluded or removed files are pruned from Obsidian
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_DEST="/home/worker/Documents/obsidian/evox2/appdocs/hcm"
DEST_DIR="${1:-${OBSIDIAN_HCM_DIR:-$DEFAULT_DEST}}"

echo "=========================================================="
echo " Syncing HCM Documentation to Obsidian"
echo " Source:      $SCRIPT_DIR"
echo " Destination: $DEST_DIR"
echo "=========================================================="

mkdir -p "$DEST_DIR"

rsync -av --delete --delete-excluded \
  --exclude="TODO.md" \
  --exclude="DONE.md" \
  --exclude=".git/" \
  --exclude=".gemini/" \
  --exclude="node_modules/" \
  --exclude="front/node_modules/" \
  --exclude="front/dist/" \
  --exclude="front/src/" \
  --exclude="front/public/" \
  --exclude="data/" \
  --exclude="cert/" \
  --exclude="docker/" \
  --exclude="go_src/" \
  --exclude="*.go" \
  --exclude="*.sh" \
  --exclude="*.toml" \
  --include="*/" \
  --include="*.md" \
  --include="*.png" \
  --include="*.jpg" \
  --include="*.jpeg" \
  --include="*.svg" \
  --include="*.gif" \
  --include="*.webp" \
  --exclude="*" \
  --prune-empty-dirs \
  "$SCRIPT_DIR/" "$DEST_DIR/"

echo "----------------------------------------------------------"
echo "Sync completed successfully!"
echo "Current Obsidian documentation structure:"
find "$DEST_DIR" -type f | sort | sed "s|^$DEST_DIR/|  - |"
echo "=========================================================="
