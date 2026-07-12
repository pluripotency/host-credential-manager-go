#!/bin/bash
# Move to the script's directory to ensure relative paths work
cd "$(dirname "$0")"

echo "Building the Docker image..."
docker build -t host-credential-manager-go:latest -f Dockerfile ..

echo "Saving the Docker image to host-credential-manager-go.tgz..."
docker save host-credential-manager-go:latest | gzip > ../host-credential-manager-go.tgz

echo "Docker image saved successfully to host-credential-manager-go.tgz"
