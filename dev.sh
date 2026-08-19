#!/usr/bin/env bash

# Ensure Go binaries (air) and Node binaries (npm) are in PATH
# export PATH="$PATH:$(go env GOPATH 2>/dev/null || echo "$HOME/go")/bin:$HOME/.nodebrew/current/bin"

# Cleanup background processes upon termination
cleanup() {
  echo ""
  echo "Stopping development servers..."
  if [ -n "$AIR_PID" ]; then
    kill "$AIR_PID" 2>/dev/null
  fi
  if [ -n "$VITE_PID" ]; then
    kill "$VITE_PID" 2>/dev/null
  fi
  wait 2>/dev/null
  echo "Development environment stopped."
  exit 0
}

trap cleanup EXIT INT TERM

echo "Starting Go Air server..."
air &
AIR_PID=$!

echo "Starting Vite dev server..."
(cd front && npm run dev) &
VITE_PID=$!

wait
