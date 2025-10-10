#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [ -f .env ]; then
  set -a
  source .env || true
  set +a
fi

if [ ! -x ./setup.sh ]; then
  chmod +x ./setup.sh || true
fi


echo "Building server..."
for bin in  sudocrypt25; do
  if [ -f "$bin" ]; then
    rm -f "$bin" || true
    echo "Removed old binary: $bin"
  fi
done

if go build -o sudocrypt25 .; then
  echo "Built sudocrypt25"
else
  echo "Build failed" >&2
  exit 1
fi
echo "███████╗ ██╗  ██╗ ██╗   ██╗ ███╗   ██╗"
echo "██╔════╝ ██║  ██║ ██║   ██║ ████╗  ██║"
echo "█████╗     ███╔═╝ ██║   ██║ ██╔██╗ ██║"
echo "██╔══╝   ██╔══██║ ██║   ██║ ██║╚██╗██║"
echo "███████╗ ██║  ██║ ╚██████╔╝ ██║ ╚████║"
echo "╚══════╝ ╚═╝  ╚═╝  ╚═════╝  ╚═╝  ╚═══╝"

./sudocrypt25