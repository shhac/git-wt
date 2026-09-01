#!/bin/bash
# Setup script for git-wt demo recording.
#
# Everything happens inside /tmp/wt-demo — a throwaway repo created fresh on
# every run. No real repository, remote, or global git config is touched, and
# nothing is pushed anywhere.
set -e

rm -rf /tmp/wt-demo
mkdir -p /tmp/wt-demo/acme-api
cd /tmp/wt-demo/acme-api

export GIT_AUTHOR_NAME=Demo GIT_AUTHOR_EMAIL=demo@example.com
export GIT_COMMITTER_NAME=Demo GIT_COMMITTER_EMAIL=demo@example.com

git init -q -b main

cat > server.js << 'JSEOF'
export const port = 8080

export function health() {
  return { status: "ok" }
}
JSEOF

cat > .env << 'ENVEOF'
API_KEY=demo-not-a-real-key
ENVEOF

printf '.env\n.worktrees/\n' > .gitignore
printf '.env\n' > .git-wt-copy-files

git add -A
git commit -qm "initial"

# A second branch, so `gwt add` has an existing branch to check out.
git branch fix/rate-limit
