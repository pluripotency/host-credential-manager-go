#!/bin/bash
CERT_DIR="$(cd "$(dirname "$0")"; pwd)"
CA_KEY="${CERT_DIR}/cakey.pem"
CA_CERT="${CERT_DIR}/cacert.pem"
SERVER_KEY="${CERT_DIR}/key.pem"
SERVER_CERT="${CERT_DIR}/cert.pem"
CSR="${CERT_DIR}/cert.csr"
EXT_CONF="${CERT_DIR}/extfile.cnf"

mkdir -p "${CERT_DIR}"

# Remove existing certificates to force regeneration
rm -f "${SERVER_KEY}" "${SERVER_CERT}" "${CA_CERT}" "${CA_KEY}"

echo "1. Generating Root CA with CA extensions..."
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:4096 \
  -keyout "${CA_KEY}" -out "${CA_CERT}" \
  -subj "/CN=MyLocalSSHCA" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"

echo "2. Generating Server Key and CSR..."
openssl req -new -nodes -newkey rsa:2048 \
  -keyout "${SERVER_KEY}" -out "${CSR}" \
  -subj "/CN=localhost"

echo "3. Creating extensions config..."
cat <<EOF > "${EXT_CONF}"
basicConstraints=CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
subjectAltName=DNS:localhost,IP:127.0.0.1
EOF

echo "4. Signing Server Certificate with Root CA..."
openssl x509 -req -days 365 -in "${CSR}" \
  -CA "${CA_CERT}" -CAkey "${CA_KEY}" -CAcreateserial \
  -out "${SERVER_CERT}" -extfile "${EXT_CONF}" -sha256

rm -f "${CSR}" "${EXT_CONF}" "${CERT_DIR}/cacert.srl"

echo "Certificates generated successfully in ${CERT_DIR}/"
