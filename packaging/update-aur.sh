#!/bin/bash
# Complete script to update AUR package after a new release

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.0"
    exit 1
fi

VERSION=$1
AUR_DIR="../hyprvoice-bin"

echo "🚀 Updating AUR package for version $VERSION..."
echo ""

# Check if we're in the packaging directory
if [ ! -f "PKGBUILD" ]; then
    echo "❌ Error: PKGBUILD not found. Run this from the packaging/ directory."
    exit 1
fi

# Check if AUR directory exists
if [ ! -d "$AUR_DIR" ]; then
    echo "❌ Error: AUR directory not found at $AUR_DIR"
    echo "   Expected structure:"
    echo "   ├── hyprvoice/           # Main repo"
    echo "   │   └── packaging/       # You are here"
    echo "   └── hyprvoice-bin/       # AUR repo"
    echo ""
    echo "   Run the initial AUR setup first."
    exit 1
fi

# Update version in PKGBUILD
echo "📝 Updating version to $VERSION..."
sed -i "s/pkgver=.*/pkgver=${VERSION}/" PKGBUILD

# Update checksums
echo "🔐 Updating checksums..."
if ! updpkgsums; then
    echo "❌ Error: Failed to update checksums."
    echo "   Make sure GitHub release v$VERSION exists and is accessible."
    exit 1
fi

# Copy files to AUR repo
echo "📋 Copying files to AUR repository..."
cp PKGBUILD "$AUR_DIR/"
cp hyprvoice.service "$AUR_DIR/"
cp hyprvoice.install "$AUR_DIR/"

# Switch to AUR directory
cd "$AUR_DIR"

# Generate .SRCINFO
echo "📄 Generating .SRCINFO..."
makepkg --printsrcinfo > .SRCINFO

# Test build
echo "🔨 Testing package build..."
if ! makepkg --noextract --nodeps; then
    echo "❌ Error: Package build failed."
    exit 1
fi

echo "✅ Package build successful!"
echo ""

# Show git status
echo "📊 AUR repository status:"
git status --short

echo ""
echo "🚀 Ready to publish to AUR:"
echo "   git add ."
echo "   git commit -m \"Update to version $VERSION\""
echo "   git push origin master"
echo ""

read -p "Push to AUR now? (y/N): " push_confirm
if [[ $push_confirm == [yY] ]]; then
    git add .
    git commit -m "Update to version $VERSION"
    git push origin master
    echo ""
    echo "🎉 Successfully updated AUR package to v$VERSION!"
    echo "   AUR page: https://aur.archlinux.org/packages/hyprvoice-bin"
else
    echo "📝 To push later:"
    echo "   cd $AUR_DIR"
    echo "   git add . && git commit -m \"Update to version $VERSION\" && git push"
fi

echo ""
echo "✅ AUR update complete!"