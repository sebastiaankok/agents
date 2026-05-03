#!/usr/bin/env bash
set -euo pipefail

: "${ISSUE_NUMBER:?ISSUE_NUMBER must be set}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set}"
: "${REPO_URL:?REPO_URL must be set}"
: "${DEFAULT_BRANCH:=main}"

AGENTS_REPO="https://github.com/sebastiaankok/agents"
BRANCH="agent/issue-${ISSUE_NUMBER}"

export GH_TOKEN="${GITHUB_TOKEN}"

# 1. Clone agents skills repo
git clone --depth=1 "${AGENTS_REPO}" /skills

# 2. Clone target repo
git clone "${REPO_URL}" /workspace
cd /workspace

# Configure git identity for commits inside the job
git config user.email "agent@agentctl"
git config user.name "agentctl"

# Authenticate gh CLI with the provided token
echo "${GITHUB_TOKEN}" | gh auth login --with-token

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
