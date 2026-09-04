package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}

