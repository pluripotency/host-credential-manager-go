package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTargetNormalization(t *testing.T) {
	cases := []struct {
		target        Target
		expectedProto string
		expectedPort  int
	}{
		{
			target:        Target{Protocol: "ssh", Port: "22"},
			expectedProto: "ssh",
			expectedPort:  22,
		},
		{
			target:        Target{Protocol: "SSH", Port: "10022"},
			expectedProto: "ssh",
			expectedPort:  10022,
		},
		{
			target:        Target{Protocol: "telnet", Port: "23"},
			expectedProto: "telnet",
			expectedPort:  23,
		},
		{
			target:        Target{Protocol: "TELNET", Port: ""},
			expectedProto: "telnet",
			expectedPort:  23,
		},
		{
			target:        Target{Protocol: "", Port: "23"},
			expectedProto: "telnet",
			expectedPort:  23,
		},
		{
			target:        Target{Protocol: "", Port: "22"},
			expectedProto: "ssh",
			expectedPort:  22,
		},
	}

	for _, c := range cases {
		proto := c.target.NormalizedProtocol()
		if proto != c.expectedProto {
			t.Errorf("NormalizedProtocol() = %s, expected %s", proto, c.expectedProto)
		}
		port := c.target.ResolvedPort()
		if port != c.expectedPort {
			t.Errorf("ResolvedPort() = %d, expected %d", port, c.expectedPort)
		}
	}
}

func TestFilterTargets(t *testing.T) {
	targets := []Target{
		{
			Hostname: "switch-floor2-core",
			IP:       "192.168.20.2",
			Port:     "23",
			Username: "admin",
			Protocol: "telnet",
			Platform: "Cisco",
			Tags:     "network,switch",
		},
		{
			Hostname: "almalinux9-srv01.internal",
			IP:       "10.0.3.50",
			Port:     "22",
			Username: "deploy",
			Protocol: "ssh",
			Platform: "Linux",
			OS:       "AlmaLinux 9",
			Tags:     "production,database",
		},
	}

	// 1. Filter by protocol
	res := filterTargets(targets, "telnet")
	if len(res) != 1 || res[0].Hostname != "switch-floor2-core" {
		t.Errorf("expected switch-floor2-core when filtering by telnet, got: %+v", res)
	}

	// 2. Filter by OS
	res = filterTargets(targets, "almalinux")
	if len(res) != 1 || res[0].Hostname != "almalinux9-srv01.internal" {
		t.Errorf("expected almalinux9 when filtering by almalinux, got: %+v", res)
	}

	// 3. Multi-token match
	res = filterTargets(targets, "cisco 23")
	if len(res) != 1 || res[0].Hostname != "switch-floor2-core" {
		t.Errorf("expected cisco 23 match, got: %+v", res)
	}

	// 4. Non-matching query
	res = filterTargets(targets, "windows non-existent")
	if len(res) != 0 {
		t.Errorf("expected 0 results, got: %+v", res)
	}
}

func TestFetchTargetsAndCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ssh-fzf" {
			if r.Method == http.MethodGet {
				data := []Target{
					{
						Hostname: "switch-floor2-core",
						IP:       "192.168.20.2",
						Port:     "23",
						Username: "admin",
						Protocol: "telnet",
						Platform: "Cisco",
					},
					{
						Hostname: "almalinux9-srv01.internal",
						IP:       "10.0.3.50",
						Port:     "22",
						Username: "deploy",
						Protocol: "ssh",
						Platform: "Linux",
						OS:       "AlmaLinux 9",
					},
				}
				json.NewEncoder(w).Encode(data)
				return
			} else if r.Method == http.MethodPost {
				var req map[string]string
				json.NewDecoder(r.Body).Decode(&req)
				if req["masterpassword"] == "secret" && req["hostname"] == "switch-floor2-core" {
					json.NewEncoder(w).Encode(map[string]string{"value": "switchpass"})
					return
				}
				json.NewEncoder(w).Encode(map[string]string{"value": ""})
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := ts.Client()

	targets, err := fetchTargets(client, ts.URL)
	if err != nil {
		t.Fatalf("fetchTargets failed: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	// Test credentials query
	pwd, err := fetchTargetPassword(client, ts.URL, "switch-floor2-core", "admin", "secret")
	if err != nil {
		t.Fatalf("fetchTargetPassword failed: %v", err)
	}
	if pwd != "switchpass" {
		t.Fatalf("expected 'switchpass', got '%s'", pwd)
	}

	// Test invalid masterpassword
	pwdBad, err := fetchTargetPassword(client, ts.URL, "switch-floor2-core", "admin", "wrong")
	if err != nil {
		t.Fatalf("fetchTargetPassword failed: %v", err)
	}
	if pwdBad != "" {
		t.Fatalf("expected empty password for wrong masterpassword, got '%s'", pwdBad)
	}
}

func TestBuildHTTPClientTLS(t *testing.T) {
	// Start TLS server with test cert
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// Insecure mode should connect cleanly
	client := buildHTTPClient("", "", "", true)
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("insecure client failed to connect: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBuildHTTPClientMTLS(t *testing.T) {
	// Generate a CA cert and key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "TestCA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}
	caCert, _ := x509.ParseCertificate(caCertDER)

	// Generate Client cert and key signed by CA
	clientPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientPrivKey.PublicKey, caPrivKey)
	if err != nil {
		t.Fatalf("failed to create client cert: %v", err)
	}

	// Write temp files
	tmpDir := t.TempDir()
	caCertFile := filepath.Join(tmpDir, "cacert.pem")
	clientCertFile := filepath.Join(tmpDir, "client_cert.pem")
	clientKeyFile := filepath.Join(tmpDir, "client_key.pem")

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	clientPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	clientKeyBytes, _ := x509.MarshalPKCS8PrivateKey(clientPrivKey)
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: clientKeyBytes})

	_ = os.WriteFile(caCertFile, caPEM, 0644)
	_ = os.WriteFile(clientCertFile, clientPEM, 0644)
	_ = os.WriteFile(clientKeyFile, clientKeyPEM, 0600)

	// Start mTLS test server
	clientPool := x509.NewCertPool()
	clientPool.AddCert(caCert)

	serverReceivedClientCert := false
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			if r.TLS.PeerCertificates[0].Subject.CommonName == "test-client" {
				serverReceivedClientCert = true
			}
		}
		w.Write([]byte("ok"))
	}))
	ts.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientPool,
	}
	ts.StartTLS()
	defer ts.Close()

	// 1. Client with client cert connects successfully
	client := buildHTTPClient(caCertFile, clientCertFile, clientKeyFile, true)
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("mTLS client failed to connect: %v", err)
	}
	resp.Body.Close()
	if !serverReceivedClientCert {
		t.Errorf("expected server to receive test-client certificate")
	}

	// 2. Client without client cert is rejected by mTLS server
	noCertClient := buildHTTPClient(caCertFile, "", "", true)
	_, err = noCertClient.Get(ts.URL)
	if err == nil {
		t.Errorf("expected connection error without client cert, got nil")
	}
}
