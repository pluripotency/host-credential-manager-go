#!/bin/bash
# Move to the script's directory to ensure relative paths work
cd "$(dirname "$0")"

echo "Stopping container"
docker compose down

