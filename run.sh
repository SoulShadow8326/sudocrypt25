#!/usr/bin/env bash
set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
