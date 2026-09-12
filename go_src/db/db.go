package db

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"host-credential-manager-go/go_src/models"

	"github.com/pelletier/go-toml/v2"
)

// DefaultProtocol returns a standard protocol for a given platform and port
func DefaultProtocol(platform, port string) string {
	if port == "23" {
		return "telnet"
	}
	p := strings.ToLower(platform)
	switch p {
	case "linux", "macos", "freebsd", "cisco", "router", "switch":
		return "ssh"
	case "windows":
		return "rdp"
	case "mysql":
		return "mysql"
	case "postgresql", "postgres":
		return "postgres"
	case "redis":
		return "redis"
	case "mongodb":
		return "mongodb"
	case "oracle":
		return "oracle"
	default:
		if port == "443" || port == "8443" || port == "8006" || port == "6443" {
			return "https"
		}
		if port == "80" || port == "8080" || port == "3000" {
			return "http"
		}
		if port == "22" {
			return "ssh"
		}
		if port == "3389" {
			return "rdp"
		}
		return "http"
	}
}

var (
	dataDir          = "./config"
	tabConfigPath    = filepath.Join(dataDir, "tab_config.toml")
	tomlFilePath     = filepath.Join(dataDir, "hostlist.toml")
	credTomlFilePath = filepath.Join(dataDir, "host_credentials.toml")
	configFilePath   = filepath.Join(dataDir, "config.toml")
	tmpHcmDir        = "/tmp/hcm"

	dbMutex sync.RWMutex
)

// SetTmpHcmDir allows overriding the /tmp/hcm staging directory for tests
func SetTmpHcmDir(dir string) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	tmpHcmDir = dir
}

// GetTmpHcmDir returns the staging directory for tab imports
func GetTmpHcmDir() string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return tmpHcmDir
}

// SetDataDir allows overriding the configuration directory (e.g. for testing)
func SetDataDir(dir string) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	dataDir = dir
	tabConfigPath = filepath.Join(dataDir, "tab_config.toml")
	tomlFilePath = filepath.Join(dataDir, "hostlist.toml")
	credTomlFilePath = filepath.Join(dataDir, "host_credentials.toml")
	configFilePath = filepath.Join(dataDir, "config.toml")
}

// GetDataDir returns the current configuration directory
func GetDataDir() string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return dataDir
}

// ReadTabConfig loads tab configurations from tab_config.toml if present
func ReadTabConfig() ([]models.TabItem, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	return readTabConfigLocked()
}

func readTabConfigLocked() ([]models.TabItem, error) {
	tabPath := filepath.Join(dataDir, "tab_config.toml")
	if _, err := os.Stat(tabPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(tabPath)
	if err != nil {
		return nil, err
	}

	var cfg models.TabConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return cfg.Tab, nil
}

// WriteTabConfig writes tab configurations to tab_config.toml
func WriteTabConfig(tabs []models.TabItem) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	return writeTabConfigLocked(tabs)
}

func writeTabConfigLocked(tabs []models.TabItem) error {
	tabPath := filepath.Join(dataDir, "tab_config.toml")
	cfg := models.TabConfig{Tab: tabs}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal tab config: %w", err)
	}
	return os.WriteFile(tabPath, data, 0644)
}

func resolveTabDir(tab models.TabItem) string {
	if filepath.IsAbs(tab.DirPath) {
		return tab.DirPath
	}
	return filepath.Join(dataDir, tab.DirPath)
}

func resolveTabListPath(tab models.TabItem) string {
	dir := resolveTabDir(tab)
	listFile := tab.ListFilename
	if listFile == "" {
		listFile = "hostlist.toml"
	}
	return filepath.Join(dir, listFile)
}

