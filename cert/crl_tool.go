package main

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run crl_tool.go [init|revoke <cert_file>|check <cert_file>]")
		os.Exit(1)
	}

	certDir := "."
	if d := os.Getenv("CERT_DIR"); d != "" {
		certDir = d
	} else if _, err := os.Stat("cert/cacert.pem"); err == nil {
		certDir = "cert"
	} else if _, err := os.Stat("cacert.pem"); err == nil {
		certDir = "."
	}

	caCertPath := filepath.Join(certDir, "cacert.pem")
	caKeyPath := filepath.Join(certDir, "cakey.pem")
	crlPath := filepath.Join(certDir, "crl.pem")

	action := os.Args[1]

	switch action {
	case "init":
		if err := initCRL(caCertPath, caKeyPath, crlPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing CRL: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Initialized empty CRL at %s\n", crlPath)

	case "revoke":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run crl_tool.go revoke <cert_file>")
			os.Exit(1)
		}
		targetCertPath := os.Args[2]
		if err := revokeCert(caCertPath, caKeyPath, crlPath, targetCertPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error revoking certificate: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully revoked %s and updated %s\n", targetCertPath, crlPath)

	case "check":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run crl_tool.go check <cert_file>")
			os.Exit(1)
		}
		targetCertPath := os.Args[2]
		revoked, err := isRevoked(crlPath, targetCertPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking CRL: %v\n", err)
			os.Exit(1)
		}
		if revoked {
			fmt.Printf("Certificate %s is REVOKED\n", targetCertPath)
			os.Exit(2)
		} else {
			fmt.Printf("Certificate %s is VALID (not revoked)\n", targetCertPath)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		os.Exit(1)
	}
}

func loadCA(caCertPath, caKeyPath string) (*x509.Certificate, any, error) {
	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	block, _ := pem.Decode(caCertPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode CA cert PEM")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA cert: %w", err)
	}

	caKeyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse CA private key: %w", err)
		}
	}

	return caCert, key, nil
}

func initCRL(caCertPath, caKeyPath, crlPath string) error {
	caCert, caKey, err := loadCA(caCertPath, caKeyPath)
	if err != nil {
		return err
	}

	template := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-1 * time.Minute),
		NextUpdate: time.Now().Add(3650 * 24 * time.Hour),
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, template, caCert, caKey.(crypto.Signer))
	if err != nil {
		return fmt.Errorf("failed to create CRL: %w", err)
	}

	f, err := os.Create(crlPath)
	if err != nil {
		return fmt.Errorf("failed to create crl file: %w", err)
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{Type: "X509 CRL", Bytes: crlDER})
}

func revokeCert(caCertPath, caKeyPath, crlPath, targetCertPath string) error {
	caCert, caKey, err := loadCA(caCertPath, caKeyPath)
	if err != nil {
		return err
	}

	targetPEM, err := os.ReadFile(targetCertPath)
	if err != nil {
		return fmt.Errorf("failed to read target cert: %w", err)
	}
	targetBlock, _ := pem.Decode(targetPEM)
	if targetBlock == nil {
		return fmt.Errorf("failed to decode target cert PEM")
	}
	targetCert, err := x509.ParseCertificate(targetBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse target cert: %w", err)
	}

	// Read existing revoked serials from CRL if exists
	revokedEntries := []x509.RevocationListEntry{}
	revokedMap := make(map[string]bool)

	crlNumber := big.NewInt(1)
	if crlData, err := os.ReadFile(crlPath); err == nil {
		if block, _ := pem.Decode(crlData); block != nil {
			if existingCRL, err := x509.ParseRevocationList(block.Bytes); err == nil {
				if existingCRL.Number != nil {
					crlNumber = new(big.Int).Add(existingCRL.Number, big.NewInt(1))
				}
				for _, entry := range existingCRL.RevokedCertificateEntries {
					revokedEntries = append(revokedEntries, entry)
					revokedMap[entry.SerialNumber.String()] = true
				}
			}
		}
	}

	// Add target if not already revoked
	if !revokedMap[targetCert.SerialNumber.String()] {
		revokedEntries = append(revokedEntries, x509.RevocationListEntry{
			SerialNumber:   targetCert.SerialNumber,
			RevocationTime: time.Now(),
			ReasonCode:     4, // superseded
		})
	}

	template := &x509.RevocationList{
		Number:                    crlNumber,
		ThisUpdate:                time.Now().Add(-1 * time.Minute),
		NextUpdate:                time.Now().Add(3650 * 24 * time.Hour),
		RevokedCertificateEntries: revokedEntries,
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, template, caCert, caKey.(crypto.Signer))
	if err != nil {
		return fmt.Errorf("failed to create revocation list: %w", err)
	}

	f, err := os.Create(crlPath)
	if err != nil {
		return fmt.Errorf("failed to open CRL file: %w", err)
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{Type: "X509 CRL", Bytes: crlDER})
}

func isRevoked(crlPath, targetCertPath string) (bool, error) {
	crlData, err := os.ReadFile(crlPath)
	if err != nil {
		return false, fmt.Errorf("failed to read CRL: %w", err)
	}
	block, _ := pem.Decode(crlData)
	if block == nil {
		return false, fmt.Errorf("failed to decode CRL PEM")
	}
	crl, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse CRL: %w", err)
	}

	targetPEM, err := os.ReadFile(targetCertPath)
	if err != nil {
		return false, fmt.Errorf("failed to read target cert: %w", err)
	}
	tBlock, _ := pem.Decode(targetPEM)
	if tBlock == nil {
		return false, fmt.Errorf("failed to decode target cert PEM")
	}
	targetCert, err := x509.ParseCertificate(tBlock.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse target cert: %w", err)
	}

	for _, entry := range crl.RevokedCertificateEntries {
		if entry.SerialNumber.Cmp(targetCert.SerialNumber) == 0 {
			return true, nil
		}
	}

	for _, r := range crl.RevokedCertificates {
		if r.SerialNumber.Cmp(targetCert.SerialNumber) == 0 {
			return true, nil
		}
	}

	return false, nil
}
