#!/usr/bin/env bash
set -euo pipefail

: "${ISSUE_NUMBER:?ISSUE_NUMBER must be set}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set}"
: "${REPO_URL:?REPO_URL must be set}"
: "${DEFAULT_BRANCH:=main}"

BRANCH="agent/issue-${ISSUE_NUMBER}"

export GH_TOKEN="${GITHUB_TOKEN}"

# Authenticate gh CLI before any clones (needed for private repos)
echo "${GITHUB_TOKEN}" | gh auth login --with-token

# 1. Clone agents skills repo
gh repo clone sebastiaankok/agents /skills -- --depth=1

# 2. Clone target repo
gh repo clone "${REPO_URL}" /workspace --
cd /workspace

# Configure git identity for commits inside the job
git config user.email "agent@agentctl"
git config user.name "agentctl"

# 3. Create branch
git checkout -b "${BRANCH}"

# 4. Fetch issue body
ISSUE_BODY="$(gh issue view "${ISSUE_NUMBER}" --repo "${REPO_URL}" --json body --jq '.body')"

# 5. Run opencode with issue body + /tdd skill
PROMPT="${ISSUE_BODY}

/tdd"

opencode --print "${PROMPT}"

# 6. Push branch
git push origin "${BRANCH}"

# 7. Open PR
gh pr create \
  --title "agent: issue #${ISSUE_NUMBER}" \
  --body "Closes #${ISSUE_NUMBER}" \
  --base "${DEFAULT_BRANCH}" \
  --head "${BRANCH}"

# 8. Enable auto-merge
gh pr merge --auto --squash
