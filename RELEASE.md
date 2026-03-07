# Release Process

## Version Management

This project maintains version synchronization across multiple plugin configuration files. Always update versions **before** tagging.

## Release Workflow

1. **Update version files** to the new version:
   ```bash
   bash scripts/update-marketplace-version.sh vX.X.X
   ```

2. **Verify the changes**:
   ```bash
   git diff .claude-plugin/
   ```

3. **Commit the version bump**:
   ```bash
   git add .claude-plugin/
   git commit -m "chore: bump version to X.X.X"
   ```

4. **Tag the release**:
   ```bash
   git tag vX.X.X
   ```

5. **Push commits and tag**:
   ```bash
   git push origin main vX.X.X
   ```

This triggers GitHub Actions to run GoReleaser, which builds binaries and publishes the release to GitHub.

## Files Updated by Version Script

- `.claude-plugin/marketplace.json` - Marketplace entry version
- `.claude-plugin/plugin/plugin.json` - Plugin manifest version

Both must match the release tag for marketplace installations to work correctly.
