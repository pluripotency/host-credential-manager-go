package db

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	if hosts[0].Hostname != "server-a" || hosts[0].IP != "192.168.1.10" || hosts[0].Platform != "Linux" || len(hosts[0].Accesslist) != 1 || hosts[0].Accesslist[0].Port != "22" || hosts[0].Tags != "web;prod" || hosts[0].Description != "Primary web server" {
		t.Errorf("unexpected host[0] data: %+v", hosts[0])
	}

	if hosts[1].Hostname != "server-b" || hosts[1].IP != "192.168.1.11" || hosts[1].Platform != "Windows" || len(hosts[1].Accesslist) != 1 || hosts[1].Accesslist[0].Port != "3389" || hosts[1].Tags != "db;staging" || hosts[1].Description != "Staging DB server" {
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
			Tags:        "test",
			Description: "Test host",
			UpdatedAt:   "2026-01-01T12:00:00Z",
		},
		{
			ID:          "ignored_custom_id_2",
			Hostname:    "host-two",
			IP:          "172.16.0.2",
			Platform:    "Debian",
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

func TestReadWriteAccessListToml(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := dataDir
	originalToml := tomlFilePath
	defer func() {
		dataDir = originalDataDir
		tomlFilePath = originalToml
	}()

	dataDir = tmpDir
	tomlFilePath = filepath.Join(tmpDir, "hostlist.toml")

	testHosts := []models.Host{
		{
			Hostname: "web-server-test",
			IP:       "192.168.1.100",
			Platform: "Linux",
			Accesslist: []models.AccessItem{
				{Protocol: "http", Port: "8080", Path: "/app"},
				{Protocol: "https", Port: "8443", Path: "/admin"},
				{Protocol: "ssh", Port: "10022"},
			},
		},
	}

	if err := WriteHostList(testHosts); err != nil {
		t.Fatalf("failed to write host list: %v", err)
	}

	readBack, err := ReadHostList()
	if err != nil {
		t.Fatalf("failed to read host list: %v", err)
	}

	if len(readBack) != 1 {
		t.Fatalf("expected 1 host, got %d", len(readBack))
	}

	if len(readBack[0].Accesslist) != 3 {
		t.Fatalf("expected 3 access items, got %d", len(readBack[0].Accesslist))
	}

	if readBack[0].Accesslist[0].Protocol != "http" || readBack[0].Accesslist[0].Port != "8080" || readBack[0].Accesslist[0].Path != "/app" {
		t.Errorf("unexpected access item [0]: %+v", readBack[0].Accesslist[0])
	}
}

func TestTabConfig_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := dataDir
	originalToml := tomlFilePath
	originalCred := credTomlFilePath
	defer func() {
		dataDir = originalDataDir
		tomlFilePath = originalToml
		credTomlFilePath = originalCred
	}()

	dataDir = tmpDir
	tomlFilePath = filepath.Join(tmpDir, "hostlist.toml")
	credTomlFilePath = filepath.Join(tmpDir, "host_credentials.toml")

	// Create tab_config.toml
	tabConfigToml := `
[[tab]]
name = 'tab1'
dirpath = './tab1_dir'
list_filename = 'hostlist.toml'
cred_filename = 'hostcredentials.toml'

[[tab]]
name = 'tab2'
dirpath = './tab2_dir'
list_filename = 'hostlist.toml'
cred_filename = 'host_credentials.toml'
`
	if err := os.WriteFile(filepath.Join(tmpDir, "tab_config.toml"), []byte(tabConfigToml), 0644); err != nil {
		t.Fatalf("failed to write tab_config: %v", err)
	}

	tab1Dir := filepath.Join(tmpDir, "tab1_dir")
	tab2Dir := filepath.Join(tmpDir, "tab2_dir")
	_ = os.MkdirAll(tab1Dir, 0755)
	_ = os.MkdirAll(tab2Dir, 0755)

	tab1HostsToml := `
[[host]]
hostname = 'tab1-srv'
ip = '10.1.1.1'
platform = 'Linux'
[[host.accesslist]]
protocol = 'ssh'
port = '22'
`
	tab1CredsToml := `
[[host]]
hostname = 'tab1-srv'
[[host.userlist]]
username = 'admin1'
password = 'pass1'
`
	tab2HostsToml := `
[[host]]
hostname = 'tab2-srv'
ip = '10.2.2.2'
platform = 'Windows'
[[host.accesslist]]
protocol = 'rdp'
port = '3389'
`
	tab2CredsToml := `
[[host]]
hostname = 'tab2-srv'
[[host.userlist]]
username = 'admin2'
password = 'pass2'
`
	_ = os.WriteFile(filepath.Join(tab1Dir, "hostlist.toml"), []byte(tab1HostsToml), 0644)
	_ = os.WriteFile(filepath.Join(tab1Dir, "hostcredentials.toml"), []byte(tab1CredsToml), 0644)
	_ = os.WriteFile(filepath.Join(tab2Dir, "hostlist.toml"), []byte(tab2HostsToml), 0644)
	_ = os.WriteFile(filepath.Join(tab2Dir, "host_credentials.toml"), []byte(tab2CredsToml), 0644)

	// Test ReadTabConfig
	tabs, err := ReadTabConfig()
	if err != nil {
		t.Fatalf("unexpected error reading tab config: %v", err)
	}
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tabs))
	}
	if tabs[0].Name != "tab1" || tabs[1].Name != "tab2" {
		t.Errorf("unexpected tabs: %+v", tabs)
	}

	// Test ReadHostList with tabs
	hosts, err := ReadHostList()
	if err != nil {
		t.Fatalf("unexpected error reading host list with tabs: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hosts[0].Tab != "tab1" || hosts[0].Hostname != "tab1-srv" {
		t.Errorf("unexpected host 0: %+v", hosts[0])
	}
	if hosts[1].Tab != "tab2" || hosts[1].Hostname != "tab2-srv" {
		t.Errorf("unexpected host 1: %+v", hosts[1])
	}

	// Test ReadHostCredentials with tabs
	creds, err := ReadHostCredentials()
	if err != nil {
		t.Fatalf("unexpected error reading creds with tabs: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 creds, got %d", len(creds))
	}

	// Test WriteHostList with tabs
	hosts[0].Description = "Updated tab1 description"
	if err := WriteHostList(hosts); err != nil {
		t.Fatalf("failed to write host list: %v", err)
	}

	reReadHosts, err := ReadHostList()
	if err != nil {
		t.Fatalf("failed to re-read host list: %v", err)
	}
	if len(reReadHosts) != 2 || reReadHosts[0].Description != "Updated tab1 description" {
		t.Errorf("re-read host not updated: %+v", reReadHosts[0])
	}
}

