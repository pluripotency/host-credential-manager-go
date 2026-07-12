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
	csvFilePath      = filepath.Join(dataDir, "hostlist.csv")
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

	// Seed hostlist.csv if it does not exist
	if _, err := os.Stat(csvFilePath); os.IsNotExist(err) {
		if err := writeHostListCSV(csvFilePath, hostListSeed); err != nil {
			return fmt.Errorf("failed to seed hostlist.csv: %w", err)
		}
		fmt.Printf("Host Database seeded successfully with %d hosts (CSV)!\n", len(hostListSeed))
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
			PermitIPList: []string{"127.0.0.1"},
		}
		data, err := toml.Marshal(conf)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(configFilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config.toml: %w", err)
		}
		fmt.Println("Config file config.toml initialized with default IP permit list!")
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
		return conf, nil
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return conf, err
	}

	if err := toml.Unmarshal(data, &conf); err != nil {
		return conf, err
	}

	return conf, nil
}

// ReadHostList loads hosts from the CSV file
func ReadHostList() ([]models.Host, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	file, err := os.Open(csvFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Host{}, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
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

// WriteHostList saves hosts to both CSV and TOML files
func WriteHostList(hosts []models.Host) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if err := writeHostListCSV(csvFilePath, hosts); err != nil {
		return err
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

// Helper to write CSV
func writeHostListCSV(path string, hosts []models.Host) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
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
	list := models.HostList{Host: hosts}
	data, err := toml.Marshal(list)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Helper to write credentials TOML
func writeHostCredentialsTOML(path string, creds []models.HostCredentials) error {
	list := models.HostCredentialsList{Host: creds}
	data, err := toml.Marshal(list)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