func resolveTabCredPath(tab models.TabItem) string {
	dir := resolveTabDir(tab)
	credFile := tab.CredFilename
	if credFile == "" {
		credFile = "host_credentials.toml"
	}
	primary := filepath.Join(dir, credFile)
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	// Fallback between hostcredentials.toml and host_credentials.toml
	if strings.Contains(credFile, "_") {
		alt := filepath.Join(dir, strings.ReplaceAll(credFile, "_", ""))
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	} else {
		alt := filepath.Join(dir, strings.Replace(credFile, "hostcredentials", "host_credentials", 1))
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return primary
}

// ErrTabConflict indicates that a tab's directory or name already exists
type ErrTabConflict struct {
	Name    string
	DirPath string
}

func (e *ErrTabConflict) Error() string {
	return fmt.Sprintf("tab directory '%s' already exists", e.DirPath)
}

// ExportTabArchive exports a tab's [dirpath]/* directory as a .tgz archive.
// If list_filename != "hostlist.toml" or cred_filename != "hostcredentials.toml",
// it generates [dirpath]/meta.md before packaging.
func ExportTabArchive(tabName string, w io.Writer) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return fmt.Errorf("failed to read tab config: %w", err)
	}

	var targetTab *models.TabItem
	for _, t := range tabs {
		if t.Name == tabName {
			tabCopy := t
			targetTab = &tabCopy
			break
		}
	}

	if targetTab == nil {
		return fmt.Errorf("tab '%s' not found", tabName)
	}

	tabDir := resolveTabDir(*targetTab)
	if _, err := os.Stat(tabDir); err != nil {
		return fmt.Errorf("tab directory '%s' does not exist: %w", tabDir, err)
	}

	// Determine effective list_filename and cred_filename
	listFilename := targetTab.ListFilename
	if listFilename == "" {
		listFilename = "hostlist.toml"
	}
	credFilename := targetTab.CredFilename
	if credFilename == "" {
		credFilename = "hostcredentials.toml"
	}
	// Check on-disk actual cred filename if set to default
	if credFilename == "hostcredentials.toml" {
		if _, err := os.Stat(filepath.Join(tabDir, "hostcredentials.toml")); os.IsNotExist(err) {
			if _, err2 := os.Stat(filepath.Join(tabDir, "host_credentials.toml")); err2 == nil {
				credFilename = "host_credentials.toml"
			}
		}
	}

	// If list_filename != hostlist.toml or cred_filename != hostcredentials.toml, create [dirpath]/meta.toml & [dirpath]/meta.md
	if listFilename != "hostlist.toml" || credFilename != "hostcredentials.toml" {
		metaTomlPath := filepath.Join(tabDir, "meta.toml")
		metaTomlContent := fmt.Sprintf("list_filename = %q\ncred_filename = %q\n", listFilename, credFilename)
		_ = os.WriteFile(metaTomlPath, []byte(metaTomlContent), 0644)

		metaMdPath := filepath.Join(tabDir, "meta.md")
		metaMdContent := fmt.Sprintf("# Tab Metadata\nlist_filename: %s\ncred_filename: %s\n", listFilename, credFilename)
		_ = os.WriteFile(metaMdPath, []byte(metaMdContent), 0644)
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write root directory entry for the tab name
	dirHdr := &tar.Header{
		Name:     targetTab.Name + "/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
		ModTime:  time.Now(),
	}
	if err := tw.WriteHeader(dirHdr); err != nil {
		return fmt.Errorf("failed to write directory header to tar: %w", err)
	}

	err = filepath.Walk(tabDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(tabDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		entryName := filepath.ToSlash(filepath.Join(targetTab.Name, relPath))
		if info.IsDir() {
			entryName += "/"
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		hdr.Name = entryName

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("failed to write file content to tar: %w", err)
			}
		}
		return nil
	})

	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func findCredFilename(dir string) string {
	candidates := []string{
		"host_credentials.toml",
		"hostcredentials.toml",
		"host_credentails.toml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(dir, c)); err == nil {
			return c
		}
	}
	return ""
}

func parseMetaTomlOrMd(dir string) (string, string) {
	// 1. Check meta.toml
	tomlPath := filepath.Join(dir, "meta.toml")
	if data, err := os.ReadFile(tomlPath); err == nil {
		var meta struct {
			ListFilename string `toml:"list_filename"`
			CredFilename string `toml:"cred_filename"`
		}
		if err := toml.Unmarshal(data, &meta); err == nil && (meta.ListFilename != "" || meta.CredFilename != "") {
			return meta.ListFilename, meta.CredFilename
		}
		var tabWrapper struct {
			Tab struct {
				ListFilename string `toml:"list_filename"`
				CredFilename string `toml:"cred_filename"`
			} `toml:"tab"`
		}
		if err := toml.Unmarshal(data, &tabWrapper); err == nil && (tabWrapper.Tab.ListFilename != "" || tabWrapper.Tab.CredFilename != "") {
			return tabWrapper.Tab.ListFilename, tabWrapper.Tab.CredFilename
		}
		l, c := parseMetaMd(string(data))
		if l != "" || c != "" {
			return l, c
		}
	}

	// 2. Check meta.md
	mdPath := filepath.Join(dir, "meta.md")
	if data, err := os.ReadFile(mdPath); err == nil {
		return parseMetaMd(string(data))
	}

	return "", ""
}

