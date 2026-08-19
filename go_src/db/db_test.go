package db

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"host-credential-manager-go/go_src/models"
)

func TestReadHostListFromCsv(t *testing.T) {
	csvData := `id,hostname,ip,platform,port,tags,description,updatedAt
1,server-a,192.168.1.10,Linux,22,web;prod,Primary web server,2026-01-01T00:00:00.000Z
2,server-b,192.168.1.11,Windows,3389,db;staging,Staging DB server,2026-01-02T00:00:00.000Z
`
	hosts, err := ReadHostListFromCsv(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error reading CSV: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	if hosts[0].ID != "1" || hosts[0].Hostname != "server-a" || hosts[0].IP != "192.168.1.10" || hosts[0].Platform != "Linux" || hosts[0].Port != "22" || hosts[0].Tags != "web;prod" || hosts[0].Description != "Primary web server" {
		t.Errorf("unexpected host[0] data: %+v", hosts[0])
	}

	if hosts[1].ID != "2" || hosts[1].Hostname != "server-b" || hosts[1].IP != "192.168.1.11" || hosts[1].Platform != "Windows" || hosts[1].Port != "3389" || hosts[1].Tags != "db;staging" || hosts[1].Description != "Staging DB server" {
		t.Errorf("unexpected host[1] data: %+v", hosts[1])
	}
}

func TestWriteHostListToCsv(t *testing.T) {
	hosts := []models.Host{
		{
			ID:          "100",
			Hostname:    "web01",
			IP:          "10.0.0.1",
			Platform:    "Linux",
			Port:        "22",
			Tags:        "web",
			Description: "Main web",
			UpdatedAt:   "2026-01-01T00:00:00.000Z",
		},
	}

	var buf bytes.Buffer
	if err := WriteHostListToCsv(&buf, hosts); err != nil {
		t.Fatalf("unexpected error writing CSV: %v", err)
	}

	parsed, err := ReadHostListFromCsv(&buf)
	if err != nil {
		t.Fatalf("unexpected error re-parsing CSV: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 host, got %d", len(parsed))
	}

	if parsed[0].ID != "100" || parsed[0].Hostname != "web01" || parsed[0].IP != "10.0.0.1" {
		t.Errorf("parsed host does not match: %+v", parsed[0])
	}
}

func TestReadWriteHostListToml(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := dataDir
	originalToml := tomlFilePath
	defer func() {
		dataDir = originalDataDir
		tomlFilePath = originalToml
	}()

	dataDir = tmpDir
	tomlFilePath = filepath.Join(tmpDir, "hostlist.toml")

	// Initially empty
	hosts, err := ReadHostList()
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected 0 hosts, got %d", len(hosts))
	}

	testHosts := []models.Host{
		{
			ID:          "h1",
			Hostname:    "host-one",
			IP:          "172.16.0.1",
			Platform:    "Ubuntu",
			Port:        "22",
			Tags:        "test",
			Description: "Test host",
			UpdatedAt:   "2026-01-01T12:00:00Z",
		},
	}

	if err := WriteHostList(testHosts); err != nil {
		t.Fatalf("failed to write host list: %v", err)
	}

	// Verify hostlist.toml was created and hostlist.csv was NOT created
	if _, err := os.Stat(tomlFilePath); os.IsNotExist(err) {
		t.Fatalf("expected hostlist.toml to exist")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "hostlist.csv")); !os.IsNotExist(err) {
		t.Fatalf("expected hostlist.csv NOT to exist")
	}

	readBack, err := ReadHostList()
	if err != nil {
		t.Fatalf("failed to read back host list: %v", err)
	}

	if len(readBack) != 1 {
		t.Fatalf("expected 1 host, got %d", len(readBack))
	}
	if readBack[0].Hostname != "host-one" || readBack[0].IP != "172.16.0.1" {
		t.Errorf("unexpected host data: %+v", readBack[0])
	}
}
