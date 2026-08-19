package db

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"host-credential-manager-go/go_src/models"
)

func TestReadHostListFromCsv_WithoutId(t *testing.T) {
	csvData := `hostname,ip,platform,port,tags,description,updatedAt
server-a,192.168.1.10,Linux,22,web;prod,Primary web server,2026-01-01T00:00:00.000Z
server-b,192.168.1.11,Windows,3389,db;staging,Staging DB server,2026-01-02T00:00:00.000Z
`
	hosts, err := ReadHostListFromCsv(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error reading CSV: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	if hosts[0].Hostname != "server-a" || hosts[0].IP != "192.168.1.10" || hosts[0].Platform != "Linux" || hosts[0].Port != "22" || hosts[0].Tags != "web;prod" || hosts[0].Description != "Primary web server" {
		t.Errorf("unexpected host[0] data: %+v", hosts[0])
	}

	if hosts[1].Hostname != "server-b" || hosts[1].IP != "192.168.1.11" || hosts[1].Platform != "Windows" || hosts[1].Port != "3389" || hosts[1].Tags != "db;staging" || hosts[1].Description != "Staging DB server" {
		t.Errorf("unexpected host[1] data: %+v", hosts[1])
	}
}

func TestReadHostListFromCsv_WithLegacyId(t *testing.T) {
	csvData := `id,hostname,ip,platform,port,tags,description,updatedAt
999,server-legacy,192.168.1.99,Linux,22,legacy,Legacy host,2026-01-01T00:00:00.000Z
`
	hosts, err := ReadHostListFromCsv(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error reading legacy CSV: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}

	if hosts[0].Hostname != "server-legacy" || hosts[0].IP != "192.168.1.99" {
		t.Errorf("unexpected host[0] data: %+v", hosts[0])
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

	csvStr := buf.String()
	if strings.Contains(csvStr, "id,") || strings.HasPrefix(csvStr, "id") {
		t.Errorf("CSV header must not contain id, got:\n%s", csvStr)
	}
	if strings.Contains(csvStr, "100") {
		t.Errorf("CSV rows must not contain id value 100, got:\n%s", csvStr)
	}

	parsed, err := ReadHostListFromCsv(strings.NewReader(csvStr))
	if err != nil {
		t.Fatalf("unexpected error re-parsing CSV: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 host, got %d", len(parsed))
	}

	if parsed[0].Hostname != "web01" || parsed[0].IP != "10.0.0.1" {
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
			ID:          "ignored_custom_id",
			Hostname:    "host-one",
			IP:          "172.16.0.1",
			Platform:    "Ubuntu",
			Port:        "22",
			Tags:        "test",
			Description: "Test host",
			UpdatedAt:   "2026-01-01T12:00:00Z",
		},
		{
			ID:          "ignored_custom_id_2",
			Hostname:    "host-two",
			IP:          "172.16.0.2",
			Platform:    "Debian",
			Port:        "22",
			Tags:        "test2",
			Description: "Test host 2",
			UpdatedAt:   "2026-01-01T12:00:00Z",
		},
	}

	if err := WriteHostList(testHosts); err != nil {
		t.Fatalf("failed to write host list: %v", err)
	}

	// Verify hostlist.toml was created and contains no id field
	tomlContent, err := os.ReadFile(tomlFilePath)
	if err != nil {
		t.Fatalf("failed to read toml file: %v", err)
	}
	if strings.Contains(string(tomlContent), "id =") || strings.Contains(string(tomlContent), "ignored_custom_id") {
		t.Errorf("hostlist.toml should not contain id field, got:\n%s", string(tomlContent))
	}

	// Read back and verify IDs assigned by database ("1", "2")
	readBack, err := ReadHostList()
	if err != nil {
		t.Fatalf("failed to read back host list: %v", err)
	}

	if len(readBack) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(readBack))
	}
	if readBack[0].ID != "1" || readBack[0].Hostname != "host-one" {
		t.Errorf("expected ID '1' for first host, got: %+v", readBack[0])
	}
	if readBack[1].ID != "2" || readBack[1].Hostname != "host-two" {
		t.Errorf("expected ID '2' for second host, got: %+v", readBack[1])
	}

	// Test backward compatibility: writing TOML with legacy id field
	legacyToml := `
[[host]]
id = '999'
hostname = 'legacy-host'
ip = '10.99.99.99'
platform = 'Linux'
port = '22'
tags = 'legacy'
description = 'Legacy host'
updatedAt = '2026-01-01T00:00:00Z'
`
	if err := os.WriteFile(tomlFilePath, []byte(legacyToml), 0644); err != nil {
		t.Fatalf("failed to write legacy toml: %v", err)
	}

	readLegacy, err := ReadHostList()
	if err != nil {
		t.Fatalf("expected no error reading legacy TOML, got: %v", err)
	}
	if len(readLegacy) != 1 {
		t.Fatalf("expected 1 host, got %d", len(readLegacy))
	}
	if readLegacy[0].ID != "1" || readLegacy[0].Hostname != "legacy-host" {
		t.Errorf("expected ID '1' assigned by database, got: %+v", readLegacy[0])
	}
}
