#!/bin/bash
# Shrinkray QSV Test Setup Script
# For Intel UHD 630 (Coffee Lake) on Ubuntu

set -e

echo "=== Shrinkray QSV Test Environment Setup ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Install Intel media drivers
echo -e "${YELLOW}[1/5] Installing Intel media drivers...${NC}"
sudo apt update
sudo apt install -y \
    intel-media-va-driver-non-free \
    vainfo \
    intel-gpu-tools

# Step 2: Install Docker if not present
echo -e "${YELLOW}[2/5] Installing Docker...${NC}"
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sudo sh
    sudo usermod -aG docker $USER
    echo -e "${YELLOW}NOTE: You may need to log out and back in for Docker group to take effect${NC}"
else
    echo "Docker already installed"
fi

# Step 3: Check GPU access
echo -e "${YELLOW}[3/5] Checking GPU access...${NC}"
if [ -d "/dev/dri" ]; then
    echo -e "${GREEN}✓ /dev/dri exists${NC}"
    ls -la /dev/dri/
else
    echo -e "${RED}✗ /dev/dri not found - GPU passthrough won't work${NC}"
    exit 1
fi

# Check VAAPI
echo ""
echo "VAAPI info:"
vainfo 2>&1 | head -20 || echo "vainfo failed - drivers may not be loaded yet"

# Step 4: Create test directories
echo -e "${YELLOW}[4/5] Creating test directories...${NC}"
mkdir -p ~/shrinkray-test/media
mkdir -p ~/shrinkray-test/config
mkdir -p ~/shrinkray-test/temp

echo -e "${GREEN}Created:${NC}"
echo "  ~/shrinkray-test/media  - Put test videos here"
echo "  ~/shrinkray-test/config - Shrinkray config"
echo "  ~/shrinkray-test/temp   - Temp files during transcode"

# Step 5: Pull Shrinkray image
echo -e "${YELLOW}[5/5] Pulling Shrinkray Docker image...${NC}"
sudo docker pull ghcr.io/gwlsn/shrinkray:latest

echo ""
echo -e "${GREEN}=== Setup Complete ===${NC}"
echo ""
echo "Next steps:"
echo ""
echo "1. Put your test video in ~/shrinkray-test/media/"
echo ""
echo "2. Run Shrinkray with:"
echo "   docker run -d --name shrinkray \\"
echo "     --device=/dev/dri \\"
echo "     -v ~/shrinkray-test/media:/media \\"
echo "     -v ~/shrinkray-test/config:/config \\"
echo "     -v ~/shrinkray-test/temp:/temp \\"
echo "     -p 8080:8080 \\"
echo "     ghcr.io/gwlsn/shrinkray:latest"
echo ""
echo "3. Enable debug logging:"
echo "   docker exec shrinkray sh -c 'echo \"log_level: debug\" >> /config/shrinkray.yaml'"
echo "   docker restart shrinkray"
echo ""
echo "4. Open http://localhost:8080 in your browser"
echo ""
echo "5. To test FFmpeg directly in the container:"
echo "   docker exec -it shrinkray /bin/sh"
echo ""