func parseMetaMd(content string) (string, string) {
	var listFilename, credFilename string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		var key, val string
		if idx := strings.Index(line, ":"); idx != -1 {
			key = strings.TrimSpace(line[:idx])
			val = strings.TrimSpace(line[idx+1:])
		} else if idx := strings.Index(line, "="); idx != -1 {
			key = strings.TrimSpace(line[:idx])
			val = strings.TrimSpace(line[idx+1:])
		}
		key = strings.ToLower(key)
		val = strings.Trim(val, "\"'")
		if key == "list_filename" {
			listFilename = val
		} else if key == "cred_filename" {
			credFilename = val
		}
	}
	return listFilename, credFilename
}

// ImportTabArchive extracts the archive into /tmp/hcm, validates that hostlist.toml and credentials
// (or files specified in meta.toml/meta.md) exist and are loadable.
// If the extracted directory name is present in tab_config.toml's tab.name and overwrite is false,
// returns *ErrTabConflict to prompt the user for confirmation.
func ImportTabArchive(r io.Reader, fallbackName string, overwrite bool) (*models.TabItem, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive data: %w", err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	type tarEntry struct {
		name    string
		isDir   bool
		mode    int64
		modTime time.Time
		content []byte
	}
	var entries []tarEntry

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar entry: %w", err)
		}

		cleanName := filepath.ToSlash(filepath.Clean(hdr.Name))
		if cleanName == "." || cleanName == "/" {
			continue
		}
		if strings.HasPrefix(cleanName, "../") || strings.HasPrefix(cleanName, "/") {
			return nil, fmt.Errorf("insecure path in archive: %s", hdr.Name)
		}

		entry := tarEntry{
			name:    cleanName,
			isDir:   hdr.FileInfo().IsDir(),
			mode:    hdr.Mode,
			modTime: hdr.ModTime,
		}
		if !entry.isDir {
			content, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("failed to read file content for %s: %w", hdr.Name, err)
			}
			entry.content = content
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("archive contains no files")
	}

	// 1. Determine extracted directory name
	var detectedFolder string
	hasConsistentRoot := true
	for _, entry := range entries {
		parts := strings.Split(strings.TrimPrefix(entry.name, "/"), "/")
		if len(parts) > 0 && parts[0] != "" {
			if detectedFolder == "" {
				detectedFolder = parts[0]
			} else if detectedFolder != parts[0] {
				hasConsistentRoot = false
				break
			}
		}
	}

	var extractedDirName string
	if hasConsistentRoot && detectedFolder != "" {
		extractedDirName = detectedFolder
	} else {
		extractedDirName = fallbackName
	}
	extractedDirName = strings.TrimSuffix(extractedDirName, ".tgz")
	extractedDirName = strings.TrimSuffix(extractedDirName, ".tar.gz")
	extractedDirName = strings.TrimSuffix(extractedDirName, ".tar")
	extractedDirName = filepath.Base(extractedDirName)
	extractedDirName = strings.TrimSpace(extractedDirName)
	if extractedDirName == "" || extractedDirName == "." {
		extractedDirName = "imported_tab"
	}

	// 2. Requirement:
	// "ない場合は作成した/tmp/hcmにimportファイルを展開し"
	tmpHcmBase := GetTmpHcmDir()
	if err := os.MkdirAll(tmpHcmBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging directory %s: %w", tmpHcmBase, err)
	}

	extractedDir := filepath.Join(tmpHcmBase, extractedDirName)
	_ = os.RemoveAll(extractedDir)
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extracted directory %s: %w", extractedDir, err)
	}

	for _, entry := range entries {
		rel := entry.name
		if hasConsistentRoot && detectedFolder != "" {
			if rel == detectedFolder || rel == detectedFolder+"/" {
				continue
			}
			rel = strings.TrimPrefix(rel, detectedFolder+"/")
		}
		rel = filepath.Clean(rel)
		if rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}

		destPath := filepath.Join(extractedDir, rel)
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(extractedDir)) {
			continue
		}

		if entry.isDir {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create parent directory for %s: %w", destPath, err)
			}
			if err := os.WriteFile(destPath, entry.content, 0644); err != nil {
				return nil, fmt.Errorf("failed to write file %s: %w", destPath, err)
			}
		}
	}

	// 3. Requirement:
	// "展開したディレクトリ直下にhostlist.toml, host_credentails.tomlがある、またはmeta.tomlで指定したファイルがあってロードが可能な場合のみ"
	listFilename := "hostlist.toml"
	credFilename := ""

	metaList, metaCred := parseMetaTomlOrMd(extractedDir)
	if metaList != "" {
		listFilename = filepath.Clean(strings.TrimSpace(metaList))
	}
	if metaCred != "" {
		credFilename = filepath.Clean(strings.TrimSpace(metaCred))
	} else {
		credFilename = findCredFilename(extractedDir)
	}

	listPath := filepath.Join(extractedDir, listFilename)
	if _, err := os.Stat(listPath); err != nil {
		return nil, fmt.Errorf("hostlist file '%s' does not exist in extracted tab", listFilename)
	}

	if credFilename == "" {
		return nil, fmt.Errorf("credentials file (host_credentials.toml / hostcredentials.toml / host_credentails.toml) does not exist in extracted tab")
	}
	credPath := filepath.Join(extractedDir, credFilename)
	if _, err := os.Stat(credPath); err != nil {
		return nil, fmt.Errorf("credentials file '%s' does not exist in extracted tab", credFilename)
	}

	// Verify loadability
	listData, err := os.ReadFile(listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read hostlist file '%s': %w", listFilename, err)
	}
	var hostListWrapper struct {
		Host []models.Host `toml:"host"`
	}
	if err := toml.Unmarshal(listData, &hostListWrapper); err != nil {
		var rawMap map[string]interface{}
		if err2 := toml.Unmarshal(listData, &rawMap); err2 != nil {
			return nil, fmt.Errorf("hostlist file '%s' is not loadable TOML: %w", listFilename, err2)
		}
	}

	credData, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file '%s': %w", credFilename, err)
	}
	var credListWrapper struct {
		Host []models.HostCredentials `toml:"host"`
	}
	if err := toml.Unmarshal(credData, &credListWrapper); err != nil {
		var rawMap map[string]interface{}
		if err2 := toml.Unmarshal(credData, &rawMap); err2 != nil {
			return nil, fmt.Errorf("credentials file '%s' is not loadable TOML: %w", credFilename, err2)
		}
	}

	// 4. Requirement:
	// "展開したディレクトリ名がtab_config.tomlのtab.nameにある場合は確認ダイアログを表示して。"
	dbMutex.Lock()
	defer dbMutex.Unlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return nil, fmt.Errorf("failed to read tab config: %w", err)
	}

	nameInTabConfig := false
	for _, t := range tabs {
		if strings.TrimSpace(t.Name) == strings.TrimSpace(extractedDirName) {
			nameInTabConfig = true
			break
		}
	}

	if nameInTabConfig && !overwrite {
		return nil, &ErrTabConflict{
			Name:    extractedDirName,
			DirPath: "./" + extractedDirName,
		}
	}

	// 5. Copy to destination: ./config/<extractedDirName>
	destDir := filepath.Join(dataDir, extractedDirName)
	if nameInTabConfig && overwrite {
		_ = os.RemoveAll(destDir)
	}
	if err := copyDir(extractedDir, destDir); err != nil {
		return nil, fmt.Errorf("failed to copy files from %s to %s: %w", extractedDir, destDir, err)
	}

	// 6. Update tab_config.toml
	found := false
	for i := range tabs {
		if strings.TrimSpace(tabs[i].Name) == strings.TrimSpace(extractedDirName) {
			tabs[i].Name = extractedDirName
			tabs[i].DirPath = "./" + extractedDirName
			tabs[i].ListFilename = listFilename
			tabs[i].CredFilename = credFilename
			found = true
			break
		}
	}
	if !found {
		tabs = append(tabs, models.TabItem{
			Name:         extractedDirName,
			DirPath:      "./" + extractedDirName,
			ListFilename: listFilename,
			CredFilename: credFilename,
		})
	}

	if err := writeTabConfigLocked(tabs); err != nil {
		return nil, fmt.Errorf("failed to update tab config: %w", err)
	}

	importedTab := models.TabItem{
		Name:         extractedDirName,
		DirPath:      "./" + extractedDirName,
		ListFilename: listFilename,
		CredFilename: credFilename,
	}

	return &importedTab, nil
}

