#!/bin/bash
# Helper script to update PKGBUILD after a new release

set -e

if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.0"
    exit 1
fi

VERSION=$1
echo "Updating PKGBUILD for version $VERSION..."

# Update version in PKGBUILD
sed -i "s/pkgver=.*/pkgver=${VERSION}/" PKGBUILD

# Update checksums
echo "Updating checksums..."
updpkgsums

echo "✅ PKGBUILD updated for version $VERSION"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Test the package: makepkg -si"
echo "3. Commit and push to AUR"