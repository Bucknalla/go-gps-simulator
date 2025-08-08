#!/bin/bash
# Build script for go-gps-simulator with version information

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BINARY_NAME="go-gps-simulator"

# Get version information from git
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Determine version: use git tag if available, otherwise use "dev"
if [ -z "$GIT_TAG" ]; then
    VERSION="dev"
else
    VERSION="$GIT_TAG"
fi

echo -e "${GREEN}Building $BINARY_NAME${NC}"
echo -e "${YELLOW}Version: $VERSION${NC}"
echo -e "${YELLOW}Commit: $GIT_COMMIT${NC}"
echo -e "${YELLOW}Build Date: $BUILD_DATE${NC}"
echo ""

# Build with version information
LDFLAGS="-X main.Version=$VERSION -X main.Commit=$GIT_COMMIT -X main.BuildDate=$BUILD_DATE"

if [ "$1" = "release" ]; then
    echo -e "${GREEN}Building release version (optimized)${NC}"
    LDFLAGS="$LDFLAGS -s -w"
fi

go build -ldflags "$LDFLAGS" -o "$BINARY_NAME" .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Build successful!${NC}"
    echo -e "${GREEN}Binary: ./$BINARY_NAME${NC}"
    echo ""
    echo -e "${YELLOW}Test the version:${NC}"
    echo "./$BINARY_NAME --version"
else
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi
