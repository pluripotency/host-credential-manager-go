#!/bin/bash
# Move to the script's directory to ensure relative paths work
cd "$(dirname "$0")"

if [ ! -d "../config" ]; then
  echo "Copying ../test_config to ../config..."
  cp -r ../test_config ../config
fi

echo "Building and starting container in detached mode..."
docker compose up -d

echo "Streaming logs (Ctrl+C to exit)..."
docker compose logs -f
