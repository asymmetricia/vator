#!/bin/sh

set -e

trap 'rm -f static/js/graph.js' EXIT

[ -d "$HOME/.nvm" ] || curl -L -s -f -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

export NVM_DIR="$HOME/.nvm"
. "$NVM_DIR/nvm.sh"

echo "Installing node v22 ..."
nvm install 22
cd static/js

echo "Installing dependencies...,"
npm install

echo "Compiling typescript..."
npm exec tsc

trap - EXIT
