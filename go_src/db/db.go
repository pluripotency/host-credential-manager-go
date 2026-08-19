package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"host-credential-manager-go/go_src/models"

	"github.com/pelletier/go-toml/v2"
)

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

// ReadHostList loads hosts from the TOML file
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

		host := models.Host{
			ID:          getVal("id"),
			Hostname:    getVal("hostname"),
			IP:          getVal("ip"),
			Platform:    getVal("platform"),
			Port:        getVal("port"),
			Tags:        getVal("tags"),
			Description: getVal("description"),
			UpdatedAt:   getVal("updatedAt"),
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

// WriteHostListToCsv writes hosts in CSV format to an io.Writer
func WriteHostListToCsv(w io.Writer, hosts []models.Host) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"id", "hostname", "ip", "platform", "port", "tags", "description", "updatedAt"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, host := range hosts {
		record := []string{
			host.ID,
			host.Hostname,
			host.IP,
			host.Platform,
			host.Port,
			host.Tags,
			host.Description,
			host.UpdatedAt,
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
