#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_IP="${SERVER_IP:-127.0.0.1}"
SERVER_PORT="${SERVER_PORT:-8080}"
SERVER_URL="${HCM_URL:-https://${SERVER_IP}:${SERVER_PORT}}"

echo "=================================================="
echo " Starting Host Credential Manager CLI Client"
echo " Server URL: ${SERVER_URL}"
echo "=================================================="

CERT_ARG=""
if [ -f "${SCRIPT_DIR}/cert/cacert.pem" ]; then
  CERT_ARG="--cert ${SCRIPT_DIR}/cert/cacert.pem"
fi

CLIENT_CERT_ARG=""
if [ -f "${SCRIPT_DIR}/cert/client_cert.pem" ] && [ -f "${SCRIPT_DIR}/cert/client_key.pem" ]; then
  CLIENT_CERT_ARG="--client-cert ${SCRIPT_DIR}/cert/client_cert.pem --client-key ${SCRIPT_DIR}/cert/client_key.pem"
fi

exec "${SCRIPT_DIR}/hcm-client" --url "${SERVER_URL}" ${CERT_ARG} ${CLIENT_CERT_ARG} "$@"
