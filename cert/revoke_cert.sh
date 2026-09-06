#!/bin/bash
set -e

CERT_DIR="$(cd "$(dirname "$0")"; pwd)"
TARGET_CERT="${1:-${CERT_DIR}/client_cert.pem}"

if [ ! -f "${TARGET_CERT}" ]; then
  echo "Error: Target certificate not found: ${TARGET_CERT}"
  exit 1
fi

echo "Revoking certificate: ${TARGET_CERT}"
CERT_DIR="${CERT_DIR}" go run "${CERT_DIR}/crl_tool.go" revoke "${TARGET_CERT}"
