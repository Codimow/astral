#!/bin/bash
# Demo script showing Astral in action

set -e  # Exit on error

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Astral (asl) - Demo${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ASL="$SCRIPT_DIR/bin/asl"

# Create temporary workspace
DEMO_DIR=$(mktemp -d -t astral-demo-XXXXXX)
echo -e "${GREEN}📁 Demo directory: $DEMO_DIR${NC}"
echo

cd "$DEMO_DIR"

# Initialize
echo -e "${YELLOW}▶ asl init${NC}"
"$ASL" init
echo

# Create files
echo "Hello, Astral!" > README.md
echo "# My Project" > main.go

# First commit
echo -e "${YELLOW}▶ asl save -m \"Initial commit\"${NC}"
"$ASL" save -m "Initial commit"
echo

# Create branch
echo -e "${YELLOW}▶ asl branch feature-1${NC}"
"$ASL" branch feature-1
echo

# List branches
echo -e "${YELLOW}▶ asl branch${NC}"
"$ASL" branch
echo

# Make changes
echo "Updated!" >> README.md

# Second commit
echo -e "${YELLOW}▶ asl save -m \"Update README\"${NC}"
"$ASL" save -m "Update README"
echo

# View log
echo -e "${YELLOW}▶ asl log${NC}"
"$ASL" log
echo

# View stack
echo -e "${YELLOW}▶ asl stack${NC}"
"$ASL" stack
echo

# Show commit
echo -e "${YELLOW}▶ asl show${NC}"
"$ASL" show
echo

# Cleanup
rm -rf "$DEMO_DIR"

echo -e "${GREEN}✓ Demo completed!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
