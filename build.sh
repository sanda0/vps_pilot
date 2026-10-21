#!/bin/bash

# VPS Pilot Build Script
# This script builds the React UI and embeds it into the Go binary

set -e

echo "🏗️  Building VPS Pilot..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we're in the right directory
if [ ! -f "build.sh" ]; then
    echo -e "${RED}❌ Error: Please run this script from the project root directory${NC}"
    exit 1
fi

# Step 1: Build the React UI
echo -e "${YELLOW}📦 Building React UI...${NC}"
cd client

# Check if node_modules exists, if not install dependencies
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}📥 Installing Node.js dependencies...${NC}"
    if command -v bun &> /dev/null; then
        bun install
    elif command -v npm &> /dev/null; then
        npm install
    else
        echo -e "${RED}❌ Error: Neither bun nor npm found. Please install Node.js and bun/npm.${NC}"
        exit 1
    fi
fi

# Build the React app
echo -e "${YELLOW}⚛️  Building React application...${NC}"
if command -v bun &> /dev/null; then
    bun run build
elif command -v npm &> /dev/null; then
    npm run build
else
    echo -e "${RED}❌ Error: Neither bun nor npm found.${NC}"
    exit 1
fi

# Vite writes straight into server/cmd/app/dist (see client/vite.config.ts)
if [ ! -f "../server/cmd/app/dist/index.html" ]; then
    echo -e "${RED}❌ Error: React build failed - server/cmd/app/dist/index.html not found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ React UI built into server/cmd/app/dist${NC}"
cd ../server

# Step 3: Build the Go server with embedded UI
echo -e "${YELLOW}🔧 Building Go server with embedded UI...${NC}"

# Build the Go binary
go build -ldflags "-s -w" -o vps_pilot .

if [ ! -f "vps_pilot" ]; then
    echo -e "${RED}❌ Error: Go build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go server built successfully${NC}"

# Step 4: Display success message
echo ""
echo -e "${GREEN}🎉 Build completed successfully!${NC}"
echo -e "${GREEN}📁 Binary location: server/vps_pilot${NC}"
echo ""
echo -e "${YELLOW}📋 Binary info:${NC}"
echo "  Size: $(du -h vps_pilot | cut -f1)"
echo ""
echo -e "${YELLOW}🚀 To run the server:${NC}"
echo "  cd server"
echo "  ./vps_pilot"
echo ""
echo -e "${YELLOW}🌐 Access:${NC}"
echo "  Frontend: http://localhost:8080"
echo "  API:      http://localhost:8080/api/v1"
echo ""

# Step 5: Build the agent (Rust)
if command -v cargo &> /dev/null; then
    echo -e "${YELLOW}🦀 Building agent...${NC}"
    (cd ../agent && cargo build --release)
    echo -e "${GREEN}✅ Agent binary: agent/target/release/vps-pilot-agent${NC}"
else
    echo -e "${YELLOW}⚠️  cargo not found, skipping agent build${NC}"
fi
