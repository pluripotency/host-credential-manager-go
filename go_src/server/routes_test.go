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
