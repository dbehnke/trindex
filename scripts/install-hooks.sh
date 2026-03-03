#!/bin/sh
# Install git hooks for Trindex development
# Usage: ./scripts/install-hooks.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOOKS_DIR="${REPO_ROOT}/.githooks"
GIT_HOOKS_DIR="${REPO_ROOT}/.git/hooks"

echo "Installing git hooks..."

# Ensure .git/hooks exists
mkdir -p "${GIT_HOOKS_DIR}"

# Install pre-commit hook
if [ -f "${HOOKS_DIR}/pre-commit" ]; then
    cp "${HOOKS_DIR}/pre-commit" "${GIT_HOOKS_DIR}/pre-commit"
    chmod +x "${GIT_HOOKS_DIR}/pre-commit"
    echo "✓ Installed pre-commit hook"
else
    echo "✗ pre-commit hook not found in ${HOOKS_DIR}"
    exit 1
fi

echo ""
echo "Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will run 'golangci-lint run' before each commit."
echo "To bypass the hook in an emergency, use: git commit --no-verify"
