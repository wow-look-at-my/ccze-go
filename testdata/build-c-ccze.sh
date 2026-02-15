#!/usr/bin/env bash
set -euo pipefail

# Build the C ccze binary from source using PCRE2 (via the compat shim).
# Usage: testdata/build-c-ccze.sh
#
# Prerequisites (ubuntu): sudo apt-get install -y libncurses-dev libpcre2-dev

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

git clone https://github.com/cornet/ccze.git /tmp/ccze
cd /tmp/ccze

# Drop in the PCRE1-to-PCRE2 shim header
cp "$SCRIPT_DIR/pcre_compat.h" src/pcre_compat.h
sed -i 's|#include <pcre.h>|#include "pcre_compat.h"|' src/ccze.h

# Create a pcre-config shim so ./configure finds "pcre"
cat > pcre-config << 'EOF'
#!/bin/sh
case "$1" in
  --cflags) echo "-DPCRE2_CODE_UNIT_WIDTH=8" ;;
  --libs)   echo "-lpcre2-8" ;;
  --version) echo "8.45" ;;
esac
EOF
chmod +x pcre-config
export PATH="/tmp/ccze:$PATH"

./configure && make

sudo cp src/ccze /usr/local/bin/ccze
sudo mkdir -p /usr/local/lib/ccze
sudo cp src/*.so /usr/local/lib/ccze/
