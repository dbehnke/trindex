#!/bin/bash
# Update plugin version files to match the release tag
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

FILES=(
    ".claude-plugin/marketplace.json"
    ".claude-plugin/plugin/plugin.json"
)

# Update version in all plugin files
for FILE in "${FILES[@]}"; do
    if [ -f "$FILE" ]; then
        sed -i.bak "s/\"version\": \"[^\"]*\"/\"version\": \"${VERSION}\"/" "$FILE"
        rm -f "$FILE.bak"
        echo "Updated $FILE to version $VERSION"
    fi
done
