package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"host-credential-manager-go/go_src/db"
	"host-credential-manager-go/go_src/models"
)

func setupTestDB(t *testing.T) {
	tmpDir := t.TempDir()
	// Set up environment or files if needed
	os.Setenv("PORT", "8080")
	// Clean or init
	_ = os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
}

func TestExportHosts(t *testing.T) {
	defer os.RemoveAll("data")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/hostlist/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Save test hosts
	testHosts := []models.Host{
		{
			ID:          "exp1",
			Hostname:    "export-test-host",
			IP:          "192.168.1.50",
			Platform:    "Linux",
			Port:        "22",
			Tags:        "test",
			Description: "Export test",
			UpdatedAt:   "2026-01-01T00:00:00.000Z",
		},
	}
	if err := db.WriteHostList(testHosts); err != nil {
		t.Fatalf("failed to write test hosts: %v", err)
	}

	if err := exportHosts(c); err != nil {
		t.Fatalf("exportHosts returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected Content-Type text/csv, got %s", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "export-test-host") || !strings.Contains(body, "192.168.1.50") {
		t.Errorf("expected CSV body to contain host data, got:\n%s", body)
	}
}

func TestImportHostsMultipartCSV(t *testing.T) {
	defer os.RemoveAll("data")

	e := echo.New()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "hosts.csv")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	csvContent := `id,hostname,ip,platform,port,tags,description,updatedAt
imp1,import-host,10.20.30.40,Linux,22,import,Imported host,2026-01-01T00:00:00.000Z
`
	if _, err := fw.Write([]byte(csvContent)); err != nil {
		t.Fatalf("failed to write csv to form file: %v", err)
	}
	_ = mw.WriteField("merge", "false")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/hostlist/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := importHosts(c); err != nil {
		t.Fatalf("importHosts returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	hosts, err := db.ReadHostList()
	if err != nil {
		t.Fatalf("failed to read hostlist: %v", err)
	}

	found := false
	for _, h := range hosts {
		if h.Hostname == "import-host" && h.IP == "10.20.30.40" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected imported host in hostlist, got: %+v", hosts)
	}
}

func TestSSHFzfTargets(t *testing.T) {
	defer os.RemoveAll("data")

	e := echo.New()

	testHosts := []models.Host{
		{
			ID:       "h1",
			Hostname: "ssh-host-1",
			IP:       "192.168.10.1",
			Platform: "Linux",
			Accesslist: []models.AccessItem{
				{Protocol: "ssh", Port: "22"},
			},
		},
		{
			ID:       "h2",
			Hostname: "ssh-host-custom-port",
			IP:       "192.168.10.2",
			Platform: "Linux",
			Accesslist: []models.AccessItem{
				{Protocol: "ssh", Port: "10022"},
			},
		},
		{
			ID:       "h3",
			Hostname: "web-only-host",
			IP:       "192.168.10.3",
			Platform: "Linux",
			Accesslist: []models.AccessItem{
				{Protocol: "https", Port: "443"},
			},
		},
		{
			ID:       "h4",
			Hostname: "telnet-switch",
			IP:       "192.168.10.4",
			Platform: "Cisco",
			Accesslist: []models.AccessItem{
				{Protocol: "telnet", Port: "23"},
			},
		},
		{
			ID:       "h5",
			Hostname: "almalinux9-host",
			IP:       "192.168.10.5",
			Platform: "Linux",
			OS:       "AlmaLinux 9",
			Accesslist: []models.AccessItem{
				{Protocol: "ssh", Port: "22"},
			},
		},
	}
	if err := db.WriteHostList(testHosts); err != nil {
		t.Fatalf("failed to write test hosts: %v", err)
	}

	testCreds := []models.HostCredentials{
		{
			Hostname: "ssh-host-1",
			Userlist: []models.UserCredential{
				{Username: "user1", Password: "pwd1"},
				{Username: "user2", Password: "pwd2"},
			},
		},
		{
			Hostname: "ssh-host-custom-port",
			Userlist: []models.UserCredential{
				{Username: "admin", Password: "adminpwd"},
			},
		},
		{
			Hostname: "web-only-host",
			Userlist: []models.UserCredential{
				{Username: "webadmin", Password: "webpwd"},
			},
		},
		{
			Hostname: "telnet-switch",
			Userlist: []models.UserCredential{
				{Username: "cisco_admin", Password: "ciscopassword"},
			},
		},
		{
			Hostname: "almalinux9-host",
			Userlist: []models.UserCredential{
				{Username: "deploy", Password: "almadeploypass"},
			},
		},
	}
	if err := db.WriteHostCredentials(testCreds); err != nil {
		t.Fatalf("failed to write test creds: %v", err)
	}

	// 1. Test GET /api/ssh-fzf
	req := httptest.NewRequest(http.MethodGet, "/api/ssh-fzf", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := getSSHFzfTargetsHandler(c); err != nil {
		t.Fatalf("getSSHFzfTargetsHandler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	// Should include ssh-host-1 user1 (port 22) and user2 (port 22)
	if !strings.Contains(body, "ssh-host-1") || !strings.Contains(body, "user1") || !strings.Contains(body, "user2") {
		t.Errorf("expected ssh-host-1 users in response, got: %s", body)
	}
	// Should include ssh-host-custom-port admin with port 10022
	if !strings.Contains(body, "ssh-host-custom-port") || !strings.Contains(body, "10022") {
		t.Errorf("expected ssh-host-custom-port in response, got: %s", body)
	}
	// Must NOT include web-only-host
	if strings.Contains(body, "web-only-host") {
		t.Errorf("web-only-host should not be included in ssh targets: %s", body)
	}
	// Should include telnet-switch with telnet protocol
	if !strings.Contains(body, "telnet-switch") || !strings.Contains(body, "cisco_admin") || !strings.Contains(body, `"protocol":"telnet"`) {
		t.Errorf("expected telnet-switch with telnet protocol in response: %s", body)
	}
	// Should include almalinux9-host with AlmaLinux 9 OS
	if !strings.Contains(body, "almalinux9-host") || !strings.Contains(body, "AlmaLinux 9") {
		t.Errorf("expected almalinux9-host in response: %s", body)
	}

	// 2. Test POST /api/ssh-fzf without hostname (returns targets)
	postReq := httptest.NewRequest(http.MethodPost, "/api/ssh-fzf", strings.NewReader(`{"masterpassword":"password"}`))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	postCtx := e.NewContext(postReq, postRec)

	if err := sshFzfHandler(postCtx); err != nil {
		t.Fatalf("sshFzfHandler returned error: %v", err)
	}
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", postRec.Code)
	}
	if !strings.Contains(postRec.Body.String(), "ssh-host-1") {
		t.Errorf("expected targets in POST empty hostname response: %s", postRec.Body.String())
	}

	// 3. Test POST /api/ssh-fzf with hostname & username (returns password)
	pwdReq := httptest.NewRequest(http.MethodPost, "/api/ssh-fzf", strings.NewReader(`{"masterpassword":"password","hostname":"ssh-host-1","username":"user1"}`))
	pwdReq.Header.Set("Content-Type", "application/json")
	pwdRec := httptest.NewRecorder()
	pwdCtx := e.NewContext(pwdReq, pwdRec)

	if err := sshFzfHandler(pwdCtx); err != nil {
		t.Fatalf("sshFzfHandler returned error: %v", err)
	}
	if pwdRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", pwdRec.Code)
	}
	if !strings.Contains(pwdRec.Body.String(), "pwd1") {
		t.Errorf("expected pwd1 in response, got: %s", pwdRec.Body.String())
	}

	// 4. Test POST /api/ssh-fzf for telnet-switch password
	telnetPwdReq := httptest.NewRequest(http.MethodPost, "/api/ssh-fzf", strings.NewReader(`{"masterpassword":"password","hostname":"telnet-switch","username":"cisco_admin"}`))
	telnetPwdReq.Header.Set("Content-Type", "application/json")
	telnetPwdRec := httptest.NewRecorder()
	telnetPwdCtx := e.NewContext(telnetPwdReq, telnetPwdRec)

	if err := sshFzfHandler(telnetPwdCtx); err != nil {
		t.Fatalf("sshFzfHandler returned error: %v", err)
	}
	if telnetPwdRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", telnetPwdRec.Code)
	}
	if !strings.Contains(telnetPwdRec.Body.String(), "ciscopassword") {
		t.Errorf("expected ciscopassword in response, got: %s", telnetPwdRec.Body.String())
	}
}

func TestDownloadHcmClient(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/client/download", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := downloadHcmClientHandler(c); err != nil {
		t.Fatalf("downloadHcmClientHandler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/gzip") {
		t.Errorf("expected Content-Type application/gzip, got %s", contentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "hcm-client.tgz") {
		t.Errorf("expected Content-Disposition to have hcm-client.tgz, got %s", contentDisposition)
	}

	// Verify tar.gz contents
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	foundBinary := false
	foundRunScript := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar reading error: %v", err)
		}
		if strings.HasSuffix(hdr.Name, "hcm-client") {
			foundBinary = true
		}
		if strings.HasSuffix(hdr.Name, "run.sh") {
			foundRunScript = true
		}
	}

	if !foundBinary {
		t.Errorf("expected hcm-client binary in tar archive")
	}
	if !foundRunScript {
		t.Errorf("expected run.sh in tar archive")
	}
}

func TestRequireClientCertMiddleware(t *testing.T) {
	e := echo.New()

	// Generate a CA cert
	caPrivKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "TestCA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivKey.PublicKey, caPrivKey)
	caCert, _ := x509.ParseCertificate(caDER)

	// Generate Client Cert (Valid)
	clientPrivKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		Subject:      pkix.Name{CommonName: "client-valid"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
	}
	clientDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientPrivKey.PublicKey, caPrivKey)
	validCert, _ := x509.ParseCertificate(clientDER)

	// Generate Client Cert (Revoked)
	revokedTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(200),
		Subject:      pkix.Name{CommonName: "client-revoked"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
	}
	revokedDER, _ := x509.CreateCertificate(rand.Reader, revokedTemplate, caCert, &clientPrivKey.PublicKey, caPrivKey)
	revokedCert, _ := x509.ParseCertificate(revokedDER)

	// Create CRL with revoked serial 200
	crlTemplate := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now().Add(-1 * time.Minute),
		NextUpdate: time.Now().Add(1 * time.Hour),
		RevokedCertificateEntries: []x509.RevocationListEntry{
			{
				SerialNumber:   big.NewInt(200),
				RevocationTime: time.Now(),
			},
		},
	}
	crlDER, err := x509.CreateRevocationList(rand.Reader, crlTemplate, caCert, caPrivKey)
	if err != nil {
		t.Fatalf("failed to create test CRL: %v", err)
	}

	tmpDir := t.TempDir()
	crlPath := filepath.Join(tmpDir, "crl.pem")
	crlPEM := pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlDER})
	_ = os.WriteFile(crlPath, crlPEM, 0644)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}
	middleware := RequireClientCertMiddleware(crlPath)

	// 1. Request without TLS (c.Request().TLS == nil)
	reqNoTLS := httptest.NewRequest(http.MethodGet, "/api/ssh-fzf", nil)
	recNoTLS := httptest.NewRecorder()
	cNoTLS := e.NewContext(reqNoTLS, recNoTLS)
	_ = middleware(handler)(cNoTLS)
	if recNoTLS.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for request without TLS, got %d", recNoTLS.Code)
	}

	// 2. Request with TLS but without peer certificates
	reqNoCerts := httptest.NewRequest(http.MethodGet, "/api/ssh-fzf", nil)
	reqNoCerts.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{}}
	recNoCerts := httptest.NewRecorder()
	cNoCerts := e.NewContext(reqNoCerts, recNoCerts)
	_ = middleware(handler)(cNoCerts)
	if recNoCerts.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for request without client cert, got %d", recNoCerts.Code)
	}

	// 3. Request with valid client certificate
	reqValid := httptest.NewRequest(http.MethodGet, "/api/ssh-fzf", nil)
	reqValid.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{validCert}}
	recValid := httptest.NewRecorder()
	cValid := e.NewContext(reqValid, recValid)
	_ = middleware(handler)(cValid)
	if recValid.Code != http.StatusOK {
		t.Errorf("expected 200 for valid client cert, got %d: %s", recValid.Code, recValid.Body.String())
	}

	// 4. Request with revoked client certificate
	reqRevoked := httptest.NewRequest(http.MethodGet, "/api/ssh-fzf", nil)
	reqRevoked.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{revokedCert}}
	recRevoked := httptest.NewRecorder()
	cRevoked := e.NewContext(reqRevoked, recRevoked)
	_ = middleware(handler)(cRevoked)
	if recRevoked.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for revoked client cert, got %d", recRevoked.Code)
	}
	if !strings.Contains(recRevoked.Body.String(), "revoked") {
		t.Errorf("expected 'revoked' error message, got: %s", recRevoked.Body.String())
	}
}

func TestIsCertRevoked(t *testing.T) {
	// Test nil handling
	if IsCertRevoked("", nil) {
		t.Errorf("expected false for nil cert")
	}

	// Test non-existent file
	cert := &x509.Certificate{SerialNumber: big.NewInt(999)}
	if IsCertRevoked("non_existent_crl.pem", cert) {
		t.Errorf("expected false for non existent crl file")
	}
}

