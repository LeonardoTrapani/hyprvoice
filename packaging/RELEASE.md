# Release Process

This document describes how to create a new release of hyprvoice.

## Automated Release Process

### 1. Create a Release Tag

```bash
# Make sure you're on main branch with latest changes
git checkout main
git pull origin main

# Create and push a version tag
git tag v0.1.0
git push origin v0.1.0
```

### 2. GitHub Actions Automatically:

- ✅ Builds the binary with CGO for Linux x86_64
- ✅ Runs all tests  
- ✅ Creates a GitHub release with changelog
- ✅ Uploads `hyprvoice-linux-x86_64` binary
- ✅ Generates and uploads SHA256 checksums

### 3. Update AUR Package

```bash
cd packaging/
./update-aur.sh v0.1.0
```

That's it! The script handles everything:
- Updates PKGBUILD version and checksums
- Copies files to AUR repository  
- Generates .SRCINFO
- Tests the build
- Commits and pushes to AUR (with confirmation)

## Manual Release (if needed)

### Build Binary

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o hyprvoice-linux-x86_64 ./cmd/hyprvoice
```

### Create Release

1. Go to GitHub Releases
2. Click "Create a new release"
3. Tag: `v0.1.0`
4. Release title: `Release 0.1.0`
5. Upload `hyprvoice-linux-x86_64`
6. Publish release

## Version Scheme

- **Major.Minor.Patch** (e.g., `0.1.0`)
- **Major**: Breaking changes
- **Minor**: New features, backwards compatible
- **Patch**: Bug fixes, small improvements

## Complete Release Checklist

### Pre-release
- [ ] All tests pass: `go test ./...`
- [ ] Binary builds: `go build ./cmd/hyprvoice` 
- [ ] Configure command works: `./hyprvoice configure --help`
- [ ] Version bumped in any relevant files
- [ ] Changes documented

### GitHub Release  
- [ ] Create and push version tag: `git tag v0.1.0 && git push origin v0.1.0`
- [ ] Verify GitHub Actions completed successfully
- [ ] Verify binary uploaded to GitHub releases
- [ ] Verify checksums generated

### AUR Package Update (if AUR package exists)
- [ ] Run complete AUR update: `./packaging/update-aur.sh 0.1.0`
- [ ] Verify AUR package page updated

### Post-release Verification
- [ ] Test AUR installation: `yay -S hyprvoice-bin`  
- [ ] Test configure command: `hyprvoice configure`
- [ ] Test service: `systemctl --user status hyprvoice.service`
- [ ] Update project README if needed

## Files Updated in Release

- `packaging/PKGBUILD` - Version and checksums
- GitHub Release - Binary and checksums
- AUR repository - Updated package

## Troubleshooting

### Build Fails
- Check that all CGO dependencies are installed
- Ensure Go version matches workflow (1.21+)

### AUR Package Issues  
- Run `makepkg -si` to test locally
- Check checksums match: `updpkgsums`
- Verify binary downloads correctly

### GitHub Actions Issues
- Check workflow logs in Actions tab
- Ensure tag follows `v*` pattern
- Verify GITHUB_TOKEN has necessary permissions
