#!/bin/bash
# Update marketplace.json version to match the release tag
# Usage: ./scripts/update-marketplace-version.sh <version>
# Example: ./scripts/update-marketplace-version.sh v1.0.2

set -e

VERSION="${1}"
if [ -z "$VERSION" ]; then
    echo "Error: version argument required"
    exit 1
fi

# Strip 'v' prefix if present
VERSION="${VERSION#v}"

MARKETPLACE_FILE=".claude-plugin/marketplace.json"

# Update version in marketplace.json
sed -i.bak "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION}\"/" "$MARKETPLACE_FILE"
rm -f "$MARKETPLACE_FILE.bak"

echo "Updated marketplace version to $VERSION"
