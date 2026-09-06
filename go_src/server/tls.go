package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// IsCertRevoked checks if a given certificate is listed in the CRL file.
func IsCertRevoked(crlFile string, cert *x509.Certificate) bool {
	if cert == nil || cert.SerialNumber == nil {
		return false
	}
	if crlFile == "" {
		crlFile = "cert/crl.pem"
	}
	crlData, err := os.ReadFile(crlFile)
	if err != nil {
		// CRL file doesn't exist or cannot be read, no cert is considered revoked
		return false
	}
	block, _ := pem.Decode(crlData)
	if block == nil {
		return false
	}
	crl, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return false
	}

	for _, entry := range crl.RevokedCertificateEntries {
		if entry.SerialNumber != nil && entry.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			return true
		}
	}
	for _, entry := range crl.RevokedCertificates {
		if entry.SerialNumber != nil && entry.SerialNumber.Cmp(cert.SerialNumber) == 0 {
			return true
		}
	}
	return false
}

// BuildTLSConfig constructs a *tls.Config supporting mTLS and CRL checking.
func BuildTLSConfig(certFile, keyFile, caCertFile, crlFile string) (*tls.Config, error) {
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	if caCertFile != "" {
		if caData, err := os.ReadFile(caCertFile); err == nil && len(caData) > 0 {
			clientCAs := x509.NewCertPool()
			if clientCAs.AppendCertsFromPEM(caData) {
				tlsConfig.ClientCAs = clientCAs
				// Verify client cert if provided, allowing browsers to connect without client certs
				tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
			}
		}
	}

	// CRL checking on TLS connection verification
	if crlFile != "" {
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			for _, peerCert := range cs.PeerCertificates {
				if IsCertRevoked(crlFile, peerCert) {
					return fmt.Errorf("tls: client certificate has been revoked (serial: %x)", peerCert.SerialNumber)
				}
			}
			return nil
		}
	}

	return tlsConfig, nil
}

// RequireClientCertMiddleware enforces valid mTLS client certificates on CLI endpoints.
func RequireClientCertMiddleware(crlFile string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if os.Getenv("DISABLE_MTLS") == "1" || os.Getenv("DISABLE_MTLS") == "true" {
				return next(c)
			}

			tlsState := c.Request().TLS
			if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Client certificate required (mTLS)",
				})
			}

			peerCert := tlsState.PeerCertificates[0]
			if IsCertRevoked(crlFile, peerCert) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Client certificate has been revoked",
				})
			}

			return next(c)
		}
	}
}