func InitDatabase() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	return nil
}

// ReadConfig loads config from config.toml
func ReadConfig() (models.Config, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	var conf models.Config
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		conf.PermitIPList = []string{"127.0.0.1"}
		conf.MasterPassword = "password"
		return conf, nil
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return conf, err
	}

	if err := toml.Unmarshal(data, &conf); err != nil {
		return conf, err
	}

	if conf.MasterPassword == "" {
		conf.MasterPassword = "password"
	}
	if conf.AdminPassword == "" {
		conf.AdminPassword = "admin"
	}
	if conf.UserPassword == "" {
		conf.UserPassword = "user"
	}

	return conf, nil
}

// ReadHostList loads hosts from the TOML file and assigns database IDs
func ReadHostList() ([]models.Host, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return nil, err
	}

	if len(tabs) > 0 {
		var allHosts []models.Host
		globalID := 1
		for _, tab := range tabs {
			listPath := resolveTabListPath(tab)
			if _, err := os.Stat(listPath); os.IsNotExist(err) {
				continue
			}

			data, err := os.ReadFile(listPath)
			if err != nil {
				return nil, err
			}

			var hostList models.HostList
			if err := toml.Unmarshal(data, &hostList); err != nil {
				return nil, err
			}

			for i := range hostList.Host {
				hostList.Host[i].ID = strconv.Itoa(globalID)
				globalID++
				hostList.Host[i].Tab = tab.Name
				allHosts = append(allHosts, hostList.Host[i])
			}
		}
		if allHosts == nil {
			return []models.Host{}, nil
		}
		return allHosts, nil
	}

	if _, err := os.Stat(tomlFilePath); os.IsNotExist(err) {
		return []models.Host{}, nil
	}

	data, err := os.ReadFile(tomlFilePath)
	if err != nil {
		return nil, err
	}

	var hostList models.HostList
	if err := toml.Unmarshal(data, &hostList); err != nil {
		return nil, err
	}

	if hostList.Host == nil {
		return []models.Host{}, nil
	}

	for i := range hostList.Host {
		hostList.Host[i].ID = strconv.Itoa(i + 1)
	}

	return hostList.Host, nil
}

