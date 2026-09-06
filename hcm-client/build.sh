#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILT_DIR="${SCRIPT_DIR}/built"

echo "==> Preparing hcm-client build..."

# Create cert and built directories
mkdir -p "${SCRIPT_DIR}/cert"
touch "${SCRIPT_DIR}/cert/.keep"
mkdir -p "${BUILT_DIR}/cert"
touch "${BUILT_DIR}/cert/.keep"

# Copy latest CA cert from project root if available
if [ -f "${ROOT_DIR}/cert/cacert.pem" ]; then
  echo "==> Copying project Root CA (cert/cacert.pem) for embedding and output..."
  cp "${ROOT_DIR}/cert/cacert.pem" "${SCRIPT_DIR}/cert/cacert.pem"
  cp "${ROOT_DIR}/cert/cacert.pem" "${BUILT_DIR}/cert/cacert.pem"
elif [ -f "${SCRIPT_DIR}/cert/cacert.pem" ]; then
  cp "${SCRIPT_DIR}/cert/cacert.pem" "${BUILT_DIR}/cert/cacert.pem"
fi

# Copy client certificate and private key from project root if available
if [ -f "${ROOT_DIR}/cert/client_cert.pem" ] && [ -f "${ROOT_DIR}/cert/client_key.pem" ]; then
  echo "==> Copying client certificate and key for embedding and output..."
  cp "${ROOT_DIR}/cert/client_cert.pem" "${SCRIPT_DIR}/cert/client_cert.pem"
  cp "${ROOT_DIR}/cert/client_key.pem" "${SCRIPT_DIR}/cert/client_key.pem"
  cp "${ROOT_DIR}/cert/client_cert.pem" "${BUILT_DIR}/cert/client_cert.pem"
  cp "${ROOT_DIR}/cert/client_key.pem" "${BUILT_DIR}/cert/client_key.pem"
elif [ -f "${SCRIPT_DIR}/cert/client_cert.pem" ] && [ -f "${SCRIPT_DIR}/cert/client_key.pem" ]; then
  cp "${SCRIPT_DIR}/cert/client_cert.pem" "${BUILT_DIR}/cert/client_cert.pem"
  cp "${SCRIPT_DIR}/cert/client_key.pem" "${BUILT_DIR}/cert/client_key.pem"
fi

OUTPUT_BIN="${BUILT_DIR}/hcm-client"

echo "==> Compiling standalone Go hcm-client binary into ${BUILT_DIR}..."
(cd "${ROOT_DIR}" && go build -ldflags="-s -w" -o "${OUTPUT_BIN}" ./hcm-client)

chmod +x "${OUTPUT_BIN}"

if [ -f "${BUILT_DIR}/run.sh" ]; then
  chmod +x "${BUILT_DIR}/run.sh"
fi

echo "==> Build successful! Binary generated at: ${OUTPUT_BIN}"
