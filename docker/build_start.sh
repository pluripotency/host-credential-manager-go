#!/bin/bash
# Move to the script's directory to ensure relative paths work
cd "$(dirname "$0")"

echo "Building and starting container in detached mode..."
docker compose up --build -d

echo "Streaming logs (Ctrl+C to exit)..."
docker compose logs -f
