package db

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
	dataDir          = "./data"
	tomlFilePath     = filepath.Join(dataDir, "hostlist.toml")
	credTomlFilePath = filepath.Join(dataDir, "host_credentials.toml")
	configFilePath   = filepath.Join(dataDir, "config.toml")

	dbMutex sync.RWMutex
)

func InitDatabase() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	// Seed hostlist.toml if it does not exist
	if _, err := os.Stat(tomlFilePath); os.IsNotExist(err) {
		if err := writeHostListTOML(tomlFilePath, hostListSeed); err != nil {
			return fmt.Errorf("failed to seed hostlist.toml: %w", err)
		}
		fmt.Printf("Host Database (TOML) seeded successfully with %d hosts (hostlist.toml)!\n", len(hostListSeed))
	}

	// Seed host_credentials.toml if it does not exist
	if _, err := os.Stat(credTomlFilePath); os.IsNotExist(err) {
		if err := writeHostCredentialsTOML(credTomlFilePath, hostUserCredSeed); err != nil {
			return fmt.Errorf("failed to seed host_credentials.toml: %w", err)
		}
		fmt.Printf("Host Credentials Database seeded successfully with %d credentials!\n", len(hostUserCredSeed))
	}

	// Seed config.toml if it does not exist
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		conf := models.Config{
			PermitIPList:   []string{"127.0.0.1"},
			MasterPassword: "password",
			AdminPassword:  "admin",
			UserPassword:   "user",
		}
		data, err := toml.Marshal(conf)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(configFilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config.toml: %w", err)
		}
		fmt.Println("Config file config.toml initialized with default IP permit list, masterpassword, admin_password, and user_password!")
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
		if len(hostList.Host[i].Accesslist) == 0 && hostList.Host[i].Port != "" {
			proto := DefaultProtocol(hostList.Host[i].Platform, hostList.Host[i].Port)
			hostList.Host[i].Accesslist = []models.AccessItem{
				{Protocol: proto, Port: hostList.Host[i].Port},
			}
		}
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
			Port:        portVal,
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

	if err := writeHostListTOML(tomlFilePath, hosts); err != nil {
		return err
	}

	return nil
}

// ReadHostCredentials loads credentials from host_credentials.toml
func ReadHostCredentials() ([]models.HostCredentials, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

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

	return writeHostCredentialsTOML(credTomlFilePath, creds)
}

// WriteHostListToCsv writes hosts in CSV format to an io.Writer without ID
func WriteHostListToCsv(w io.Writer, hosts []models.Host) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"hostname", "ip", "platform", "os", "port", "tags", "description", "updatedAt", "accesslist"}
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
			host.Port,
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
