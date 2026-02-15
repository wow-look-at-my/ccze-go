#!/usr/bin/env bash
set -euo pipefail

# Build the C ccze binary from source.
# Usage: testdata/build-c-ccze.sh
#
# Prerequisites (ubuntu): sudo apt-get install -y libncurses-dev libpcre3-dev

git clone https://github.com/cornet/ccze.git /tmp/ccze
cd /tmp/ccze

./configure && make

sudo cp src/ccze /usr/local/bin/ccze
sudo mkdir -p /usr/local/lib/ccze
sudo cp src/*.so /usr/local/lib/ccze/
