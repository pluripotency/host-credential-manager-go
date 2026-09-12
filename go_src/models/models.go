package models

type UserCredential struct {
	Username string `toml:"username" json:"username"`
	Password string `toml:"password" json:"password"`
}

type HostCredentials struct {
	Hostname string           `toml:"hostname" json:"hostname"`
	Tab      string           `toml:"-" json:"tab,omitempty"`
	Userlist []UserCredential `toml:"userlist" json:"userlist"`
}

// HostCredentialsList は TOML 全体を表すためのラップ用構造体です
type HostCredentialsList struct {
	Host []HostCredentials `toml:"host"`
}

type TabItem struct {
	Name         string `toml:"name" json:"name"`
	DirPath      string `toml:"dirpath" json:"dirpath"`
	ListFilename string `toml:"list_filename" json:"list_filename"`
	CredFilename string `toml:"cred_filename" json:"cred_filename"`
}

type TabConfig struct {
	Tab []TabItem `toml:"tab" json:"tab"`
}

type AccessItem struct {
	Protocol string `json:"protocol" csv:"protocol" toml:"protocol"`
	Port     string `json:"port" csv:"port" toml:"port"`
	Path     string `json:"path,omitempty" csv:"path,omitempty" toml:"path,omitempty"`
}

type Host struct {
	ID          string           `json:"id" csv:"-" toml:"-"`
	Tab         string           `json:"tab,omitempty" csv:"tab,omitempty" toml:"tab,omitempty"`
	Hostname    string           `json:"hostname" csv:"hostname" toml:"hostname"`
	IP          string           `json:"ip" csv:"ip" toml:"ip"`
	Platform    string           `json:"platform" csv:"platform" toml:"platform"`
	OS          string           `json:"os" csv:"os" toml:"os"`
	Tags        string           `json:"tags" csv:"tags" toml:"tags"`
	Description string           `json:"description" csv:"description" toml:"description"`
	UpdatedAt   string           `json:"updatedAt" csv:"updatedAt" toml:"updatedAt"`
	Userlist    []UserCredential `json:"userlist" csv:"-" toml:"-"`
	Accesslist  []AccessItem     `json:"accesslist" csv:"-" toml:"accesslist"`
}

// HostList は TOML 全体を表すためのラップ用構造体です
type HostList struct {
	Host []Host `toml:"host"`
}

type Config struct {
	PermitIPList   []string `toml:"permit_ip_list" json:"permit_ip_list"`
	MasterPassword string   `toml:"masterpassword" json:"masterpassword"`
	AdminPassword  string   `toml:"admin_password" json:"admin_password"`
	UserPassword   string   `toml:"user_password" json:"user_password"`
}
