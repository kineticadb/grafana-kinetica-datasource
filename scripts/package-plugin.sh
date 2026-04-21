#!/bin/bash

##
## Package Grafana Plugin for Distribution
##
## This script creates a deployment archive from the dist directory and generates
## a SHA256 hash for verification. This is useful for manual releases or testing
## the packaging process locally.
##
## The GitHub release workflow (.github/workflows/release.yml) uses
## grafana/plugin-actions/build-plugin which does this automatically.
##
## Usage:
##   ./scripts/package-plugin.sh [version]
##
## Example:
##   ./scripts/package-plugin.sh 1.0.0
##
## Output:
##   - kinetica-datasource-<version>.zip (the plugin archive)
##   - kinetica-datasource-<version>.zip.sha256 (SHA256 hash for verification)
##

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Plugin configuration
PLUGIN_ID="grafana-kinetica-datasource"
DIST_DIR="dist"
OUTPUT_DIR="."

# Get version from argument or package.json
if [ -n "$1" ]; then
    VERSION="$1"
else
    VERSION=$(node -p "require('./package.json').version")
fi

echo -e "${GREEN}Packaging Grafana Plugin${NC}"
echo "Plugin ID: ${PLUGIN_ID}"
echo "Version: ${VERSION}"
echo "Dist directory: ${DIST_DIR}"
echo ""

# Check if dist directory exists
if [ ! -d "${DIST_DIR}" ]; then
    echo -e "${RED}Error: ${DIST_DIR} directory not found${NC}"
    echo "Run 'npm run build' first to build the plugin"
    exit 1
fi

# Check if dist directory has content
if [ -z "$(ls -A ${DIST_DIR})" ]; then
    echo -e "${RED}Error: ${DIST_DIR} directory is empty${NC}"
    echo "Run 'npm run build' first to build the plugin"
    exit 1
fi

# Archive name
ARCHIVE_NAME="${PLUGIN_ID}-${VERSION}.zip"
ARCHIVE_PATH="${OUTPUT_DIR}/${ARCHIVE_NAME}"

echo -e "${YELLOW}Creating archive...${NC}"

# Remove old archive if exists
if [ -f "${ARCHIVE_PATH}" ]; then
    echo "Removing old archive: ${ARCHIVE_PATH}"
    rm "${ARCHIVE_PATH}"
fi

# Create zip archive
# The archive should contain the plugin files inside a directory named after the plugin ID
# Temporarily rename dist to plugin ID for correct archive structure
mv "${DIST_DIR}" "${PLUGIN_ID}"
zip -r "${ARCHIVE_NAME}" "${PLUGIN_ID}" -x "*.map" -x ".DS_Store"
# Rename back to dist
mv "${PLUGIN_ID}" "${DIST_DIR}"

echo -e "${GREEN}✓ Archive created: ${ARCHIVE_PATH}${NC}"

# Get archive size
ARCHIVE_SIZE=$(du -h "${ARCHIVE_PATH}" | cut -f1)
echo "Archive size: ${ARCHIVE_SIZE}"

# List contents of archive
echo ""
echo -e "${YELLOW}Archive contents:${NC}"
unzip -l "${ARCHIVE_PATH}" | head -20

# Generate SHA256 hash
echo ""
echo -e "${YELLOW}Generating SHA256 hash...${NC}"

HASH_FILE="${ARCHIVE_PATH}.sha256"

# Generate hash (compatible with both macOS and Linux)
if command -v sha256sum &> /dev/null; then
    # Linux
    sha256sum "${ARCHIVE_PATH}" > "${HASH_FILE}"
    HASH=$(sha256sum "${ARCHIVE_PATH}" | cut -d' ' -f1)
elif command -v shasum &> /dev/null; then
    # macOS
    shasum -a 256 "${ARCHIVE_PATH}" > "${HASH_FILE}"
    HASH=$(shasum -a 256 "${ARCHIVE_PATH}" | cut -d' ' -f1)
else
    echo -e "${RED}Error: Neither sha256sum nor shasum found${NC}"
    echo "Cannot generate hash"
    exit 1
fi

echo -e "${GREEN}✓ Hash file created: ${HASH_FILE}${NC}"
echo "SHA256: ${HASH}"

# Summary
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Package Created Successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Archive:  ${ARCHIVE_PATH}"
echo "Hash:     ${HASH_FILE}"
echo "Size:     ${ARCHIVE_SIZE}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Test the plugin by extracting the archive to your Grafana plugins directory"
echo "2. Upload both the .zip and .sha256 files to GitHub release"
echo "3. Update release notes with installation instructions"
echo ""
echo -e "${GREEN}For GitHub release:${NC}"
echo "  git tag v${VERSION}"
echo "  git push origin v${VERSION}"
echo ""
