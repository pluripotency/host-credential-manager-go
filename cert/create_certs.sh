#!/bin/bash
set -e

CERT_DIR="$(cd "$(dirname "$0")"; pwd)"
CA_KEY="${CERT_DIR}/cakey.pem"
CA_CERT="${CERT_DIR}/cacert.pem"
SERVER_KEY="${CERT_DIR}/key.pem"
SERVER_CERT="${CERT_DIR}/cert.pem"
CLIENT_KEY="${CERT_DIR}/client_key.pem"
CLIENT_CERT="${CERT_DIR}/client_cert.pem"
CSR="${CERT_DIR}/cert.csr"
CLIENT_CSR="${CERT_DIR}/client.csr"
EXT_CONF="${CERT_DIR}/extfile.cnf"
CLIENT_EXT_CONF="${CERT_DIR}/client_extfile.cnf"
CRL_FILE="${CERT_DIR}/crl.pem"

mkdir -p "${CERT_DIR}"

# Remove existing certificates to force regeneration
rm -f "${SERVER_KEY}" "${SERVER_CERT}" "${CLIENT_KEY}" "${CLIENT_CERT}" "${CA_CERT}" "${CA_KEY}" "${CRL_FILE}"

echo "1. Generating Root CA with CA and CRL extensions..."
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:4096 \
  -keyout "${CA_KEY}" -out "${CA_CERT}" \
  -subj "/CN=MyLocalSSHCA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign" \
  -addext "subjectKeyIdentifier=hash"

echo "2. Generating Server Key and CSR..."
openssl req -new -nodes -newkey rsa:2048 \
  -keyout "${SERVER_KEY}" -out "${CSR}" \
  -subj "/CN=localhost"

echo "3. Creating server extensions config..."
cat <<EOF > "${EXT_CONF}"
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:localhost,IP:127.0.0.1
EOF

echo "4. Signing Server Certificate with Root CA..."
openssl x509 -req -days 365 -in "${CSR}" \
  -CA "${CA_CERT}" -CAkey "${CA_KEY}" -CAcreateserial \
  -out "${SERVER_CERT}" -extfile "${EXT_CONF}" -sha256

echo "5. Generating Client Key and CSR..."
openssl req -new -nodes -newkey rsa:2048 \
  -keyout "${CLIENT_KEY}" -out "${CLIENT_CSR}" \
  -subj "/CN=hcm-client-user"

echo "6. Creating client extensions config..."
cat <<EOF > "${CLIENT_EXT_CONF}"
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF

echo "7. Signing Client Certificate with Root CA..."
openssl x509 -req -days 365 -in "${CLIENT_CSR}" \
  -CA "${CA_CERT}" -CAkey "${CA_KEY}" -CAcreateserial \
  -out "${CLIENT_CERT}" -extfile "${CLIENT_EXT_CONF}" -sha256

echo "8. Initializing empty CRL (Certificate Revocation List)..."
CERT_DIR="${CERT_DIR}" go run "${CERT_DIR}/crl_tool.go" init

rm -f "${CSR}" "${CLIENT_CSR}" "${EXT_CONF}" "${CLIENT_EXT_CONF}" "${CERT_DIR}/cacert.srl"

echo "Certificates & CRL generated successfully in ${CERT_DIR}/"