func TestTabArchive_ExportImport(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataDir := GetDataDir()
	defer func() {
		SetDataDir(originalDataDir)
	}()

	SetDataDir(tmpDir)

	// Create tab with custom list_filename and cred_filename to trigger meta.md creation
	tabName := "custom_tab"
	tabDir := filepath.Join(tmpDir, tabName)
	_ = os.MkdirAll(tabDir, 0755)

	customListFile := "custom_hosts.toml"
	customCredFile := "custom_creds.toml"

	_ = os.WriteFile(filepath.Join(tabDir, customListFile), []byte("[[host]]\nhostname = 'host-cust'\nip = '192.168.10.1'\nplatform = 'Linux'\n"), 0644)
	_ = os.WriteFile(filepath.Join(tabDir, customCredFile), []byte("[[host]]\nhostname = 'host-cust'\n[[host.userlist]]\nusername = 'u1'\npassword = 'p1'\n"), 0644)

	initialTabs := []models.TabItem{
		{
			Name:         tabName,
			DirPath:      "./" + tabName,
			ListFilename: customListFile,
			CredFilename: customCredFile,
		},
	}
	if err := WriteTabConfig(initialTabs); err != nil {
		t.Fatalf("failed to write tab config: %v", err)
	}

	// 1. Export tab
	var buf bytes.Buffer
	if err := ExportTabArchive(tabName, &buf); err != nil {
		t.Fatalf("ExportTabArchive failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected non-empty archive")
	}

	// Verify meta.md was created in the source tab dir
	metaBytes, err := os.ReadFile(filepath.Join(tabDir, "meta.md"))
	if err != nil {
		t.Fatalf("expected meta.md to be created: %v", err)
	}
	metaStr := string(metaBytes)
	if !strings.Contains(metaStr, customListFile) || !strings.Contains(metaStr, customCredFile) {
		t.Errorf("meta.md missing parameters: %s", metaStr)
	}

	// 2. Test Import with conflict (already exists, overwrite = false)
	_, err = ImportTabArchive(bytes.NewReader(buf.Bytes()), tabName, false)
	if err == nil {
		t.Fatalf("expected ErrTabConflict when overwrite is false and tab exists")
	}
	conflictErr, ok := err.(*ErrTabConflict)
	if !ok || conflictErr.Name != tabName {
		t.Fatalf("expected *ErrTabConflict for '%s', got: %T: %v", tabName, err, err)
	}

	// 3. Test Import with overwrite = true
	importedTab, err := ImportTabArchive(bytes.NewReader(buf.Bytes()), tabName, true)
	if err != nil {
		t.Fatalf("ImportTabArchive with overwrite failed: %v", err)
	}
	if importedTab.Name != tabName {
		t.Errorf("expected tab name '%s', got '%s'", tabName, importedTab.Name)
	}
	if importedTab.DirPath != "./"+tabName {
		t.Errorf("expected dirpath './%s', got '%s'", tabName, importedTab.DirPath)
	}
	if importedTab.ListFilename != customListFile {
		t.Errorf("expected list filename '%s', got '%s'", customListFile, importedTab.ListFilename)
	}
	if importedTab.CredFilename != customCredFile {
		t.Errorf("expected cred filename '%s', got '%s'", customCredFile, importedTab.CredFilename)
	}

	// Verify ReadHostList returns the host from the imported tab
	hosts, err := ReadHostList()
	if err != nil {
		t.Fatalf("ReadHostList failed: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Hostname != "host-cust" || hosts[0].Tab != tabName {
		t.Errorf("unexpected hosts after import: %+v", hosts)
	}
}

func createTestTarGz(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		data := []byte(content)
		hdr := &tar.Header{
			Name:    name,
			Mode:    0644,
			Size:    int64(len(data)),
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("Write content failed: %v", err)
		}
	}

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func TestTabArchive_ValidationAndLoadability(t *testing.T) {
	tempDir := t.TempDir()
	dataDir = tempDir
	tabConfigPath = filepath.Join(tempDir, "tab_config.toml")
	tomlFilePath = filepath.Join(tempDir, "hostlist.toml")
	credTomlFilePath = filepath.Join(tempDir, "host_credentials.toml")

	stagingDir := filepath.Join(tempDir, "tmp_hcm")
	SetTmpHcmDir(stagingDir)

	// Setup tab_config.toml with existing tab "myexistingtab"
	existingTab := models.TabItem{
		Name:         "myexistingtab",
		DirPath:      "./myexistingtab",
		ListFilename: "hostlist.toml",
		CredFilename: "host_credentials.toml",
	}
	if err := WriteTabConfig([]models.TabItem{existingTab}); err != nil {
		t.Fatalf("WriteTabConfig failed: %v", err)
	}

	// Case 1: Missing credentials file -> should fail with error, NOT return conflict
	archiveMissingCred := createTestTarGz(t, map[string]string{
		"myexistingtab/hostlist.toml": "[[host]]\nhostname = 'h1'\n",
	})
	_, err := ImportTabArchive(bytes.NewReader(archiveMissingCred), "myexistingtab", false)
	if err == nil || strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected error for missing credentials, got: %v", err)
	}

	// Case 2: Broken/unloadable TOML in hostlist -> should fail with error, NOT return conflict
	archiveBrokenToml := createTestTarGz(t, map[string]string{
		"myexistingtab/hostlist.toml":         "[[host\nINVALID TOML [[{",
		"myexistingtab/host_credentials.toml": "[[host]]\nhostname = 'h1'\n",
	})
	_, err = ImportTabArchive(bytes.NewReader(archiveBrokenToml), "myexistingtab", false)
	if err == nil || strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected error for broken TOML in hostlist, got: %v", err)
	}

	// Case 3: Valid loadable TOML, tab in tab_config.toml, overwrite=false -> MUST return ErrTabConflict
	archiveValid := createTestTarGz(t, map[string]string{
		"myexistingtab/hostlist.toml":         "[[host]]\nhostname = 'h1'\nip = '1.1.1.1'\nplatform = 'Linux'\n",
		"myexistingtab/host_credentials.toml": "[[host]]\nhostname = 'h1'\n[[host.userlist]]\nusername = 'admin'\npassword = 'secret'\n",
	})
	_, err = ImportTabArchive(bytes.NewReader(archiveValid), "myexistingtab", false)
	if err == nil {
		t.Fatalf("expected ErrTabConflict, got nil")
	}
	conflict, ok := err.(*ErrTabConflict)
	if !ok {
		t.Fatalf("expected *ErrTabConflict, got %T: %v", err, err)
	}
	if conflict.Name != "myexistingtab" {
		t.Errorf("expected conflict name 'myexistingtab', got '%s'", conflict.Name)
	}

	// Verify files were extracted to staging directory
	extractedList := filepath.Join(stagingDir, "myexistingtab", "hostlist.toml")
	if _, err := os.Stat(extractedList); err != nil {
		t.Errorf("staging file not found at %s: %v", extractedList, err)
	}

	// Case 4: Valid loadable TOML, tab in tab_config.toml, overwrite=true -> MUST succeed
	tabItem, err := ImportTabArchive(bytes.NewReader(archiveValid), "myexistingtab", true)
	if err != nil {
		t.Fatalf("expected success with overwrite=true, got: %v", err)
	}
	if tabItem.Name != "myexistingtab" {
		t.Errorf("expected tab name 'myexistingtab', got '%s'", tabItem.Name)
	}

	// Case 5: Valid archive with meta.toml specifying custom files
	archiveMeta := createTestTarGz(t, map[string]string{
		"newtab/meta.toml": "list_filename = 'my_hosts.toml'\ncred_filename = 'my_creds.toml'\n",
		"newtab/my_hosts.toml": "[[host]]\nhostname = 'h2'\nip = '2.2.2.2'\nplatform = 'Linux'\n",
		"newtab/my_creds.toml": "[[host]]\nhostname = 'h2'\n[[host.userlist]]\nusername = 'u2'\npassword = 'p2'\n",
	})
	// "newtab" is not in tab_config.toml -> MUST succeed without conflict
	importedNewTab, err := ImportTabArchive(bytes.NewReader(archiveMeta), "newtab", false)
	if err != nil {
		t.Fatalf("expected success for new tab, got: %v", err)
	}
	if importedNewTab.Name != "newtab" {
		t.Errorf("expected tab name 'newtab', got '%s'", importedNewTab.Name)
	}
	if importedNewTab.ListFilename != "my_hosts.toml" {
		t.Errorf("expected list filename 'my_hosts.toml', got '%s'", importedNewTab.ListFilename)
	}
	if importedNewTab.CredFilename != "my_creds.toml" {
		t.Errorf("expected cred filename 'my_creds.toml', got '%s'", importedNewTab.CredFilename)
	}
}

