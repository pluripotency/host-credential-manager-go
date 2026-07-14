package models

type UserCredential struct {
	Username string `toml:"username" json:"username"`
	Password string `toml:"password" json:"password"`
}

type HostCredentials struct {
	Hostname string           `toml:"hostname" json:"hostname"`
	Userlist []UserCredential `toml:"userlist" json:"userlist"`
}

// HostCredentialsList は TOML 全体を表すためのラップ用構造体です
type HostCredentialsList struct {
	Host []HostCredentials `toml:"host"`
}

type Host struct {
	ID          string           `json:"id" csv:"id" toml:"id"`
	Hostname    string           `json:"hostname" csv:"hostname" toml:"hostname"`
	IP          string           `json:"ip" csv:"ip" toml:"ip"`
	Platform    string           `json:"platform" csv:"platform" toml:"platform"`
	Port        string           `json:"port" csv:"port" toml:"port"`
	Tags        string           `json:"tags" csv:"tags" toml:"tags"`
	Description string           `json:"description" csv:"description" toml:"description"`
	UpdatedAt   string           `json:"updatedAt" csv:"updatedAt" toml:"updatedAt"`
	Userlist    []UserCredential `json:"userlist" csv:"-" toml:"-"`
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
