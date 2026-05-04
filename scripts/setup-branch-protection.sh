#!/usr/bin/env bash
# Sets up branch protection on main so CI must pass before merge.
# Auto-merge is per-PR: run `gh pr merge --auto <pr>` after opening a PR.
# Requires: gh CLI authenticated with repo scope.

set -euo pipefail

OWNER="sebastiaankok"
REPO="agents"
BRANCH="main"

echo "Setting up branch protection on ${OWNER}/${REPO}:${BRANCH} ..."

gh api "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" \
  --method PUT \
  -f enforce_admins=false \
  -F required_status_checks='{"strict":true,"contexts":["CI"]}' \
  -F restrictions='{"users":[],"teams":[],"apps":[]}'

echo ""
echo "Done. Branch protection is configured."
echo ""
echo "To auto-merge a PR when CI passes:"
echo "  gh pr merge --auto <pr-number>"