// ReadHostListFromCsv loads hosts from a CSV reader
func ReadHostListFromCsv(r io.Reader) ([]models.Host, error) {
	reader := csv.NewReader(r)
	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return []models.Host{}, nil
		}
		return nil, err
	}

	// Create column index map
	colMap := make(map[string]int)
	for i, name := range header {
		colMap[name] = i
	}

	var hosts []models.Host
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		getVal := func(key string) string {
			if idx, ok := colMap[key]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		var accesslist []models.AccessItem
		accesslistVal := getVal("accesslist")
		if accesslistVal != "" {
			_ = json.Unmarshal([]byte(accesslistVal), &accesslist)
		}

		portVal := getVal("port")
		platformVal := getVal("platform")
		if len(accesslist) == 0 && portVal != "" {
			proto := DefaultProtocol(platformVal, portVal)
			accesslist = []models.AccessItem{
				{Protocol: proto, Port: portVal},
			}
		}

		host := models.Host{
			ID:          getVal("id"),
			Hostname:    getVal("hostname"),
			IP:          getVal("ip"),
			Platform:    platformVal,
			OS:          getVal("os"),
			Tags:        getVal("tags"),
			Description: getVal("description"),
			UpdatedAt:   getVal("updatedAt"),
			Accesslist:  accesslist,
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// WriteHostList saves hosts to the TOML file
func WriteHostList(hosts []models.Host) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return err
	}

	if len(tabs) > 0 {
		tabHostsMap := make(map[string][]models.Host)
		for _, tab := range tabs {
			tabHostsMap[tab.Name] = []models.Host{}
		}

		defaultTab := tabs[0].Name
		for _, h := range hosts {
			t := h.Tab
			if t == "" || tabHostsMap[t] == nil {
				t = defaultTab
			}
			tabHostsMap[t] = append(tabHostsMap[t], h)
		}

		for _, tab := range tabs {
			listPath := resolveTabListPath(tab)
			if err := writeHostListTOML(listPath, tabHostsMap[tab.Name]); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeHostListTOML(tomlFilePath, hosts); err != nil {
		return err
	}

	return nil
}

// ReadHostCredentials loads credentials from host_credentials.toml
func ReadHostCredentials() ([]models.HostCredentials, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return nil, err
	}

	if len(tabs) > 0 {
		var allCreds []models.HostCredentials
		seenMap := make(map[string]int)
		for _, tab := range tabs {
			credPath := resolveTabCredPath(tab)
			if _, err := os.Stat(credPath); os.IsNotExist(err) {
				continue
			}

			data, err := os.ReadFile(credPath)
			if err != nil {
				return nil, err
			}

			var credList models.HostCredentialsList
			if err := toml.Unmarshal(data, &credList); err != nil {
				return nil, err
			}

			for _, cr := range credList.Host {
				cr.Tab = tab.Name
				if idx, exists := seenMap[cr.Hostname]; exists {
					allCreds[idx].Userlist = append(allCreds[idx].Userlist, cr.Userlist...)
				} else {
					seenMap[cr.Hostname] = len(allCreds)
					allCreds = append(allCreds, cr)
				}
			}
		}
		if allCreds == nil {
			return []models.HostCredentials{}, nil
		}
		return allCreds, nil
	}

	if _, err := os.Stat(credTomlFilePath); os.IsNotExist(err) {
		return []models.HostCredentials{}, nil
	}

	data, err := os.ReadFile(credTomlFilePath)
	if err != nil {
		return nil, err
	}

	var credList models.HostCredentialsList
	if err := toml.Unmarshal(data, &credList); err != nil {
		return nil, err
	}

	if credList.Host == nil {
		return []models.HostCredentials{}, nil
	}

	return credList.Host, nil
}

// WriteHostCredentials saves credentials to host_credentials.toml
func WriteHostCredentials(creds []models.HostCredentials) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	tabs, err := readTabConfigLocked()
	if err != nil {
		return err
	}

	if len(tabs) > 0 {
		hostTabMap := make(map[string]string)
		for _, tab := range tabs {
			listPath := resolveTabListPath(tab)
			if data, err := os.ReadFile(listPath); err == nil {
				var hl models.HostList
				if err := toml.Unmarshal(data, &hl); err == nil {
					for _, h := range hl.Host {
						hostTabMap[h.Hostname] = tab.Name
					}
				}
			}
		}

		tabCredsMap := make(map[string][]models.HostCredentials)
		for _, tab := range tabs {
			tabCredsMap[tab.Name] = []models.HostCredentials{}
		}

		defaultTab := tabs[0].Name
		for _, cr := range creds {
			t := cr.Tab
			if t == "" {
				if mappedTab, ok := hostTabMap[cr.Hostname]; ok {
					t = mappedTab
				} else {
					t = defaultTab
				}
			}
			if tabCredsMap[t] == nil {
				t = defaultTab
			}
			tabCredsMap[t] = append(tabCredsMap[t], cr)
		}

		for _, tab := range tabs {
			credPath := resolveTabCredPath(tab)
			if err := writeHostCredentialsTOML(credPath, tabCredsMap[tab.Name]); err != nil {
				return err
			}
		}
		return nil
	}

	return writeHostCredentialsTOML(credTomlFilePath, creds)
}

// WriteHostListToCsv writes hosts in CSV format to an io.Writer without ID
func WriteHostListToCsv(w io.Writer, hosts []models.Host) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"hostname", "ip", "platform", "os", "tags", "description", "updatedAt", "accesslist"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, host := range hosts {
		accessJson, _ := json.Marshal(host.Accesslist)
		record := []string{
			host.Hostname,
			host.IP,
			host.Platform,
			host.OS,
			host.Tags,
			host.Description,
			host.UpdatedAt,
			string(accessJson),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// Helper to write host list TOML
func writeHostListTOML(path string, hosts []models.Host) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	list := models.HostList{Host: hosts}
	data, err := toml.Marshal(list)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Helper to write credentials TOML
func writeHostCredentialsTOML(path string, creds []models.HostCredentials) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	list := models.HostCredentialsList{Host: creds}
	data, err := toml.Marshal(list)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
